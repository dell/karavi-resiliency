package monitor

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	repv1 "github.com/dell/csm-replication/api/v1"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
)

const (
	failoverLabelKey             = "failover.podmon.dellemc.com"
	replicationGroupKey          = "replication-group.podmon.dellemc.com"
	replicationDefaultDomain     = "replication.storage.dell.com"
	replicationGroupName         = "/replicationGroupName"
	ActionFailoverRemote         = "FAILOVER_REMOTE"
	ActionFailoverLocalUnplanned = "UNPLANNED_FAILOVER_LOCAL"
	ActionReprotect              = "REPROTECT_LOCAL"
	SynchronizedState            = "SYNCHRONIZED"
	ReplicatedPrefix             = "replicated-"
)

// keeps multiple pods from updating ReplicationGroupInfo concurrently
var replicationGroupInfoMutex sync.Mutex

type ReplicationGroupInfo struct {
	RGName            string              // ReplicationGroup Name
	PodKeysToPVNames  map[string][]string // map[podkey][pvnames]
	pvNamesInRG       []string
	failoverMutex     sync.Mutex
	awaitingReprotect bool   // Have failed over - need replicationGroupInfoMutexd to reprotecg
	arrayID           string // arrayID of the array the PVs belong to
}

func logReplicationGroupInfo(place string, rginfo *ReplicationGroupInfo) {
	log.Infof("%s: RGINFO RGName %s\n pods %d pvs %d pvNamesInRG %v\n awaitingReprotect %t arrayID %s", place, rginfo.RGName, len(rginfo.PodKeysToPVNames), len(rginfo.pvNamesInRG), rginfo.pvNamesInRG, rginfo.awaitingReprotect, rginfo.arrayID)
}

// returns true if the pod has failover label
func podHasFailoverLabel(pod *v1.Pod) bool {
	return pod.Labels[failoverLabelKey] != ""
}

func (cm *PodMonitorType) checkReplicatedPod(ctx context.Context, pod *v1.Pod) bool {
	if !FeatureDisasterRecoveryActions {
		return false
	}
	fields := make(map[string]interface{})
	fields["namespace"] = pod.ObjectMeta.Namespace
	fields["pod"] = pod.ObjectMeta.Name
	fields["reason"] = "Pod in pending state"

	// Check that this pod has the failover annotation
	if !podHasFailoverLabel(pod) {
		log.WithFields(fields).Infof("checkReplicatedPod Pod does not have the %s label so cannot consider DR failover", failoverLabelKey)
		return false
	}
	// Check that pod is initialized
	ready, initialized, _ := podStatus(pod.Status.Conditions)
	if !initialized {
		log.WithFields(fields).Infof("checkReplicatedPod: Pod in wrong state: ready %t, initialized %t, pending %t, ready, initialized, pending")
		return false
	}

	podkey := getPodKey(pod)
	log.WithFields(fields).Infof("checkReplicatedPod Reading PVs in pod %s", podkey)

	// Get all the PVCs in the pod.
	pvclist, err := K8sAPI.GetPersistentVolumeClaimsInPod(ctx, pod)
	if err != nil {
		log.WithFields(fields).Errorf("checkReplicatedPod Could not get PersistentVolumeClaims: %s", err)
	}

	// Get the PVs associated with this pod.
	pvlist, err := K8sAPI.GetPersistentVolumesInPod(ctx, pod)
	if err != nil {
		log.WithFields(fields).Errorf("checkReplicatedPod Could not get PersistentVolumes: %s", err)
		return false
	}

	// Check that we have matching number of pvcs and pvs.
	if len(pvclist) != len(pvlist) {
		log.WithFields(fields).Errorf("checkReplicatedPod: mismatch counts pvcs %d pvs %d", len(pvclist), len(pvlist))
	}

	// Update the ControllerPodInfo to reflect the RG
	podInfoValue, ok := cm.PodKeyToControllerPodInfo.Load(podkey)
	if !ok {
		podInfoValue = &ControllerPodInfo{
			PodKey: podkey,
			PodUID: string(pod.ObjectMeta.UID),
		}
	}
	controllerPodInfo := podInfoValue.(*ControllerPodInfo)
	arrayIDs, _, err := cm.podToArrayIDs(ctx, pod)
	if err != nil {
		log.Errorf("checkReplicatedPod: Could not determine pod to arrayIDs: %s", err)
	} else {
		controllerPodInfo.ArrayIDs = arrayIDs
	}
	if len(arrayIDs) > 1 {
		log.Infof("checkReplicatedPod: Pod uses multiple array IDs, can't handle failover")
		return false
	}
	if len(arrayIDs) == 0 {
		log.Info("checkReplicatedPod: Pod does not use any array IDs, can't handle failover")
		return false
	}
	arrayID := arrayIDs[0]
	log.Infof("checkReplicatedPod: considering pod %s UID %s arrayID %s for replication", podkey, pod.UID, arrayID)

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
				log.Infof("checkReplicatedPod: multiple storage classes used: %s %s", storageClassName, pv.Spec.StorageClassName)
			}
			if name != rgName {
				log.WithFields(fields).Infof("checkReplicatedPod pv %s has a different replicationGroupName label than pv %s", pv.Name, firstPVName)
				return false
			}
		}
	}
	log.WithFields(fields).Infof("checkReplicatedPod: rgName %s", rgName)

	// Next read the replication group
	// Next find all the global PVs in the replication group (not just limited to this pod)
	rg, err := K8sAPI.GetReplicationGroup(ctx, rgName)
	if rg != nil {
		log.Infof("checkReplicatedPod: ReplicationGroup %+v", rg)
	}
	if err != nil || rg == nil {
		return false
	}

	// Make sure the RG is replicating to the same cluster
	if rg.Annotations["replication.storage.dell.com/remoteClusterID"] != "self" {
		log.Infof("checkReplicatedPod: ReplicationGroup %s is not a single-cluster replication configuration- podmon cannot manage the failover", rgName)
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
	controllerPodInfo.ReplicationGroup = rgName
	cm.PodKeyToControllerPodInfo.Store(podkey, controllerPodInfo)

	// See if there is ane existing ReplicationGroupInfo
	replicationGroupInfoMutex.Lock()
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
	rginfo.arrayID = arrayID
	cm.RGNameToReplicationGroupInfo.Store(rgName, rginfo)
	replicationGroupInfoMutex.Unlock()
	//log.Infof("stored ReplicationGroupInfo %+v", rginfo)
	logReplicationGroupInfo("stored ReplicationGroupInfo: "+podkey, rginfo)

	// Add replication-group label to the pod.
	replacedLabels := make(map[string]string)
	deletedLabels := make([]string, 0)
	replacedLabels[replicationGroupKey] = rgName
	err = K8sAPI.PatchPodLabels(ctx, pod.Namespace, pod.Name, replacedLabels, deletedLabels)
	if err != nil {
		log.Errorf("PatchPodLabels for pod %s failed: %s", podkey, err.Error())
	}

	// If pod is ready don't trigger a failover.
	if ready {
		log.Infof("checkReplicatedPod: pod %s is ready", pod.Name)
		return false
	}

	// Determine if there is a problem with the array.
	nodeList := make([]string, 0)
	nodeConnectivity := cm.getNodeConnectivityMap(arrayID)
	log.Infof("checkReplicatedPod: NodeConnectivityMap for array %s %v", arrayID, nodeConnectivity)
	var hasConnection bool
	var connectionEntries int
	for key, value := range nodeConnectivity {
		nodeList = append(nodeList, key)
		connectionEntries++
		if value {
			log.Infof("checkReplicatedPod: not initiating failover because array %s connected to node %s", arrayID, key)
			hasConnection = true
		}
	}
	if connectionEntries == 0 || hasConnection {
		log.Infof("checkReplicatedPod: not initiating failover connectionEntries %d hasConnection %t", connectionEntries, hasConnection)
		return false
	}

	cm.tryFailover(rgName, nodeList)
	return true
}

func (cm *PodMonitorType) addPodToReplicationGroup(ctx context.Context, pod *v1.Pod) error {
}

func (cm *PodMonitorType) tryFailover(rgName string, nodeList []string) bool {
	var rgTarget string
	taintedNodes := make(map[string]string)

	// toReplicated means we are failing over to the replicated PVs
	var toReplicated bool
	if strings.HasPrefix(rgName, ReplicatedPrefix) {
		rgTarget = rgName[11:]
	} else {
		rgTarget = ReplicatedPrefix + rgName
		toReplicated = true
	}
	ctx, cancel := context.WithTimeout(context.Background(), FailoverTimeout)
	defer cancel()
	var rginfo *ReplicationGroupInfo
	replicationGroupInfoMutex.Lock()
	defer replicationGroupInfoMutex.Unlock()
	rgAny, ok := cm.RGNameToReplicationGroupInfo.Load(rgName)
	if !ok {
		log.Infof("tryFailover: ReplicationGroupInfo not found: %s", rgName)
		return false
	}
	rginfo = rgAny.(*ReplicationGroupInfo)
	// log.Infof("tryFailover rginfo %+v", rginfo)
	logReplicationGroupInfo("tryFailover", rginfo)

	// Make sure there's only one failover attempt at a time.
	if !rginfo.failoverMutex.TryLock() {
		log.Infof("tryFailover: RG %s has a failover already in progress - not starting another", rgName)
		return false
	}
	defer rginfo.failoverMutex.Unlock()

	// Check the the replication groups link state is SYNCHRONIZED
	name1 := rgName
	if strings.HasPrefix(name1, ReplicatedPrefix) {
		name1 = rgName[len(ReplicatedPrefix):]
	}
	rg1, err := K8sAPI.GetReplicationGroup(ctx, name1)
	if err != nil {
		log.Errorf("tryFailover: Couldn't read source RG1 %s: %s", name1, err.Error())
		return false
	}
	linkState := rg1.Status.ReplicationLinkState.State
	if linkState != SynchronizedState {
		log.Errorf("tryFailover: RG1 %s link state not %s", name1, SynchronizedState)
		return false
	}
	name2 := ReplicatedPrefix + name1
	rg2, err := K8sAPI.GetReplicationGroup(ctx, name2)
	if err != nil {
		log.Errorf("tryFailover: Couldn't read source RG2 %s: %s", name2, err.Error())
		return false
	}
	linkState = rg2.Status.ReplicationLinkState.State
	if linkState != SynchronizedState {
		log.Errorf("tryFailover: RG2 %s link state not %s", name2, SynchronizedState)
		return false
	}

	// Loop through the nodes tainting them
	failoverTaint := getFailoverTaint(rgName)
	for _, nodeName := range nodeList {
		tainterr := taintNode(nodeName, failoverTaint, false)
		if tainterr == nil {
			log.Infof("tryFailover: tained node %s", nodeName)
			taintedNodes[nodeName] = nodeName
		}
	}

	// Ket PVC names for each pod
	podKeyToTargetPVCNames := make(map[string][]string)
	var podKeyToTargetPVCNamesMutex sync.Mutex
	for podkey, _ := range rginfo.PodKeysToPVNames {
		func(podkey string) []string {
			pvcnames := make([]string, 0)
			namespace, name := splitPodKey(podkey)
			pod, err := K8sAPI.GetPod(ctx, namespace, name)
			if err != nil {
				log.Errorf("tryFailover couldn't read pod: %s: %s", podkey, err.Error())
				return pvcnames
			}
			for _, vol := range pod.Spec.Volumes {
				if vol.VolumeSource.PersistentVolumeClaim != nil {
					pvcname := vol.VolumeSource.PersistentVolumeClaim.ClaimName
					pvcnames = append(pvcnames, pvcname)
				}
			}
			log.Debugf("tryFailover: podkey %s pvcnames %v", podkey, pvcnames)
			podKeyToTargetPVCNamesMutex.Lock()
			podKeyToTargetPVCNames[podkey] = pvcnames
			podKeyToTargetPVCNamesMutex.Unlock()
			return pvcnames
		}(podkey)
	}

	// Loop through the pods to see if they're in pending or creating state
	nf1pods := len(rginfo.PodKeysToPVNames)
	f1chan := make(chan bool, nf1pods)
	f1 := func(podkey string) bool {
		var done bool
		for i := 0; i < 10; i++ {
			if done = forceDeletePod(ctx, podkey); done {
				break
			}
		}
		time.Sleep(5 * time.Second)
		for i := 0; i < 10; i++ {
			if done = forceDeletePod(ctx, podkey); done {
				break
			}
		}
		f1chan <- done
		return done
	}

	npods := 0
	for podkey, _ := range rginfo.PodKeysToPVNames {
		go f1(podkey)
		npods++
	}

	var ncomplete, nincomplete int
	for i := 0; i < nf1pods; i++ {
		done := <-f1chan
		if done {
			ncomplete++
		} else {
			nincomplete++
		}
	}
	log.Infof("tryFailover: forceDeletePod pods %d complete %d incomplete %d", nf1pods, ncomplete, nincomplete)

	// Read the volumes in the Replication Group
	// pvsInRG, err := getPVsInReplicationGroup(ctx, rgName)
	// if err != nil {
	// 	return false
	// }
	// // // We only want to count the regular PVs or the replicated PVs, whichever is higher.
	// var rgpvscount, rgreplicatedpvscount int
	// for _, pv := range pvsInRG.Items {
	// 	if strings.HasPrefix(pv.Name, ReplicatedPrefix) {
	// 		rgreplicatedpvscount++
	// 	} else {
	// 		rgpvscount++
	// 	}
	// }
	// if rgreplicatedpvscount > rgpvscount {
	// 	rgpvscount = rgreplicatedpvscount
	// }
	// If the number of PVs in the replication group not equal to the number
	// of PVs we know about from the pods, we cannot fail over.
	// if rgpvscount != npvsinpods {
	// 	log.Infof("tryFailover; aborting: RG has %d unique PVs, all %d pods have total %d PVs, cannot failover because they're not equal", rgpvscount, npods, npvsinpods)
	// 	return false
	// }
	log.Infof("ATTEMPTING FAILOVER REPLICATION GROUP %s target %s pods %d pvs %d", rgName, rgTarget, npods)
	unplanned := true
	rgToLoad := rgName
	if unplanned {
		rgToLoad = rgTarget
	}
	rg, err := K8sAPI.GetReplicationGroup(ctx, rgToLoad)
	if err != nil || rg == nil {
		if err == nil {
			err = errors.New("not found")
		}
		log.Errorf("tryFailover: aborting: Could not get ReplicationGrop %s: %s", rgName, err)
		untaintNodes(taintedNodes, failoverTaint)
		return false
	}

	startingTime := rg.Status.ReplicationLinkState.LastSuccessfulUpdate.Time
	if unplanned {
		rg.Spec.Action = ActionFailoverLocalUnplanned
		log.Infof("Updating RG %s with action %s", rg.Name, rg.Spec.Action)
		err := K8sAPI.UpdateReplicationGroup(ctx, rg)
		if err != nil {
			log.Errorf("tryFailover: aborting: Error updating RG %s with action %s: %s", rg.Name, rg.Spec.Action, err.Error())
			untaintNodes(taintedNodes, failoverTaint)
			return false
		}
		// rg = waitForRGStateToUpdate(ctx, rg.Name, "FAILOVER")
		// log.Infof("Unplanned Failover completed RG %s:\n Last Action %+v\n Last Condition %+v\nLink State %+v", rgName, rg.Status.LastAction, rg.Status.Conditions[0], rg.Status.ReplicationLinkState.State)
	} else {
		rg.Spec.Action = ActionFailoverRemote
		log.Infof("Updating RG %s with action %s", rg.Name, rg.Spec.Action)
		err := K8sAPI.UpdateReplicationGroup(ctx, rg)
		if err != nil {
			log.Errorf("tryFailover: aborting: Error updating RG %s with action %s: %s", rg.Name, rg.Spec.Action, err.Error())
			untaintNodes(taintedNodes, failoverTaint)
			return false
		}
	}

	// Set up a channel to indicate each pod has completed or timedout.
	npods = len(rginfo.PodKeysToPVNames)
	donechan := make(chan bool, npods)

	// Kill the pods again, hoping each container will restart
	// Loop through the pods to see if they're in pending or creating state
	var failoverTimeout bool
	f2 := func(podkey string) bool {
		var done bool
		for i := 0; i < 50; i++ {
			_, _, done = killAPodToRemapPVCs(ctx, podkey, podKeyToTargetPVCNames, toReplicated)
			if done {
				log.Infof("tryFailover: pod %s done", podkey)
				donechan <- done
				return done
			}
			time.Sleep(5 * time.Second)
		}
		log.Infof("tryFailover: pod %s timed out", podkey)
		failoverTimeout = true
		donechan <- done
		return done
	}

	for podkey, _ := range rginfo.PodKeysToPVNames {
		go f2(podkey)
	}
	// Waiting on the pods to be done
	for i := 0; i < npods; i++ {
		log.Debugf("waiting on donechan %d", i)
		done := <-donechan
		log.Debugf("done %d %t", i, done)
	}

	// The failover will only complete after the PVs are remapped?
	rg = waitForRGStateToUpdate(ctx, rgTarget, "UNPLANNED_FAILOVER_LOCAL")
	if rg != nil {
		log.Infof("Unplanned Failover completed RG %s:\n Last Action %+v\n Last Condition %+v\nLink State %+v", rgName, rg.Status.LastAction, rg.Status.Conditions[0], rg.Status.ReplicationLinkState.State)
	}
	if failoverTimeout {
		message := fmt.Sprintf("failover timeout: %v", time.Now().Sub(startingTime))
		logReplicationGroupInfo(message, rginfo)
	}
	log.Infof("failover time: %v", time.Now().Sub(startingTime))

	// untaint the nodes allow pods to be created
	untaintNodes(taintedNodes, failoverTaint)

	// Set awaitingReprotect on the RGInfo and save it.
	rginfo.awaitingReprotect = true
	cm.RGNameToReplicationGroupInfo.Store(rginfo.RGName, rginfo)

	// Create an event for the replication
	// // This code will not work on the RG custom resource without an event recorder
	// eventRG := rg1
	// if rg1.Name != rgName {
	// 	eventRG = rg2
	// }
	// if err = K8sAPI.CreateEvent(podmon, eventRG, k8sapi.EventTypeWarning, "Replication failover",
	// 	"replication failed over from RG %s to %s", rgName, rgTarget); err != nil {
	// 	log.Errorf("Error reporting replication failover event: %s %s: %s", rgName, rgTarget, err.Error())
	// }

	return true
}

// Returns pod.Spec.NodeName, bool indicating all PVCs remapped or not
// Returns the pod's pod.Spec.NodeName, number of pvcs the pod has, and a bool indicating done or not.
// This code is looking to see if the PVC remap login in dell-replication-controller has finished for the pod.
// We try hard here to make the minimum number of K8S API calls.
func killAPodToRemapPVCs(ctx context.Context, podkey string, podKeyToTargetPVCNames map[string][]string, toReplicated bool) (string, int, bool) {
	var reason string
	namespace, name := splitPodKey(podkey)
	// Get the PVCs in the pod. The list of pvc names was precalculated and passed in in podKeyToTargetPVCNames.
	// Get all the PVCs in the pod.
	targetPvcList := podKeyToTargetPVCNames[podkey]
	done := true
	for _, pvcName := range targetPvcList {
		pvc, _ := K8sAPI.GetPersistentVolumeClaim(ctx, namespace, pvcName)
		if pvc == nil {
			reason = fmt.Sprintf("pvc %s nil", pvcName)
			done = false
		} else {
			if pvc.Spec.VolumeName == "" {
				reason = fmt.Sprintf("pvc %s empty VolumeName %s", pvcName, pvc.Spec.VolumeName)
				done = false
			}
			if toReplicated && !strings.HasPrefix(pvc.Spec.VolumeName, ReplicatedPrefix) {
				reason = fmt.Sprintf("pvc %s toReplicated %t pv name %s", pvcName, toReplicated, pvc.Spec.VolumeName)
				done = false
			}
			if !toReplicated && strings.HasPrefix(pvc.Spec.VolumeName, ReplicatedPrefix) {
				reason = fmt.Sprintf("pvc %s toReplicated %t pv name %s", pvcName, toReplicated, pvc.Spec.VolumeName)
				done = false
			}
		}
	}
	if done {
		log.Infof("killAPodToRemapPVCs: pod %s all pvcs remapped", podkey)
		return "", len(targetPvcList), done
	} else {
		log.Infof("killAPodToRemapPVCs not done reason: %s", reason)
	}

	pod, err := K8sAPI.GetPod(ctx, namespace, name)
	if err != nil && !strings.Contains(err.Error(), "not found") {
		log.Infof("killAPodToRmapPVCs: couldn't load pod %s: %s", podkey, err.Error())
	}

	if pod != nil {
		ready, initialized, pending := podStatus(pod.Status.Conditions)
		log.Debugf("killAPodToRemapPVCs: pod %s ready %t initialized %t pending %t", podkey, ready, initialized, pending)

		// Kill the pod
		if initialized && !pending {
			log.Infof("killAPodToRemapPVCs: force deleting pod %s", podkey)
			K8sAPI.DeletePod(ctx, pod.Namespace, pod.Name, pod.UID, true)
		}
		return pod.Spec.NodeName, len(targetPvcList), false
	}
	return "", len(targetPvcList), done
}

// forceDeleteAPod force deletes a pod, returns done
func forceDeletePod(ctx context.Context, podkey string) bool {
	namespace, name := splitPodKey(podkey)
	pod, err := K8sAPI.GetPod(ctx, namespace, name)
	if err != nil {
		log.Infof("forceDeletePod: couldn't load pod %s: %s", podkey, err.Error())
		// Return done if the pod wasn't found, because in the second pass, there are taints and the pods
		// cannot be restarted until the replication command finished.
		done := strings.Contains(err.Error(), "not found")
		return done
	}
	log.Infof("forceDeletePod: force deleting pod %s", podkey)
	err = K8sAPI.DeletePod(ctx, pod.Namespace, pod.Name, pod.UID, true)
	if err != nil {
		log.Errorf("forceDeletePod: error deleting pod: %s", err.Error())
	}
	return err == nil
}

// untaintNodes untaints s list of nodes.
func untaintNodes(taintedNodeNames map[string]string, failoverTaint string) {
	for tainted, _ := range taintedNodeNames {
		log.Infof("tryFailover: removing taint %s", tainted)
		go taintNode(tainted, failoverTaint, true)
	}
}

func getFailoverTaint(rgName string) string {
	if strings.HasPrefix(rgName, ReplicatedPrefix) {
		rgName = rgName[len(ReplicatedPrefix):]
	}
	parts := strings.Split(rgName, "-")
	return "failover." + parts[0] + "-" + parts[1] + "-" + parts[2] + "-" + parts[3]
}

func waitForRGStateToUpdate(ctx context.Context, rgName string, condition string) *repv1.DellCSIReplicationGroup {
	var rg *repv1.DellCSIReplicationGroup
	for i := 0; i < 10; i++ {
		time.Sleep(3 * time.Second)
		rg, err := K8sAPI.GetReplicationGroup(ctx, rgName)
		if err != nil {
			log.Errorf("Error reaading RG %s: %s", rgName, err.Error())
			continue
		}
		log.Infof("waitForRGStateToUpdate: %d %s:", i, rg.Status.LastAction.Condition)
		if strings.Contains(rg.Status.LastAction.Condition, condition) {
			return rg
		}
	}
	log.Infof("Timed out waiting for RG %s action %s to complete", rgName, condition)
	return rg
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

func (cm *PodMonitorType) checkFailover(arrayID string) {
	f := func(key any, value any) bool {
		rgName := key.(string)
		rginfo := value.(*ReplicationGroupInfo)
		if rginfo.awaitingReprotect {
			// if awaitingReprotect we have already failed over
			return true
		}
		if rginfo.arrayID == arrayID {
			logReplicationGroupInfo("checkFailover called", rginfo)
			// Determine if there is a problem with the array.
			nodeList := make([]string, 0)
			nodeConnectivity := cm.getNodeConnectivityMap(arrayID)
			log.Infof("checkReplicatedPod: NodeConnectivityMap for array %s %v", arrayID, nodeConnectivity)
			var hasConnection bool
			for key, value := range nodeConnectivity {
				nodeList = append(nodeList, key)
				if value {
					log.Infof("checkReplicatedPod: not initiating failover because array %s connected to node %s", arrayID, key)
					hasConnection = true
				}
			}
			if !hasConnection {
				log.Infof("checkFailover: initiating failover rg %s", rgName)
				cm.tryFailover(rgName, nodeList)
			} else {
				log.Infof("checkFailover: array %s had connection", arrayID)
			}
		}
		return true
	}
	cm.RGNameToReplicationGroupInfo.Range(f)
}

// Check Reprotect scans all the saved RGs calling checkReprotectReplicationGroup.
func (cm *PodMonitorType) checkReprotect(arrayID string) {
	log.Infof("checkReprotect called for arrayID: %s", arrayID)
	cm.RGNameToReplicationGroupInfo.Range(cm.reprotectReplicationGroup)
}

// checkReprotect looks for the boolean awaitingReprotect on RegplicationGroup
// info structure. This boolean is set at the completion of the failover process
// on the ReplicationGroupInfo that was failed over from.
// So the reprotect needs to be initiated on the paired replication group.
// This function must always return true.
func (cm *PodMonitorType) reprotectReplicationGroup(key any, value any) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	rgName := key.(string)
	rginfo := value.(*ReplicationGroupInfo)
	if !rginfo.awaitingReprotect {
		log.Infof("RG %s not awaitingReprotect", rgName)
		return true
	}
	// Check that the array that reconnected matches our rginfo
	nodeConnectivity := cm.getNodeConnectivityMap(rginfo.arrayID)
	var connected bool
	for _, v := range nodeConnectivity {
		if v {
			connected = true
		}
	}
	if !connected {
		return true
	}

	// Clear the awaitingReprotect flag, we get one shot do do it
	rginfo.awaitingReprotect = false

	// Remove the volume attachments
	go cm.removeVolumeAttachmentsForPVs(ctx, rginfo)

	cm.RGNameToReplicationGroupInfo.Store(rgName, rginfo)
	rgPairName := getRGPairName(rgName)
	// Initiate a reprotect
	log.Infof("reprotectReplicationGroup: Attempting to initiate a reprotect on RG %s", rgPairName)

	// Read the replication group
	rg, err := K8sAPI.GetReplicationGroup(ctx, rgPairName)
	if err != nil || rg == nil {
		if err == nil {
			err = errors.New("not found")
		}
		log.Errorf("reprotectReplicationGroup: Could not get ReplicationGrop %s: %s", rgName, err)
		return false
	}
	// Set the ActionReprotect
	rg.Spec.Action = ActionReprotect

	// Update the Replication Group
	log.Infof("reprotectReplicationGroup: Updating RG %s with action %s", rg.Name, rg.Spec.Action)
	err = K8sAPI.UpdateReplicationGroup(ctx, rg)
	if err != nil {
		log.Errorf("reprotectReplicationGroup: error updating RG %s: %s", rg.Name, err.Error())
		return true
	}

	// Wait on the action to update
	rg = waitForRGStateToUpdate(ctx, rg.Name, "REPROTECT")
	log.Infof("reprotectReplicationGroup %s ended with action: %s", rgName, rg.Status.LastAction.Condition)

	// Must always return true
	return true
}

func (cm *PodMonitorType) removeVolumeAttachmentsForPVs(ctx context.Context, rginfo *ReplicationGroupInfo) {
	hasReplicatedPrefix := strings.HasPrefix(rginfo.RGName, ReplicatedPrefix)
	pvnames := make([]string, 0)
	for _, pvname := range rginfo.pvNamesInRG {
		hasPrefix := strings.HasPrefix(pvname, ReplicatedPrefix)
		if hasReplicatedPrefix == hasPrefix {
			pvnames = append(pvnames, pvname)
		}
	}
	log.Infof("removing the following volumeattachments for rg %s: %v", rginfo.RGName, pvnames)
	volumeAttachmentsList, err := K8sAPI.GetVolumeAttachments(ctx)
	if err != nil {
		log.Errorf("Could not read VolumeAttachments for cleanup during replication reprotect")
		return
	}
	for _, va := range volumeAttachmentsList.Items {
		pvName := *va.Spec.Source.PersistentVolumeName
		if !stringInSlice(pvName, pvnames) {
			continue
		}
		err = K8sAPI.DeleteVolumeAttachment(ctx, va.Name)
		if err != nil {
			log.Errorf("Error deleting volumeattachment %s: %s", va.Name, err.Error())
		} else {
			log.Infof("Deleted volumeattachment %s for replicated volume", va.Name)
		}
	}
}

// getRGPairName takes an RG name and returns the name of the paired RG.
func getRGPairName(name string) string {
	if strings.HasPrefix(name, ReplicatedPrefix) {
		return name[len(ReplicatedPrefix):]
	}
	return ReplicatedPrefix + name
}
