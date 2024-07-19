/*
* Copyright (c) 2021-2023 Dell Inc., or its subsidiaries. All Rights Reserved.
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
* http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
 */

package monitor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"podmon/internal/k8sapi"
	"strings"
	"sync"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	csiext "github.com/dell/dell-csi-extensions/podmon"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/watch"
)

// MaxCrashLoopBackOffRetry is the maximum number of times for a pod to be deleted in response to a CrashLoopBackOff
const MaxCrashLoopBackOffRetry = 5

// ControllerPodInfo has information for tracking health of the system
type ControllerPodInfo struct { // information controller keeps on hand about a pod
	PodKey            string            // the Pod Key (namespace/name) of the pod
	Node              *v1.Node          // the associated node structure
	PodUID            string            // the pod container's UID
	ArrayIDs          []string          // string of array IDs used by the pod's volumes
	PodAffinityLabels map[string]string // A list of pod affinity labels for the pod

	// The following fields used by disaster recovery if enabled.
	ReplicationGroup   string   // pod's association to a ReplicationGrup if any
	ReplicatedPVCNames []string // Names of replicated PVCs
}

const (
	notFound                     = "not found"
	hostNameTopologyKey          = "kubernetes.io/hostname"
	arrayIDVolumeAttribute       = "arrayID"
	storageSystemVolumeAttribute = "StorageSystem"
	defaultArray                 = "default"
)

// controllerModePodHandler handles controller mode functionality when a pod event happens
func (cm *PodMonitorType) controllerModePodHandler(pod *v1.Pod, eventType watch.EventType) error {
	log.Debugf("podMonitorHandler-controller:  name %s/%s node %s message %s reason %s event %v",
		pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, pod.Spec.NodeName, pod.Status.Message, pod.Status.Reason, eventType)

	driverNamespace := os.Getenv("MY_POD_NAMESPACE")
	log.Debugf("podMonitorHandler-controller: driverNamespace %s", driverNamespace)
	// For driver pod
	if driverNamespace == pod.ObjectMeta.Namespace {
		return cm.controllerModeDriverPodHandler(pod, eventType)
	}
	// Lock so that only one thread is processing pod at a time
	podKey := getPodKey(pod)
	// Clean up pod key to PodInfo and CrashLoopBackOffCount mappings if deleting.
	if eventType == watch.Deleted {
		cm.PodKeyToControllerPodInfo.Delete(podKey)
		cm.PodKeyToCrashLoopBackOffCount.Delete(podKey)
		return nil
	}
	// Single thread processing of this pod
	Lock(podKey, pod, LockSleepTimeDelay)
	defer Unlock(podKey)
	// Check that pod is still present
	ctx, cancel := K8sAPI.GetContext(MediumTimeout)
	defer cancel()
	pod, err := K8sAPI.GetPod(ctx, pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
	if err != nil {
		log.Errorf("GetPod failed: %s: %s", podKey, err)
		return err
	}
	if pod.Spec.NodeName != "" {
		log.Debugf("Getting node %s", pod.Spec.NodeName)
		node, err := K8sAPI.GetNode(ctx, pod.Spec.NodeName)
		if err != nil {
			log.Errorf("GetNode failed: %s: %s", pod.Spec.NodeName, err)
		} else {
			if cm.GetNodeUID(pod.Spec.NodeName) != string(node.ObjectMeta.UID) {
				log.Debugf("Updating NodeUid from GetNode: %s -> %s", pod.Spec.NodeName, node.ObjectMeta.UID)
				cm.StoreNodeUID(pod.Spec.NodeName, string(node.ObjectMeta.UID))
			}

			// Determine if node tainted
			taintnosched := nodeHasTaint(node, nodeUnreachableTaint, v1.TaintEffectNoSchedule)
			taintnoexec := nodeHasTaint(node, nodeUnreachableTaint, v1.TaintEffectNoExecute)
			taintpodmon := nodeHasTaint(node, PodmonTaintKey, v1.TaintEffectNoSchedule) || nodeHasTaint(node, PodmonDriverPodTaintKey, v1.TaintEffectNoSchedule)

			// Determine pod status
			ready, initialized, _ := podStatus(pod.Status.Conditions)

			// Loop for containerStatus for CrashLoopBackOff
			crashLoopBackOff := false
			containerStatuses := pod.Status.ContainerStatuses
			for _, containerStatus := range containerStatuses {
				log.Debugf("container status ID %s ready %v state %v", containerStatus.ContainerID, containerStatus.Ready, containerStatus.State)
				if containerStatus.State.Waiting != nil && containerStatus.State.Waiting.Reason == crashLoopBackOffReason {
					crashLoopBackOff = true
				}
			}

			// If ready, we want to save the PodKeyToControllerPodInfo
			// It will use these items to clean up pods if the array reports no connectivity.
			// Update podInfo if pod is evict from a node to another node
			if ready || (eventType == watch.Modified) {
				arrayIDs, pvcCount, err := cm.podToArrayIDs(ctx, pod)
				if err != nil {
					log.Errorf("Could not determine pod to arrayIDs: %s", err)
				} else {
					// Do not keep track of Volumeless pods
					if IgnoreVolumelessPods && pvcCount == 0 {
						log.Infof("podKey %s ignore because Volumeless", podKey)
						return nil
					}
				}
				log.Infof("podKey %s pvcCount %d arrayIDs %v", podKey, pvcCount, arrayIDs)

				podAffinityLabels := cm.getPodAffinityLabels(pod)
				if len(podAffinityLabels) > 0 {
					log.Infof("podKey %s podAffinityLabels %v", podKey, podAffinityLabels)
				}
				podUID := string(pod.ObjectMeta.UID)
				podInfo := &ControllerPodInfo{
					PodKey:            podKey,
					Node:              node.DeepCopy(),
					PodUID:            podUID,
					ArrayIDs:          arrayIDs,
					PodAffinityLabels: podAffinityLabels,
				}
				log.Infof("Updating protected pod info podKey %s pvcCount %d arrayIDs %v", podKey, pvcCount, arrayIDs)
				cm.PodKeyToControllerPodInfo.Store(podKey, podInfo)
				if ready {
					// Delete (reset) the CrashLoopBackOff counter since we're running.
					cm.PodKeyToCrashLoopBackOffCount.Delete(podKey)
				}
			}

			log.Infof("podMonitorHandler: namespace: %s name: %s nodename: %s initialized: %t ready: %t taints [nosched: %t noexec: %t podmon: %t ]",
				pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, pod.Spec.NodeName, initialized, ready, taintnosched, taintnoexec, taintpodmon)
			if (taintnoexec || taintnosched || taintpodmon) && !ready {
				// Use the last podInfo recorded when pod ready to make sure node has an annotation for the CSI NodeID
				podInfoValue, ok := cm.PodKeyToControllerPodInfo.Load(podKey)
				if ok {
					controllerPodInfo := podInfoValue.(*ControllerPodInfo)
					if getCSINodeIDAnnotation(controllerPodInfo.Node, cm.DriverPathStr) != "" {
						node = controllerPodInfo.Node
					}
				}
				go cm.controllerCleanupPod(pod, node, "NodeFailure", taintnoexec, taintpodmon)
			} else if !ready && crashLoopBackOff {
				cnt, _ := cm.PodKeyToCrashLoopBackOffCount.LoadOrStore(podKey, 0)
				crashLoopBackOffCount := cnt.(int)
				if crashLoopBackOffCount < MaxCrashLoopBackOffRetry {
					log.Infof("cleaning up CrashLoopBackOff pod %s", podKey)
					if err = K8sAPI.CreateEvent(podmon, pod, k8sapi.EventTypeWarning, crashLoopBackOffReason, "podmon cleaning pod %s with delete",
						string(pod.ObjectMeta.UID), node.ObjectMeta.Name, fmt.Sprintf("retry: %d", crashLoopBackOffCount)); err != nil {
						log.Errorf("Failed to send %s event: %s", crashLoopBackOffReason, err.Error())
					}
					err = K8sAPI.DeletePod(ctx, pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, pod.ObjectMeta.UID, false)
					crashLoopBackOffCount = crashLoopBackOffCount + 1
					cm.PodKeyToCrashLoopBackOffCount.Store(podKey, crashLoopBackOffCount)
				}
			}
		}

	} else {
		// no Node association for podready, initialized, _ := podStatus(pod.Status.Conditions)
		_, _, pending := podStatus(pod.Status.Conditions)
		if pending {
			cm.checkPendingPod(ctx, pod)
		}
	}
	return nil
}

// Attempts to cleanup a Pod that is in trouble. Returns true if made it all the way to deleting the pod.
func (cm *PodMonitorType) controllerCleanupPod(pod *v1.Pod, node *v1.Node, reason string, taintnoexec, taintpodmon bool) bool {
	fields := make(map[string]interface{})
	fields["namespace"] = pod.ObjectMeta.Namespace
	fields["pod"] = pod.ObjectMeta.Name
	fields["node"] = node.ObjectMeta.Name
	fields["reason"] = reason
	// Lock so that only one thread is processing pod at a time
	podKey := getPodKey(pod)
	// Single thread processing of this pod
	Lock(podKey, pod, LockSleepTimeDelay)
	defer Unlock(podKey)

	// If ControllerPodInfo struct has UID mismatch, assume pod deleted already
	podInfoValue, ok := cm.PodKeyToControllerPodInfo.Load(podKey)
	if ok {
		controllerPodInfo := podInfoValue.(*ControllerPodInfo)
		if controllerPodInfo.PodUID != string(pod.ObjectMeta.UID) {
			log.Infof("monitored pod UID %s different than pod to clean UID %s - aborting pod cleanup", controllerPodInfo.PodUID, string(pod.ObjectMeta.UID))
			return false
		}
	}

	log.WithFields(fields).Infof("Cleaning up pod")
	ctx, cancel := K8sAPI.GetContext(LongTimeout)
	defer cancel()
	// Get the volume attachments

	// Get the PVs associated with this pod.
	pvlist, err := K8sAPI.GetPersistentVolumesInPod(ctx, pod)
	if err != nil {
		log.WithFields(fields).Errorf("Could not get PersistentVolumes: %s", err)
		return false
	}

	// ignoreVolumeless pod
	if IgnoreVolumelessPods && len(pvlist) == 0 {
		log.WithFields(fields).Infof("Ignoring volumeless pod")
		return true
	}

	// Get the volume handles from the PVs
	volIDs := make([]string, 0)
	for _, pv := range pvlist {
		pvsrc := pv.Spec.PersistentVolumeSource
		if pvsrc.CSI != nil {
			volIDs = append(volIDs, pvsrc.CSI.VolumeHandle)
		}
	}
	if len(pvlist) != len(volIDs) {
		log.WithFields(fields).Warnf("Could not get volume handles for every PV: pvs %d volIDs %d", len(pvlist), len(volIDs))
	}

	// Get the VolumeAttachments for each of the PVs.
	valist := make([]*storagev1.VolumeAttachment, 0)
	vaNamesToDelete := make([]string, 0)
	for _, pv := range pvlist {
		va, err := K8sAPI.GetCachedVolumeAttachment(ctx, pv.ObjectMeta.Name, node.ObjectMeta.Name)
		if err != nil {
			log.WithFields(fields).Errorf("Could not get cached VolumeAttachment: %s", err)
			return false
		}
		if va != nil {
			valist = append(valist, va)
			vaNamesToDelete = append(vaNamesToDelete, va.ObjectMeta.Name)
		}
	}

	// Call the driver to validate the volumes are not in use
	if cm.CSIExtensionsPresent && CSIApi.Connected() {
		log.WithFields(fields).Infof("Validating host connectivity for node %s volumes %v", node.ObjectMeta.Name, volIDs)
		connected, iosInProgress, err := cm.callValidateVolumeHostConnectivity(node, volIDs, true)
		// Don't consider connected status if taintpodmon is set, because the node may just have come back online.
		if (connected && !taintpodmon) || iosInProgress || err != nil {
			fields["connected"] = connected
			fields["iosInProgress"] = iosInProgress
			// If SkipArrayConnectionValidation and taintnoexec are set, proceed anyway
			if cm.SkipArrayConnectionValidation && taintnoexec {
				log.WithFields(fields).Info("SkipArrayConnectionValidation is set and taintnoexec is true- proceeding")
			} else {
				if err != nil {
					log.WithFields(fields).Info("Aborting pod cleanup due to error: ", err.Error())
					if strings.Contains(err.Error(), "Could not determine CSI NodeID for node") {
						if err = K8sAPI.CreateEvent(podmon, pod, k8sapi.EventTypeWarning, reason,
							"podmon aborted pod cleanup %s due to missing CSI annotations",
							string(pod.ObjectMeta.UID), node.ObjectMeta.Name); err != nil {
							log.Errorf("Failed to send %s event: %s", reason, err.Error())
						}
					} else {
						if err = K8sAPI.CreateEvent(podmon, pod, k8sapi.EventTypeWarning, reason,
							"podmon aborted pod cleanup %s due to error while validating volume host connectivity",
							string(pod.ObjectMeta.UID), node.ObjectMeta.Name); err != nil {
							log.Errorf("Failed to send %s event: %s", reason, err.Error())
						}
					}
					return false
				}
				log.WithFields(fields).Info("Aborting pod cleanup because array still connected and/or recently did I/O")
				if err = K8sAPI.CreateEvent(podmon, pod, k8sapi.EventTypeWarning, reason,
					"podmon aborted pod cleanup %s array connected or recent I/O",
					string(pod.ObjectMeta.UID), node.ObjectMeta.Name); err != nil {
					log.Errorf("Failed to send %s event: %s", reason, err.Error())
				}
				return false
			}
		}
	} else {
		log.WithFields(fields).Error("Array validation check skipped because CSIApi not connected")
	}

	// Fence all the volumes
	if CSIApi.Connected() {
		log.WithFields(fields).Infof("Commencing fencing of the node")
		nerrors := 0
		for _, volID := range volIDs {
			err := cm.callControllerUnpublishVolume(node, volID)
			if err != nil {
				nerrors++
			}
		}
		if nerrors > 0 {
			log.WithFields(fields).Errorf("There were %d errors calling ControllerUnpublishVolume to fence the node. Aborting pod cleanup.", nerrors)
			if err = K8sAPI.CreateEvent(podmon, pod, k8sapi.EventTypeWarning, reason,
				"podmon aborted pod cleanup %s couldn't fence volumes",
				string(pod.ObjectMeta.UID), node.ObjectMeta.Name); err != nil {
				log.Errorf("Failed to send %s event: %s", reason, err.Error())
			}
			return false
		}
	}

	// Add a taint for the pod on the node.
	if err = taintNode(node.ObjectMeta.Name, PodmonTaintKey, false); err != nil {
		log.WithFields(fields).Errorf("Failed to update taint against %s node: %v", node.ObjectMeta.Name, err)
		return false
	}

	// Delete all the volumeattachments attached to our pod
	for _, vaName := range vaNamesToDelete {
		err = K8sAPI.DeleteVolumeAttachment(ctx, vaName)
		if err != nil {
			err = K8sAPI.DeleteVolumeAttachment(ctx, vaName)
			if err != nil && !strings.Contains(err.Error(), notFound) {
				log.WithFields(fields).Errorf("Couldn't delete VolumeAttachment- aborting after retry: %s: %s", vaName, err.Error())
				return false
			}
		}
	}

	// Force delete the pod.
	if err = K8sAPI.CreateEvent(podmon, pod, k8sapi.EventTypeWarning, reason,
		"podmon cleaning pod %s with force delete",
		string(pod.ObjectMeta.UID), node.ObjectMeta.Name); err != nil {
		log.Errorf("Failed to send %s event: %s", reason, err.Error())
	}
	err = K8sAPI.DeletePod(ctx, pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, pod.ObjectMeta.UID, true)
	if err == nil {
		log.WithFields(fields).Infof("Successfully cleaned up pod")
		// Delete the ControllerPodInfo reference to this pod, we've deleted it.
		cm.PodKeyToControllerPodInfo.Delete(podKey)
		return true
	}
	log.WithFields(fields).Errorf("Delete pod failed")
	return false
}

// call ValidateVolumeHostConnectivity in the driver, log any messages, and then
// return the booleans Connected and IosInProgress.
func (cm *PodMonitorType) callValidateVolumeHostConnectivity(node *v1.Node, volumeIDs []string, logIt bool) (bool, bool, error) {
	// Get the CSI annotations for nodeID
	csiNodeID := getCSINodeIDAnnotation(node, cm.DriverPathStr)
	if csiNodeID == "" {
		log.Infof("retrying getCSINodeIDAnnotation node %s", node.Name)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		node, err := K8sAPI.GetNode(ctx, node.Name)
		if err == nil {
			csiNodeID = getCSINodeIDAnnotation(node, cm.DriverPathStr)
		}
	}
	if csiNodeID != "" {
		// Validate host connectivity for the node
		req := &csiext.ValidateVolumeHostConnectivityRequest{
			NodeId: csiNodeID,
		}
		if len(volumeIDs) > 0 {
			req.VolumeIds = volumeIDs
		}
		log.Debugf("calling ValidateVolumeHostConnectivity with %v", req)
		// Get the connected status of the Node to the StorageSystem
		ctx, cancel := context.WithTimeout(context.Background(), ShortTimeout)
		defer cancel()
		resp, err := CSIApi.ValidateVolumeHostConnectivity(ctx, req)
		if err != nil {
			if strings.Contains(err.Error(), "there is no corresponding SDC") {
				// This error is returned if the array cannot find the SDC, which can happen on connectivity loss
				log.Errorf("%s", err.Error())
				return false, false, nil
			}
			log.Errorf("Error checking ValidateVolumeHostConnectivity: %s", err.Error())
			return true, true, err
		}
		if logIt {
			for _, message := range resp.Messages {
				log.Info(message)
			}
		}
		log.Infof("ValidateVolumeHostConnectivity Node %s NodeId %s Connected %t", node.ObjectMeta.Name, req.NodeId, resp.GetConnected())
		return resp.GetConnected(), resp.GetIosInProgress(), nil
	}
	return false, false, fmt.Errorf("callValidateVolumeHostConnectivity: Could not determine CSI NodeID for node: %s", node.ObjectMeta.Name)
}

// callControllerUnpublishVolume in the driver, log any messages, return error.
func (cm *PodMonitorType) callControllerUnpublishVolume(node *v1.Node, volumeID string) error {
	var err error
	csiNodeID := getCSINodeIDAnnotation(node, cm.DriverPathStr)
	if csiNodeID == "" {
		log.Errorf("callControllerUnpublishVolume: Could not determine CSI NodeID for node: %s", node.ObjectMeta.Name)
		return errors.New("csiNodeID is not set")
	}
	for i := 0; i < CSIMaxRetries; i++ {
		// Get the CSI annotations for nodeID
		log.Infof("Calling ControllerUnpublishVolume node id %s volume %s", csiNodeID, volumeID)
		req := &csi.ControllerUnpublishVolumeRequest{
			NodeId:   csiNodeID,
			VolumeId: volumeID,
		}
		_, err = CSIApi.ControllerUnpublishVolume(context.Background(), req)
		if err == nil {
			break
		}
		log.Errorf("Error fencing volume using ControllerUnpublishVolum node %s volume %s: %s", csiNodeID, volumeID, err.Error())
		if !strings.HasSuffix(err.Error(), "pending") {
			break
		}
		time.Sleep(PendingRetryTime)
	}
	return err
}

// podToArrayIDs returns the array IDs used by the pod, along with pvCount, and error
// TBD: Check if VolumeAttributes of StorageSystem set for all arrays
func (cm *PodMonitorType) podToArrayIDs(ctx context.Context, pod *v1.Pod) ([]string, int, error) {
	arrayIDs := make([]string, 0)
	pvlist, err := K8sAPI.GetPersistentVolumesInPod(ctx, pod)
	if err != nil {
		return arrayIDs, len(pvlist), err
	}
	for _, pv := range pvlist {
		storageSystem := pv.Spec.CSI.VolumeAttributes["StorageSystem"]
		log.Infof("podToArrayIDs pv %s storageSystem %s", pv.Name, storageSystem)
		if storageSystem == "" {
			// Maybe this is a replicated volume using a remoteSystem as the ID
			// TBD - is this convention used by other drivers than Powerflex?
			if strings.HasPrefix("replicated-", pv.Name) {
				storageSystem = pv.Spec.CSI.VolumeAttributes["remoteSystem"]
			}
			if storageSystem == "" {
				storageSystem = defaultArray
			}
		}
		if !stringInSlice(storageSystem, arrayIDs) {
			arrayIDs = append(arrayIDs, storageSystem)
		}
	}
	return arrayIDs, len(pvlist), nil
}

// ArrayConnectivityMonitor -- periodically checks array connectivity to all the nodes using it.
// If connectivity is lost, will initiate cleanup of the pods.
// This is a never ending function, intended to be called as Go routine.
func (cm *PodMonitorType) ArrayConnectivityMonitor() {
	// Loop through all the monitored Pods making sure they still have array access
	for {
		podKeysToClean := make([]string, 0)
		nodesToTaint := make(map[string]bool)

		// Clear the connectivity cache so it will sample again.
		connectivityCache.ResetSampled()
		// Internal function for iterating PodKeyToControllerPodInfo
		// This will clean up Pods that have lost connectivity to at least one of their arrays
		fnPodKeyToControllerPodInfo := func(_, value interface{}) bool {
			controllerPodInfo := value.(*ControllerPodInfo)
			podKey := controllerPodInfo.PodKey
			node := controllerPodInfo.Node

			// Check if we have connectivity for all our array ids
			connected := true
			for _, arrayID := range controllerPodInfo.ArrayIDs {
				cnct := connectivityCache.CheckConnectivity(cm, node, arrayID)
				if !cnct {
					log.Infof("Pod %s node %s has no connectivity to arrayID %s", podKey, node.ObjectMeta.Name, arrayID)
					connected = false
				}
			}
			if !connected {
				nodesToTaint[node.ObjectMeta.Name] = true
				podKeysToClean = append(podKeysToClean, podKey)
			}
			return true
		}

		// Process all the pods, generating the associated connectivity cache entries
		cm.PodKeyToControllerPodInfo.Range(fnPodKeyToControllerPodInfo)

		// Taint all the nodes that were not connected
		for nodeName := range nodesToTaint {
			log.Infof("Tainting node %s because of connectivity loss", nodeName)
			err := taintNode(nodeName, PodmonTaintKey, false)
			if err != nil {
				log.Errorf("Unable to taint node: %s: %s", nodeName, err.Error())
			}
		}

		// Cleanup pods that are on the tainted nodes.
		for _, podKey := range podKeysToClean {
			// Fetch the pod.
			info, ok := cm.PodKeyToControllerPodInfo.Load(podKey)
			if !ok {
				continue
			}
			podInfo := info.(*ControllerPodInfo)
			if len(podInfo.PodAffinityLabels) > 0 {
				// Process all the pods with affinity together
				log.Infof("Processing pods with affinity %v", podInfo.PodAffinityLabels)
				for _, podKey := range podKeysToClean {
					// Fetch the pod.
					infox, ok := cm.PodKeyToControllerPodInfo.Load(podKey)
					if !ok {
						continue
					}
					podInfox := infox.(*ControllerPodInfo)
					if mapEqualsMap(podInfo.PodAffinityLabels, podInfox.PodAffinityLabels) {
						cm.ProcessPodInfoForCleanup(podInfox, "ArrayConnectivityLoss")
					}
				}
				log.Infof("End Processing pods with affinity %v", podInfo.PodAffinityLabels)
			} else {
				cm.ProcessPodInfoForCleanup(podInfo, "ArrayConnectivityLoss")
			}
		}

		// Sleep according to the NODE_CONNECTIVITY_POLL_RATE
		pollRate := GetArrayConnectivityPollRate()
		time.Sleep(pollRate)
		if pollRate < 10*time.Millisecond {
			// disabled or unit testing exit
			return
		}
	}
}

// ProcessPodInfoForCleanup processes a ControllerPodInfo for cleanup, checking that the UID and object are the same, and then calling controllerCleanupPod.
func (cm *PodMonitorType) ProcessPodInfoForCleanup(podInfo *ControllerPodInfo, reason string) {
	podNamespace, podName := splitPodKey(podInfo.PodKey)
	ctx, cancel := K8sAPI.GetContext(MediumTimeout)
	defer cancel()
	pod, err := K8sAPI.GetPod(ctx, podNamespace, podName)
	if err == nil {
		if string(pod.ObjectMeta.UID) == podInfo.PodUID && pod.Spec.NodeName == podInfo.Node.ObjectMeta.Name {
			log.Infof("Cleaning up pod %s/%s because of %s", reason, pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
			cm.controllerCleanupPod(pod, podInfo.Node, reason, false, false)
		} else {
			log.Infof("Skipping pod %s/%s podUID %s %s node %s %s", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name,
				string(pod.ObjectMeta.UID), podInfo.PodUID, pod.Spec.NodeName, podInfo.Node.ObjectMeta.Name)
		}
	}
}

type nodeArrayConnectivityCache struct {
	initOnce                       sync.Once       // Will be set after initialization
	nodeArrayConnectivitySampled   map[string]bool // If true, already sampled, if need to call array to verify connectivity
	nodeArrayConnectivityLossCount map[string]int  // 0 means connected, > 0 number of connection loss for n samples
}

var connectivityCache nodeArrayConnectivityCache

// ArrayConnectivityConnectionLossThreshold is the number of consecutive samples that must fail before we declare connectivity loss
var ArrayConnectivityConnectionLossThreshold = 3

// CheckConnectivity returns true if the node has connectivity to the arrayID supplied
func (nacc *nodeArrayConnectivityCache) CheckConnectivity(cm *PodMonitorType, node *v1.Node, arrayID string) bool {
	if node == nil {
		return true
	}
	nodeUID := cm.GetNodeUID(node.ObjectMeta.Name)
	if nodeUID == "" || nodeUID != string(node.ObjectMeta.UID) {
		log.Infof("CheckConnectivity: node %s has stale node uid", node.ObjectMeta.Name)
		return true
	}
	key := node.ObjectMeta.Name + ":" + arrayID
	if nacc.nodeArrayConnectivitySampled[key] == false {
		// Determine connectivity
		volumeIDs := make([]string, 0)
		connected, _, err := cm.callValidateVolumeHostConnectivity(node, volumeIDs, false)
		if err != nil {
			log.Infof("Could not determine array connectivity, assuming connected, error: %s", err)
			return true
		}
		nacc.nodeArrayConnectivitySampled[key] = true
		if connected {
			nacc.nodeArrayConnectivityLossCount[key] = 0
		} else {
			nacc.nodeArrayConnectivityLossCount[key] = nacc.nodeArrayConnectivityLossCount[key] + 1
		}
	}
	// If below the ConnectionLossThreshold, assume we could be connected
	return nacc.nodeArrayConnectivityLossCount[key] < ArrayConnectivityConnectionLossThreshold
}

func (nacc *nodeArrayConnectivityCache) ResetSampled() {
	nacc.initOnce.Do(func() {
		nacc.nodeArrayConnectivitySampled = make(map[string]bool)
		nacc.nodeArrayConnectivityLossCount = make(map[string]int)
	})
	for key := range nacc.nodeArrayConnectivitySampled {
		nacc.nodeArrayConnectivitySampled[key] = false
	}
}

// getCSINodeIDAnnotation gets the csi.volume.kubernetes.io/nodeid annotation for a given driver
// path like csi-vxflexos.dellemc.com
func getCSINodeIDAnnotation(node *v1.Node, driverPath string) string {
	annotations := node.ObjectMeta.Annotations
	if annotations != nil {
		// Get the csi.volume.kubernetes.io/nodeid annotation
		csiAnnotations := annotations["csi.volume.kubernetes.io/nodeid"]
		log.Infof("Node annotations: %s %v", driverPath, annotations)
		if csiAnnotations != "" {
			log.Debugf("csiAnnotations: %s", csiAnnotations)
			var csiAnnotationsMap map[string]json.RawMessage
			err := json.Unmarshal([]byte(csiAnnotations), &csiAnnotationsMap)
			if err != nil {
				log.Errorf("could not unmarshal csi annotations %s to json: %s", csiAnnotations, err.Error())
				return ""
			}
			var nodeID string
			err = json.Unmarshal(csiAnnotationsMap[driverPath], &nodeID)
			if err != nil {
				log.Errorf("could not unmarshal driver path key from nodeid annotation %s: to json: %s", csiAnnotations, err.Error())
				return ""
			}

			log.Debugf("Returning CSI Node ID Annotation: %s", nodeID)
			return nodeID
		}
		log.Errorf("No annotation on node %s for csi.volume.kubernetes.io/nodeid: %s", node.ObjectMeta.Name, csiAnnotations)
	}
	log.Errorf("No annotations on node %s", node.ObjectMeta.Name)
	return ""
}

func callK8sAPITaint(operation, nodeName, taintKey string, effect v1.TaintEffect, remove bool) error {
	log.Infof("Calling to %s %s with %s %s (remove = %v)", operation, nodeName, taintKey, effect, remove)
	ctx, cancel := K8sAPI.GetContext(MediumTimeout)
	defer cancel()
	return K8sAPI.TaintNode(ctx, nodeName, taintKey, effect, remove)
}

// taintNode adds or removes the podmon taint against node with 'nodeName'
func taintNode(nodeName, taintKey string, removeTaint bool) error {
	operation := "tainting "
	if removeTaint {
		operation = "untainting "
	}
	return callK8sAPITaint(operation, nodeName, taintKey, v1.TaintEffectNoSchedule, removeTaint)
}

func nodeHasTaint(node *v1.Node, key string, taintEffect v1.TaintEffect) bool {
	for _, taint := range node.Spec.Taints {
		if taint.Key == key && taint.Effect == taintEffect {
			return true
		}
	}
	return false
}

// getPodAffinityLabels returns nil if no node affinity is specified. If node affinity is specified,
// podPodAffinity returns a map of podLabels for pods the specificed pod should have affinity with.
func (cm *PodMonitorType) getPodAffinityLabels(pod *v1.Pod) map[string]string {
	result := make(map[string]string)
	affinity := pod.Spec.Affinity
	if affinity == nil {
		return result
	}
	podAffinity := affinity.PodAffinity
	if podAffinity == nil {
		return result
	}
	requiredDuringSchedulingIgnoredDuringExecution := podAffinity.RequiredDuringSchedulingIgnoredDuringExecution
	if requiredDuringSchedulingIgnoredDuringExecution == nil {
		return result
	}
	for _, schedConstraints := range requiredDuringSchedulingIgnoredDuringExecution {
		topologyKey := schedConstraints.TopologyKey
		if topologyKey != hostNameTopologyKey {
			continue
		}
		labelSelector := schedConstraints.LabelSelector
		if labelSelector == nil {
			continue
		}
		matchLabels := labelSelector.MatchLabels
		for k, v := range matchLabels {
			result[k] = v
		}
		for _, matchExpr := range labelSelector.MatchExpressions {
			if matchExpr.Operator != "In" {
				continue
			}
			for _, v := range matchExpr.Values {
				result[matchExpr.Key] = v
			}
		}

	}
	return result
}

// mapEqualsMap returns true IFF string map1 contains the same elements as map2
func mapEqualsMap(map1, map2 map[string]string) bool {
	if len(map1) != len(map2) {
		return false
	}
	for k1, v1 := range map1 {
		v2, ok := map2[k1]
		if !ok || v2 != v1 {
			return false
		}
	}
	return true
}

// controllerModeDriverPodHandler handles controller mode functionality when a driver pod event happens
func (cm *PodMonitorType) controllerModeDriverPodHandler(pod *v1.Pod, eventType watch.EventType) error {
	log.Debugf("controllerModeDriverPodHandler-controller:  name %s/%s node %s message %s reason %s event %v",
		pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, pod.Spec.NodeName, pod.Status.Message, pod.Status.Reason, eventType)

	// Lock so that only one thread is processing pod at a time
	podKey := getPodKey(pod)
	// Single thread processing of this pod
	Lock(podKey, pod, LockSleepTimeDelay)
	defer Unlock(podKey)
	// Check that pod is still present
	ctx, cancel := K8sAPI.GetContext(MediumTimeout)
	defer cancel()
	pod, err := K8sAPI.GetPod(ctx, pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
	if err != nil {
		log.Errorf("GetPod failed: %s: %s", podKey, err)
		return err
	}
	if pod.Spec.NodeName != "" {
		log.Debugf("Getting node %s", pod.Spec.NodeName)
		node, err := K8sAPI.GetNode(ctx, pod.Spec.NodeName)
		if err != nil {
			log.Errorf("GetNode failed: %s: %s", pod.Spec.NodeName, err)
		} else {
			// Determine pod status
			ready, initialized, _ := podStatus(pod.Status.Conditions)

			if !ready {
				log.Infof("Taint node %s with %s driver node pod down", node.ObjectMeta.Name, PodmonDriverPodTaintKey)
				err := taintNode(node.ObjectMeta.Name, PodmonDriverPodTaintKey, false)
				if err != nil {
					log.Errorf("Unable to taint node: %s: %s", node.ObjectMeta.Name, err.Error())
				}
			} else {
				hasTaint := nodeHasTaint(node, PodmonDriverPodTaintKey, v1.TaintEffectNoSchedule)
				log.Infof("Removing taint from node %s with %s", node.ObjectMeta.Name, PodmonDriverPodTaintKey)
				// remove taint
				if hasTaint {
					err := taintNode(node.ObjectMeta.Name, PodmonDriverPodTaintKey, true)
					if err != nil {
						log.Errorf("Unable to untaint node: %s: %s", node.ObjectMeta.Name, err.Error())
					}
				}
			}

			log.Infof("podMonitorHandler: namespace: %s name: %s nodename: %s initialized: %t ready: %t ",
				pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, pod.Spec.NodeName, initialized, ready)
		}
	}

	return nil
}

// podStatus determine pod status. Returns ready, initialized, pending
func podStatus(conditions []v1.PodCondition) (bool, bool, bool) {
	ready := false
	initialized := true
	pending := false
	for _, condition := range conditions {
		log.Debugf("pod condition.Type: %s %v", condition.Type, condition.Status)
		if condition.Type == podReadyCondition {
			ready = condition.Status == v1.ConditionTrue
		}
		if condition.Type == podInitializedCondition {
			initialized = condition.Status == v1.ConditionTrue
		}
		if condition.Type == podScheduledCondition {
			if condition.Status == v1.ConditionFalse && condition.Reason == podUnschedulableReason {
				log.Infof("pod unschedulable")
				pending = true
			}
		}
	}
	return ready, initialized, pending
}

// arrayConnectivity  map of arrayID to map[node]bool (true for connected, false for disconnected)
var arrayConnectivity sync.Map

// updateArrayConnectivityMap updates an array connectivity map based on updates to Node objects made
// by the node controller. It contains a mapping for each array of which nodes think they have connectivity.
func (cm *PodMonitorType) updateArrayConnectivityMap(node *v1.Node) {
	// Loop through the node's labels, updating array entrieso
	log.Infof("updateArrayConnectivityMap updating node %s", node.Name)
	prefix := cm.DriverPathStr + "/"
	for k, v := range node.Labels {
		if strings.HasPrefix(k, prefix) {
			log.Infof("node.Labels k %s v %s prefix %s", k, v, prefix)
			parts := strings.Split(k, "/")
			if len(parts) <= 2 {
				continue
			}
			arrayID := parts[1]
			swapped := false
			for !swapped {
				data, ok := arrayConnectivity.Load(arrayID)
				var nodeConnectivity map[string]bool
				nodeConnectivity = make(map[string]bool)
				if ok {
					nodeConnectivity = data.(map[string]bool)
				}
				original := nodeConnectivity
				nodeUID := cm.GetNodeUID(node.ObjectMeta.Name)
				if nodeUID == "" {
					log.Errorf("Could not get nodeUID: %s %s", node.Name, nodeUID)
					continue
				}
				nodeConnectivity[nodeUID] = true
				if v == "Disconnected" {
					nodeConnectivity[nodeUID] = false
				}
				log.Infof("updateArrayConnectivityMap %s %s %s %t", arrayID, node.Name, nodeUID, nodeConnectivity[nodeUID])
				swapped = arrayConnectivity.CompareAndSwap(arrayID, original, nodeConnectivity)
			}
		}
	}
}

// getArrayConnectivityMap returns the node connectivity map for the specified array
// which contains a mapping of nodeUID to boolean connected. This information was determined
// by the podmon node controller for each node.
func (cm *PodMonitorType) getNodeConnectivityMap(arrayID string) map[string]bool {
	value, ok := arrayConnectivity.Load(arrayID)
	if ok {
		return value.(map[string]bool)
	}
	return make(map[string]bool)
}

// stringInSlice return sture if the passed search string is in the slice
func stringInSlice(search string, slice []string) bool {
	for _, v := range slice {
		if search == v {
			return true
		}
	}
	return false
}
