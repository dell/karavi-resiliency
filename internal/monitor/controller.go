/*
 * Copyright (c) 2021. Dell Inc., or its subsidiaries. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 */

package monitor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	csi "github.com/container-storage-interface/spec/lib/go/csi"
	csiext "github.com/dell/dell-csi-extensions/podmon"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
)

//ControllerPodInfo has information for tracking health of the system
type ControllerPodInfo struct { // information controller keeps on hand about a pod
	PodKey   string   // the Pod Key (namespace/name) of the pod
	Node     *v1.Node // the associated node structure
	PodUID   string   // the pod container's UID
	ArrayIDs []string // string of array IDs used by the pod's volumes
}

// controllerModePodHandler handles controller mode functionality when a pod event happens
func (cm *PodMonitorType) controllerModePodHandler(pod *v1.Pod, eventType watch.EventType) error {
	log.Debugf("podMonitorHandler-controller:  name %s/%s node %s message %s reason %s event %v",
		pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, pod.Spec.NodeName, pod.Status.Message, pod.Status.Reason, eventType)
	// Lock so that only one thread is processing pod at a time
	podKey := getPodKey(pod)
	// Clean up pod key to node mapping if deleting.
	if eventType == watch.Deleted {
		cm.PodKeyToControllerPodInfo.Delete(podKey)
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

			// Determine if node tainted
			taintnosched := nodeHasTaint(node, nodeUnreachableTaint, v1.TaintEffectNoSchedule)
			taintnoexec := nodeHasTaint(node, nodeUnreachableTaint, v1.TaintEffectNoExecute)

			// Determine pod status
			ready := false
			initialized := true
			conditions := pod.Status.Conditions
			for _, condition := range conditions {
				log.Debugf("pod condition.Type: %s %v", condition.Type, condition.Status)
				if condition.Type == podReadyCondition {
					ready = condition.Status == v1.ConditionTrue
				}
				if condition.Type == podInitializedCondition {
					initialized = condition.Status == v1.ConditionTrue
				}
			}

			// If ready, we want to save the PodKeyToControllerPodInfo
			// It will use these items to clean up pods if the array reports no connectivity.
			if ready {
				arrayIDs, err := cm.podToArrayIDs(pod)
				if err != nil {
					log.Errorf("Could not determine pod to arrayIDs: %s", err)
				}
				podUID := string(pod.ObjectMeta.UID)
				podInfo := &ControllerPodInfo{
					PodKey:   podKey,
					Node:     node,
					PodUID:   podUID,
					ArrayIDs: arrayIDs,
				}
				cm.PodKeyToControllerPodInfo.Store(podKey, podInfo)
			}

			log.Printf("podMonitorHandler: namespace: %s name: %s nodename: %s initialized: %t taint-nosched: %t taint-noexec: %t ready: %t",
				pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, pod.Spec.NodeName, initialized, taintnosched, taintnoexec, ready)
			// TODO: option for taintnosched vs. taintnoexec
			if (taintnoexec || taintnosched) && !ready {
				go cm.controllerCleanupPod(pod, node)
			}
		}

	}
	return nil
}

// Attempts to cleanup a Pod that is in trouble. Returns true if made it all the way to deleting the pod.
func (cm *PodMonitorType) controllerCleanupPod(pod *v1.Pod, node *v1.Node) bool {
	fields := make(map[string]interface{})
	fields["namespace"] = pod.ObjectMeta.Namespace
	fields["pod"] = pod.ObjectMeta.Name
	fields["node"] = node.ObjectMeta.Name
	// Lock so that only one thread is processing pod at a time
	podKey := getPodKey(pod)
	// Single thread processing of this pod
	Lock(podKey, pod, LockSleepTimeDelay)
	defer Unlock(podKey)

	log.WithFields(fields).Infof("Cleaning up pod")
	ctx, cancel := K8sAPI.GetContext(LongTimeout)
	defer cancel()
	// Get the volume attachments
	volumeAttachmentList, err := K8sAPI.GetVolumeAttachments(ctx)
	if err != nil {
		log.WithFields(fields).Errorf("Could not get volumeAttachments: %s", err)
		return false
	}

	// Determine if all the volume attachments in pod namespace that are attached to the pod
	// Also collect a list of the volumeIDs to be validated.
	volIDs := make([]string, 0)
	vaNamesToDelete := make([]string, 0)
	for _, va := range volumeAttachmentList.Items {
		podVA, err := K8sAPI.IsVolumeAttachmentToPod(ctx, &va, pod)
		if err != nil {
			log.WithFields(fields).Errorf("Aborting cleanup because could not determine if VA %s belongs to pod: %s", va.ObjectMeta.Name, err.Error())
			return false
		}
		if podVA {
			volID, err := K8sAPI.GetVolumeHandleFromVA(ctx, &va)
			if err != nil {
				log.WithFields(fields).Errorf("Aborting cleanup because could not getVolumeHandleFromVA: %v %s", va, err.Error())
				return false
			}
			log.Debugf("VA %s attached to pod %s", va.ObjectMeta.Name, pod.ObjectMeta.Name)
			volIDs = append(volIDs, volID)
			vaNamesToDelete = append(vaNamesToDelete, va.ObjectMeta.Name)
		}
	}

	// Call the driver to validate the volumes are not in use
	if cm.CSIExtensionsPresent && !cm.SkipArrayConnectionValidation {
		if CSIApi.Connected() {
			log.WithFields(fields).Infof("Validating host connectivity for node %s volumes %v", node.ObjectMeta.Name, volIDs)
			connected, iosInProgress, err := cm.callValidateVolumeHostConnectivity(node, volIDs, true)
			if connected || iosInProgress || err != nil {
				log.WithFields(fields).Info("Aborting pod cleanup because array still connected and/or recently did I/O")
				return false
			}
		}
	} else {
		log.WithFields(fields).Infof("Skipped array connection validation")
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
			return false
		}
	}

	// Add a taint for the pod on the node.
	taintNode(node.ObjectMeta.Name, false)

	// Delete all the volumeattachments attached to our pod
	for _, vaName := range vaNamesToDelete {
		err = K8sAPI.DeleteVolumeAttachment(ctx, vaName)
		if err != nil {
			log.WithFields(fields).Errorf("Couldn't delete VolumeAttachment: %s", vaName)
			return false
		}
	}

	// Force delete the pod.
	err = K8sAPI.DeletePod(ctx, pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, true)
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
	return false, false, fmt.Errorf("Could not determine CSI NodeID from the node: %s annotations: %s", node.ObjectMeta.Name, csiNodeID)
}

// callControllerUnpublishVolume in the driver, log any messages, return error.
func (cm *PodMonitorType) callControllerUnpublishVolume(node *v1.Node, volumeID string) error {
	var err error
	csiNodeID := getCSINodeIDAnnotation(node, cm.DriverPathStr)
	if csiNodeID == "" {
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

// podToArrayIDs returns the array IDs used by the pod)
// TODO: multi-array
func (cm *PodMonitorType) podToArrayIDs(pod *v1.Pod) ([]string, error) {
	arrayIDs := make([]string, 1)
	arrayIDs[0] = "default"
	return arrayIDs, nil
}

// ArrayConnectivityPollRate is the rate it polls to check array connectivity to nodes.
var ArrayConnectivityPollRate = ShortTimeout

// ArrayConnectivityMonitor -- periodically checks array connectivity to all the nodes using it.
// If connectivity is lost, will initiate cleanup of the pods.
// This is a never ending function, intended to be called as Go routine.
func (cm *PodMonitorType) ArrayConnectivityMonitor(pollRate time.Duration) {
	// Loop through all the monitored Pods making sure they still have array access
	for {
		// Clear the connectivity cache so it will sample again.
		connectivityCache.ResetSampled()
		// Internal function for iterating PodKeyToControllerPodInfo
		// This will clean up Pods that have lost connectivity to at least one of their arrays
		fnPodKeyToControllerPodInfo := func(key, value interface{}) bool {
			controllerPodInfo := value.(*ControllerPodInfo)
			podKey := controllerPodInfo.PodKey
			podNamespace, podName := splitPodKey(podKey)
			podUID := controllerPodInfo.PodUID
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
				// Fetch the pod.
				ctx, cancel := K8sAPI.GetContext(MediumTimeout)
				defer cancel()
				pod, err := K8sAPI.GetPod(ctx, podNamespace, podName)
				if err == nil {
					if string(pod.ObjectMeta.UID) == podUID && pod.Spec.NodeName == node.ObjectMeta.Name {
						log.Infof("Cleaning up pod %s/%s because of array connectivity loss", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
						cm.controllerCleanupPod(pod, node)
					} else {
						log.Infof("Skipping pod %s/%s podUID %s %s node %s %s", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name,
							string(pod.ObjectMeta.UID), podUID, pod.Spec.NodeName, node.ObjectMeta.Name)
					}
				}
			}
			return true
		}
		cm.PodKeyToControllerPodInfo.Range(fnPodKeyToControllerPodInfo)
		time.Sleep(pollRate)
		if pollRate < 10*time.Millisecond {
			// unit testing exit
			return
		}
	}
}

type nodeArrayConnectivityCache struct {
	initOnce                       sync.Once       // Will be set after initialization
	nodeArrayConnectivitySampled   map[string]bool // If true, already sampled, if need to call array to verify connectivity
	nodeArrayConnectivityLossCount map[string]int  // 0 means connected, > 0 number of connection loss for n samples
}

var connectivityCache nodeArrayConnectivityCache

//ArrayConnectivityConnectionLossThreshold is the number of consecutive samples that must fail before we declare connectivity loss
var ArrayConnectivityConnectionLossThreshold int = 3

// CheckConnectivity returns true if the node has connectivity to the arrayID supplied
func (nacc *nodeArrayConnectivityCache) CheckConnectivity(cm *PodMonitorType, node *v1.Node, arrayID string) bool {
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
		log.Debugf("Node annotations: %s", annotations)
		// Get the csi.volume.kubernetes.io/nodeid annotation
		csiAnnotations := annotations["csi.volume.kubernetes.io/nodeid"]
		if csiAnnotations != "" {
			log.Debugf("csiAnnotations: %s", csiAnnotations)
			var csiAnnotationsMap map[string]json.RawMessage
			err := json.Unmarshal([]byte(csiAnnotations), &csiAnnotationsMap)
			if err != nil {
				log.Errorf("could not unmarshal csi annotations to json: %s", err.Error())
				return ""
			}
			var nodeID string
			err = json.Unmarshal(csiAnnotationsMap[driverPath], &nodeID)
			if err != nil {
				log.Errorf("could not unmarshal driver path key from nodeid annotation: %s", err.Error())
				return ""
			}

			log.Debugf("Returning CSI Node ID Annotation: %s", nodeID)
			return nodeID
		}
	}
	return ""
}

//KubectlTaint is a reference to a function that can update a node taint
var KubectlTaint = callKubectlTaint

func callKubectlTaint(operation, nodeName, taint string) error {
	cmd := exec.Command("/usr-bin/kubectl", "taint", "node", nodeName, taint)
	log.Infof("%s: %s", operation, cmd.String())
	output, err := cmd.Output()
	log.Infof("taint output: %s", string(output))
	if err != nil {
		log.Infof("taint failed: %s", err.Error())
	}
	return err
}

// taintNodeForPod uses kubectl as a safer way of adding/removing the taint instead of updating or patching the Node object.
func taintNode(nodeName string, removeTaint bool) error {
	operation := "tainting "
	if removeTaint {
		operation = "untainting "
	}
	removeFlag := ""
	if removeTaint {
		removeFlag = "-"
	}
	taint := fmt.Sprintf(podmonTaint, "NoSchedule", removeFlag)
	return KubectlTaint(operation, nodeName, taint)
}

func nodeHasTaint(node *v1.Node, key string, taintEffect v1.TaintEffect) bool {
	for _, taint := range node.Spec.Taints {
		if taint.Key == key && taint.Effect == taintEffect {
			return true
		}
	}
	return false
}
