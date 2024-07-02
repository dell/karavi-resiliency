package monitor

import (
	"context"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
)

const failoverLabelKey = "failover.podmon.dellemc.com"
const replicationDefaultDomain = "replication.storage.dell.com"
const replicationGroupName = "/replicationGroupName"

type ReplicationGroupInfo struct {
	RGName           string              // ReplicationGroup Name
	PodKeysToPVNames map[string][]string // map[podkey][pvnames]
	pvnamesInRG      []string
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
		log.WithFields(fields).Infof("checkPendingPod Pod does not have the %s label so cannot consider DR", failoverLabelKey)
		return false
	}

	podkey := getPodKey(pod)
	log.WithFields(fields).Infof("checkPendingPod Reading PVs in pod %s", podkey)

	// Get all the PVCs in the pod.
	pvclist, err := K8sAPI.GetPersistentVolumeClaimsInPod(ctx, pod)
	if err != nil {
		log.WithFields(fields).Error("checkPendingPod Could not get PersistentVolumeClaims: %s", err)
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

	// Find all the PVs using this rg
	labelSelector := replicationDefaultDomain + replicationGroupName + "=" + rgName
	pvsInRG, err := K8sAPI.GetPersistentVolumesWithLabels(ctx, replicationDefaultDomain+replicationGroupName)
	if err != nil {
		log.Infof("Unable to read PVs using label %s: %s", labelSelector, err.Error())
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
	rgAny, ok := cm.RGNameToReplicationGroupInfo.Load()
	if !ok {
		rginfo = &ReplicationGroupInfo{
			RGName: rgName,
		}
	} else {
		rginfo = rgAny.(*ReplicationGroupInfo)
	}

	return true
}
