package monitor

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	repv1 "github.com/dell/csm-replication/api/v1"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
)

const (
	failoverLabelKey             = "failover.podmon.dellemc.com"
	replicationDefaultDomain     = "replication.storage.dell.com"
	replicationGroupName         = "/replicationGroupName"
	ActionFailoverRemote         = "FAILOVER_REMOTE"
	ActionFailoverLocalUnplanned = "UNPLANNED_FAILOVER_LOCAL"
	ActionReprotect              = "REPROTECT_LOCAL"
)

type ReplicationGroupInfo struct {
	RGName           string              // ReplicationGroup Name
	PodKeysToPVNames map[string][]string // map[podkey][pvnames]
	pvNamesInRG      []string
	failoverMutex    sync.Mutex
}

func (cm *PodMonitorType) checkPendingPod(ctx context.Context, pod *v1.Pod) bool {
	if !FeatureDisasterRecoveryActions {
		return false
	}

	fields := make(map[string]interface{})
	fields["namespace"] = pod.ObjectMeta.Namespace
	fields["pod"] = pod.ObjectMeta.Name
	fields["reason"] = "Pod in pending state"

	// Check that this pod has the failover annotation
	if pod.Labels[failoverLabelKey] == "" {
		log.WithFields(fields).Infof("checkPendingPod Pod does not have the %s label so cannot consider DR failover", failoverLabelKey)
		return false
	}

	podkey := getPodKey(pod)
	log.WithFields(fields).Infof("checkPendingPod Reading PVs in pod %s", podkey)

	// Get all the PVCs in the pod.
	pvclist, err := K8sAPI.GetPersistentVolumeClaimsInPod(ctx, pod)
	if err != nil {
		log.WithFields(fields).Errorf("checkPendingPod Could not get PersistentVolumeClaims: %s", err)
	}

	// Get the PVs associated with this pod.
	pvlist, err := K8sAPI.GetPersistentVolumesInPod(ctx, pod)
	if err != nil {
		log.WithFields(fields).Errorf("checkPendingPod Could not get PersistentVolumes: %s", err)
		return false
	}

	// Check that we have matching number of pvcs and pvs.
	if len(pvclist) != len(pvlist) {
		log.WithFields(fields).Errorf("mismatch counts pvcs %d pvs %d", len(pvclist), len(pvlist))
	}

	// Update the ControllerPodInfo to reflect the RG
	podInfoValue, ok := cm.PodKeyToControllerPodInfo.Load(podkey)
	if !ok {
		arrayIDs, _, err := cm.podToArrayIDs(ctx, pod)
		if err != nil {
			log.Errorf("Could not determine pod to arrayIDs: %s", err)
		}
		podInfoValue = &ControllerPodInfo{
			PodKey:   podkey,
			PodUID:   string(pod.ObjectMeta.UID),
			ArrayIDs: arrayIDs,
		}
	}

	rgName := ""
	storageClassName := ""
	firstPVName := ""
	replicatedPVNames := make([]string, 0)

	// Loop through the PV getting the
	for _, pv := range pvlist {
		name := pv.Labels[replicationDefaultDomain+replicationGroupName]
		if name == "" {
			// Thsi volume not replicated, that's ok
			continue
		} else if rgName == "" {
			rgName = name
			firstPVName = pv.Name
			storageClassName = pv.Spec.StorageClassName
			replicatedPVNames = append(replicatedPVNames, pv.Name)
		} else {
			if pv.Spec.StorageClassName != storageClassName {
				log.Infof("multiple storage classes used: %s %s", storageClassName, pv.Spec.StorageClassName)
			}
			if name != rgName {
				log.WithFields(fields).Infof("checkPendingPod pv %s has a different replicationGroupName label than pv %s", pv.Name, firstPVName)
				return false
			}
		}
	}
	log.WithFields(fields).Infof("checkPendingPod: rgName %s", rgName)

	// Next read the replication group
	// Next find all the global PVs in the replication group (not just limited to this pod)
	rg, err := K8sAPI.GetReplicationGroup(ctx, rgName)
	if rg != nil {
		log.Infof("ReplicationGroup %+v", rg)
	}
	if err != nil || rg == nil {
		return false
	}

	// Make sure the RG is replicating to the same cluster
	if rg.Annotations["replication.storage.dell.com/remoteClusterID"] != "self" {
		log.Infof("ReplicationGroup %s is not a single-cluster replication configuration- podmon cannot manage the failover", rgName)
		return false
	}

	// Find all the PVs using this rg
	pvsInRG, err := getPVsInReplicationGroup(ctx, rgName)
	if err != nil {
		return false
	}
	pvnamesInRG := make([]string, 0)
	for _, pv := range pvsInRG.Items {
		pvnamesInRG = append(pvnamesInRG, pv.Name)
	}
	log.Infof("PVs in RG: %v", pvnamesInRG)

	// Update the controller pod info structure so ready for failover
	controllerPodInfo := podInfoValue.(*ControllerPodInfo)
	controllerPodInfo.ReplicationGroup = rgName
	cm.PodKeyToControllerPodInfo.Store(podkey, controllerPodInfo)
	log.Infof("Pod %s awaiting failover of ReplicationGroup %s", podkey, rgName)

	// See if there is an existing ReplicationGroupInfo
	var rginfo *ReplicationGroupInfo
	rgAny, ok := cm.RGNameToReplicationGroupInfo.Load(rgName)
	if !ok {
		rginfo = &ReplicationGroupInfo{
			RGName:           rgName,
			PodKeysToPVNames: make(map[string][]string),
		}
	} else {
		rginfo = rgAny.(*ReplicationGroupInfo)
	}
	rginfo.PodKeysToPVNames[podkey] = replicatedPVNames
	rginfo.pvNamesInRG = pvnamesInRG
	cm.RGNameToReplicationGroupInfo.Store(rgName, rginfo)
	log.Infof("stored ReplicationGroupInfo %+v", rginfo)

	cm.tryFailover((rgName))
	return true
}

func (cm *PodMonitorType) tryFailover(rgName string) bool {
	var rgTarget string

	if strings.HasPrefix(rgName, "replicated-") {
		rgTarget = rgName[11:]
	} else {
		rgTarget = "replicated-" + rgName
	}
	ctx, cancel := context.WithTimeout(context.Background(), MediumTimeout)
	defer cancel()
	var rginfo *ReplicationGroupInfo
	rgAny, ok := cm.RGNameToReplicationGroupInfo.Load(rgName)
	if !ok {
		log.Infof("ReplicationGroupInfo not found: %s", rgName)
		return false
	}
	rginfo = rgAny.(*ReplicationGroupInfo)
	log.Infof("tryFailover rginfo %+v", rginfo)

	// Make sure there's only one failover attempt at a time.
	if !rginfo.failoverMutex.TryLock() {
		log.Infof("RG %s has a failover already in progress - not starting another", rgName)
		return false
	}
	defer rginfo.failoverMutex.Unlock()

	// Loop through the pods to see if they're in pending state
	var npods, npending int
	var npvsinpods int
	for podkey, pvnames := range rginfo.PodKeysToPVNames {
		namespace, name := splitPodKey(podkey)
		pod, err := K8sAPI.GetPod(ctx, namespace, name)
		if err != nil {
			log.Infof("tryFailover: couldn't load pod %s: %s", podkey, err.Error())
			delete(rginfo.PodKeysToPVNames, podkey)
			continue
		}
		npods++
		ready, initialized, pending := podStatus(pod.Status.Conditions)
		log.Infof("tryFailover: pod %s ready %t initialized %t pending %t", podkey, ready, initialized, pending)
		if pending {
			npending++
		}
		npvsinpods = npvsinpods + len(pvnames)
	}
	cm.RGNameToReplicationGroupInfo.Store(rgName, rginfo)

	if npending != npods {
		log.Infof("tryFailover: only %d pods are in pending state out of %d pods, cannot failover", npending, npods)
		return false
	}
	// Read the volumes in the Replication Group
	pvsInRG, err := getPVsInReplicationGroup(ctx, rgName)
	if err != nil {
		return false
	}
	// We only want to count the regular PVs or the replicated PVs, whichever is higher.
	var rgpvscount, rgreplicatedpvscount int
	for _, pv := range pvsInRG.Items {
		if strings.HasPrefix(pv.Name, "replicated-") {
			rgreplicatedpvscount++
		} else {
			rgpvscount++
		}
	}
	if rgreplicatedpvscount > rgpvscount {
		rgpvscount = rgreplicatedpvscount
	}
	// If the number of PVs in the replication group not equal to the number
	// of PVs we know about from the pods, we cannot fail over.
	if rgpvscount != npvsinpods {
		log.Infof("RG has %d unique PVs, all %d pods have total %d PVs, cannot failover because they're not equal", rgpvscount, npods, npvsinpods)
		return false
	}
	log.Infof("ATTEMPTING FAILOVER REPLICATION GROUP %s target %s pods %d pvs %d", rgName, rgTarget, npods, npvsinpods)
	rg, err := K8sAPI.GetReplicationGroup(ctx, rgName)
	if err != nil || rg == nil {
		if err == nil {
			err = errors.New("not found")
		}
		log.Errorf("Could not get ReplicationGrop %s: %s", rgName, err)
		return false
	}
	startingTime := rg.Status.ReplicationLinkState.LastSuccessfulUpdate.Time

	unplanned := false
	if unplanned {
		log.Info("Unplanned failover not implemented yet")
		return false

	} else {
		rg.Spec.Action = ActionFailoverRemote
		log.Infof("Updating RG %s with action %s", rg.Name, rg.Spec.Action)
		err := K8sAPI.UpdateReplicationGroup(ctx, rg)
		if err != nil {
			log.Errorf("Error updating RG %s with action %s: %s", rg.Name, rg.Spec.Action, err.Error())
			return false
		}
		rg = waitForRGStateToUpdate(ctx, rgName, startingTime)
		if rg == nil {
			return false
		}
		log.Infof("Failover completed RG %s:\n Last Action %+v\n Last Condition %+v\nLink State %+v", rgName, rg.Status.LastAction, rg.Status.Conditions[0], rg.Status.ReplicationLinkState.State)
	}
	return true
}

func waitForRGStateToUpdate(ctx context.Context, rgName string, startingTime time.Time) *repv1.DellCSIReplicationGroup {
	for i := 0; i < 40; i++ {
		rg, err := K8sAPI.GetReplicationGroup(ctx, rgName)
		if err != nil {
			log.Errorf("Error reaading RG %s: %s", rgName, err.Error())
		}
		if rg.Status.ReplicationLinkState.LastSuccessfulUpdate.Time != startingTime {
			return rg
		}
		time.Sleep(5)
	}
	log.Infof("Timed out waiting for RG %s action to complete", rgName)
	// timed out
	return nil
}

func getPVsInReplicationGroup(ctx context.Context, rgName string) (*v1.PersistentVolumeList, error) {
	labelSelector := replicationDefaultDomain + replicationGroupName + "=" + rgName
	pvsInRG, err := K8sAPI.GetPersistentVolumesWithLabels(ctx, replicationDefaultDomain+replicationGroupName)
	if err != nil {
		log.Infof("Unable to read PVs using label %s: %s", labelSelector, err.Error())
		return nil, err
	}
	return pvsInRG, nil
}
