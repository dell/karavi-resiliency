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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"podmon/internal/criapi"
	"podmon/internal/k8sapi"
	"podmon/internal/utils"
	"strings"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/dell/gofsutil"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
	cri "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

//APICheckInterval interval to wait before calling node API after successful call
var APICheckInterval = NodeAPIInterval

//APICheckRetryTimeout retry wait after failure
var APICheckRetryTimeout = ShortTimeout

//APICheckFirstTryTimeout retry wait after the first failure
var APICheckFirstTryTimeout = MediumTimeout

//APIMonitorWait a function reference that can control the API monitor loop
var APIMonitorWait = internalAPIMonitorWait

// TaintCountDelay delays pod cleanup until at least TaintCountDelay iterations of the apiMonitorLoop have executed,
// giving the node time to stabalize (e.g. kubelet reconcile with API server) before initiating cleanup.
var TaintCountDelay = 4

// StartAPIMonitor checks API connectivity by pinging the indicated (self) node
func StartAPIMonitor(api k8sapi.K8sAPI, firstTimeout, retryTimeout, interval time.Duration, waitFor func(interval time.Duration) bool) error {
	nodeName := os.Getenv("KUBE_NODE_NAME")
	if nodeName == "" {
		err := errors.New("KUBE_NODE_NAME environment variable must be set")
		log.Errorf("%s", err.Error())
		return err
	}

	pm := &PodMonitor
	fn := func() {
		pm.apiMonitorLoop(api, nodeName, firstTimeout, retryTimeout, interval, waitFor)
	}
	// Start a thread for the API monitor
	go fn()
	return nil
}

func (pm *PodMonitorType) apiMonitorLoop(api k8sapi.K8sAPI, nodeName string, firstTimeout, retryTimeout, interval time.Duration, waitFor func(interval time.Duration) bool) {
	pm.APIConnected = true
	taintCount := 0
	for {
		// Retrieve our Node's state
		node, err := api.GetNodeWithTimeout(firstTimeout, nodeName)
		if err != nil {
			for i := 0; i < 3; i++ {
				time.Sleep(APICheckRetryTimeout)
				_, err = api.GetNodeWithTimeout(retryTimeout, nodeName)
				if err == nil {
					break
				}
			}
			if err != nil && pm.APIConnected {
				f := map[string]interface{}{
					"NodeID": nodeName,
					"Error":  err.Error(),
				}
				log.WithFields(f).Info("Lost API connectivity from node")
				pm.APIConnected = false
				taintCount = 0
			}
		} else {
			// No Error - we are connected to APIService
			if !pm.APIConnected {
				f := map[string]interface{}{
					"NodeID": nodeName,
				}
				log.WithFields(f).Info("API connectivity restored to node")
				pm.APIConnected = true

			}
			// If our node is tainted, we need to clean it up
			if nodeHasTaint(node, PodmonTaintKey, v1.TaintEffectNoSchedule) {
				taintCount = taintCount + 1
				// Delay the first few intervals if necessary to give the node time to stabalize
				if taintCount >= TaintCountDelay {
					if pm.nodeModeCleanupPods(node) {
						taintCount = 0
					}
				} else {
					log.Infof("Waiting on node to stabalize: %d", taintCount)
				}
			} else {
				if taintCount > 0 {
					log.Error("********** taint manually removed **********")
				}
			}
		}
		if stopLoop := waitFor(interval); stopLoop {
			break
		}
	}
}

func internalAPIMonitorWait(interval time.Duration) bool {
	time.Sleep(interval)
	return false
}

// nodeModePodHandler handles node mode functionality when a pod event happens.
func (pm *PodMonitorType) nodeModePodHandler(pod *v1.Pod, eventType watch.EventType) error {
	ctx, cancel := K8sAPI.GetContext(LongTimeout)
	defer cancel()
	// Copy the pod
	pod = pod.DeepCopy()
	podKey := getPodKey(pod)
	// Check that this pod belongs to our node
	nodeName := os.Getenv("KUBE_NODE_NAME")
	driverNamespace := os.Getenv("MY_POD_NAMESPACE")
	if driverNamespace == pod.ObjectMeta.Namespace {
		//driver pod, no need to protect at node
		return nil
	}

	fields := make(map[string]interface{})
	fields["Namespace"] = pod.ObjectMeta.Namespace
	fields["PodName"] = pod.ObjectMeta.Name
	fields["PodUID"] = string(pod.ObjectMeta.UID)
	fields["Node"] = pod.Spec.NodeName
	fields["EventType"] = eventType
	log.WithFields(fields).Infof("nodeModePodHandler")
	if nodeName == pod.Spec.NodeName {
		if eventType == watch.Added || eventType == watch.Modified {
			// If so, record the pod watch object so later we can check status of the mounts
			podInfo := &NodePodInfo{
				Pod:     pod,
				PodUID:  string(pod.ObjectMeta.UID),
				Mounts:  make([]MountPathVolumeInfo, 0),
				Devices: make([]BlockPathVolumeInfo, 0),
			}
			log.WithFields(fields).Infof("podMonitorHandler-node:  message %s reason %s event %v",
				pod.Status.Message, pod.Status.Reason, eventType)

			// See if there is already an entry, and if so, make sure we don't save result if fewer devices
			var existingVolumeCount int
			var existingDeviceCount int
			existingEntry, ok := pm.PodKeyMap.Load(podKey)
			if ok {
				nodePodInfo := existingEntry.(*NodePodInfo)
				existingVolumeCount = len(nodePodInfo.Mounts)
				existingDeviceCount = len(nodePodInfo.Devices)
			}

			// Scan for mounts
			csiVolumesPath := fmt.Sprintf(CSIVolumePathFormat, string(pod.ObjectMeta.UID))
			log.Debugf("csiVolumesPath: %s", csiVolumesPath)
			volumeEntries, err := ioutil.ReadDir(csiVolumesPath)
			if err != nil && !os.IsNotExist(err) {
				log.WithFields(fields).Errorf("Couldn't read directory %s: %s", csiVolumesPath, err.Error())
				return err
			}

			for _, volumeEntry := range volumeEntries {
				pvName := volumeEntry.Name()
				log.Debugf("mount pvName %s", pvName)
				pv, err := K8sAPI.GetPersistentVolume(ctx, pvName)
				if err != nil {
					log.Errorf("Couldn't read mount PV %s: %s", pvName, err.Error())
				} else {
					volumeID := pv.Spec.CSI.VolumeHandle
					mountPath := csiVolumesPath + "/" + pvName + "/mount"
					mountPathVolumeInfo := MountPathVolumeInfo{
						Path:     mountPath,
						VolumeID: volumeID,
						PVName:   pvName,
					}
					log.WithFields(fields).Infof("Adding mountPathVolumeInfo %v", mountPathVolumeInfo)
					podInfo.Mounts = append(podInfo.Mounts, mountPathVolumeInfo)
				}
			}

			// Scan for block devices
			csiDevicesPath := fmt.Sprintf(CSIDevicePathFormat, string(pod.ObjectMeta.UID))
			log.Infof("csiDevicesPath: %s", csiDevicesPath)
			deviceEntries, err := ioutil.ReadDir(csiDevicesPath)
			if err != nil && !os.IsNotExist(err) {
				log.WithFields(fields).Errorf("Couldn't read directory %s: %s", csiDevicesPath, err.Error())
				return err
			}
			for _, deviceEntry := range deviceEntries {
				pvName := deviceEntry.Name()
				log.Debugf("dev pvName %s", pvName)
				pv, err := K8sAPI.GetPersistentVolume(ctx, pvName)
				if err != nil {
					log.Errorf("Couldn't read block PV %s: %s", pvName, err.Error())
				} else {
					volumeID := pv.Spec.CSI.VolumeHandle
					mountPath := csiDevicesPath + "/" + pvName
					blockPathVolumeInfo := BlockPathVolumeInfo{
						Path:     mountPath,
						VolumeID: volumeID,
						PVName:   pvName,
					}
					log.WithFields(fields).Infof("Add blockPathVolumeInfo %v", blockPathVolumeInfo)
					podInfo.Devices = append(podInfo.Devices, blockPathVolumeInfo)
				}
			}

			// Save the podname key to NodePodInfo object. These are used to eventually cleanup.
			// Don't save an entry if the volume or device counts are lower than what we already have
			if len(podInfo.Mounts) >= existingVolumeCount && len(podInfo.Devices) >= existingDeviceCount {
				log.WithFields(fields).Infof("Storing podInfo %d mounts %d devices", len(podInfo.Mounts), len(podInfo.Devices))
				pm.PodKeyMap.Store(podKey, podInfo)
			} else {
				log.WithFields(fields).Infof("Skipped Storing podInfo %d mounts %d devices", len(podInfo.Mounts), len(podInfo.Devices))
			}
		}
		if eventType == watch.Deleted {
			// Do not delete a NodePodInfo structure (which is used to cleanup pods)
			// if our node is currently tainted. We could be in a situation where
			// the pod force delete finished and the event propogated while we were cleaning up.
			node, err := K8sAPI.GetNodeWithTimeout(MediumTimeout, nodeName)
			if err == nil && !nodeHasTaint(node, PodmonTaintKey, v1.TaintEffectNoSchedule) {
				pm.PodKeyMap.Delete(podKey)
			}
		}
	}
	return nil
}

//MountPathVolumeInfo holds the mount path and volume information
type MountPathVolumeInfo struct {
	Path     string
	VolumeID string
	PVName   string
}

//BlockPathVolumeInfo holds the block path and volume information
type BlockPathVolumeInfo struct {
	Path     string
	VolumeID string
	PVName   string
}

//NodePodInfo information used for monitoring a node
type NodePodInfo struct { // information we keep on hand about a pod
	Pod     *v1.Pod               // Copy of the pod itself
	PodUID  string                // Pod user id
	Mounts  []MountPathVolumeInfo // information about a mount
	Devices []BlockPathVolumeInfo // information about raw block devices
}

// nodeModeCleanupPods attempts cleanup of all the pods that were registered from the pod Watcher nodeModePodHandler
// Returns true if taint was removed, false if taint should remain.
func (pm *PodMonitorType) nodeModeCleanupPods(node *v1.Node) bool {
	crictx, cricancel := K8sAPI.GetContext(ShortTimeout)
	defer cricancel()
	// Using CRI, get the pod information
	containerInfos, err := getContainers(crictx)
	if err != nil {
		log.Errorf("Could not get container information: %s", err)
	} else {
		for _, value := range containerInfos {
			log.Infof("ContainerInfo %+v\n", *value)
		}
	}
	// Retrieve the podKeys we've been watching for our node
	ctx, cancel := K8sAPI.GetContext(MediumTimeout)
	defer cancel()
	removeTaint := true
	podKeys := make([]string, 0)
	podKeysSkipped := make([]string, 0)
	podKeysWithError := make([]string, 0)
	podInfos := make([]*NodePodInfo, 0)
	// This function executed for each registered pod to categorize it a) to be cleaned, or b) skipped because it is possibly executing
	fn := func(key, value interface{}) bool {
		podKey := key.(string)
		podInfo := value.(*NodePodInfo)

		// Check containers to make sure they're not running. This uses the containerInfos map obtained above.
		pod := podInfo.Pod
		// Get the PVs associated with this pod.
		pvlist, err := K8sAPI.GetPersistentVolumesInPod(ctx, pod)
		if err == nil && IgnoreVolumelessPods && len(pvlist) == 0 {
			log.Infof("IgnoreVolumelessPods %t pvc count %d", IgnoreVolumelessPods, len(pvlist))
			return true
		}
		for _, containerStatus := range pod.Status.ContainerStatuses {
			containerID := containerStatus.ContainerID
			cid := strings.Split(containerID, "//")
			if len(cid) > 1 && containerInfos[cid[1]] != nil {
				log.Debugf("cid %v", cid[1])
				containerInfo := containerInfos[cid[1]]
				if containerInfo.State == cri.ContainerState_CONTAINER_RUNNING || containerInfo.State == cri.ContainerState_CONTAINER_CREATED {
					log.Infof("Skipping pod %s because container %v still executing", podKey, containerInfo)
					podKeysSkipped = append(podKeysSkipped, podKey)
					return true
				}
			}
		}

		// Check to make sure the pod has been deleted, or still exists
		namespace, name := splitPodKey(podKey)
		currentPod, err := K8sAPI.GetPod(ctx, namespace, name)
		if err == nil {
			// We retrieve a pod for the namespace/name... see if same one
			currentUID := string(currentPod.ObjectMeta.UID)
			if currentUID == podInfo.PodUID {
				// same pod UID still exists, so we cannot clean it up
				// it may not have been successfully processed on controller podmon
				podKeysSkipped = append(podKeysSkipped, podKey)
				return true
			}
		} else {
			log.Infof("Could not retrieve pod %s: %s", podKey, err.Error())
		}
		// Add pod to list to be cleaned up
		podKeys = append(podKeys, podKey)
		podInfos = append(podInfos, podInfo)
		return true
	}
	pm.PodKeyMap.Range(fn)
	log.Infof("pods skipped for cleanup because still present or container executing: %v", podKeysSkipped)
	log.Infof("pods to be cleaned up: %v", podKeys)
	for i := 0; i < len(podKeys); i++ {
		err := pm.nodeModeCleanupPod(podKeys[i], podInfos[i])
		if err != nil {
			podKeysWithError = append(podKeysWithError, podKeys[i])
			// Abort removing the taint since we didn't clean up
			removeTaint = false
		} else {
			// Remove the NodePodInfo structure as it was successfully cleaned up
			pm.PodKeyMap.Delete(podKeys[i])
		}
	}
	// Don't remove the taint if we had an error cleaning up a pod, or we skipped a pod because
	// it was still present. Instead we will do another cleanup cycle.
	if removeTaint && len(podKeysSkipped) == 0 && len(podKeysWithError) == 0 {
		if err := taintNode(node.ObjectMeta.Name, PodmonTaintKey, true); err != nil {
			log.Errorf("Failed to remove taint against %s node: %v", node.ObjectMeta.Name, err)
			return false
		}
		log.Infof("Cleanup of pods complete: %v", podKeys)
		return true
	}

	log.Infof("pods skipped for cleanup because still present or container executing: %v", podKeysSkipped)
	log.Infof("pods with cleanup errors: %v", podKeysWithError)
	log.Info("Couldn't completely cleanup node- taint not removed- cleanup will be retried, or a manual reboot is advised")
	return false
}

//RemoveDir reference to a function used to clean up directories
var RemoveDir = os.Remove

//RemoveDev reference to a function used to remove devices
var RemoveDev = os.Remove

func (pm *PodMonitorType) nodeModeCleanupPod(podKey string, podInfo *NodePodInfo) error {
	var returnErr error
	fields := make(map[string]interface{})
	fields["podKey"] = podKey
	podUID := podInfo.PodUID
	fields["podUid"] = podUID
	log.WithFields(fields).Infof("Cleaning up pod")

	// Clean up volume mounts
	for _, mntInfo := range podInfo.Mounts {
		// TODO Add check if path exists, if not skip
		// Call NodeUnpublish volume for mount
		err := pm.callNodeUnpublishVolume(fields, mntInfo.Path, mntInfo.VolumeID)
		if err != nil && !Driver.NodeUnpublishExcludedError(err) {
			log.WithFields(fields).Errorf("NodeUnpublishVolume failed: %s %s %s", mntInfo.Path, mntInfo.VolumeID, err)
			returnErr = err
		} else {
			stagingDir := Driver.GetStagingMountDir(mntInfo.VolumeID, mntInfo.PVName)
			if stagingDir != "" {
				err = pm.callNodeUnstageVolume(fields, stagingDir, mntInfo.VolumeID)
				if err != nil && !Driver.NodeUnstageExcludedError(err) {
					log.WithFields(fields).Errorf("NodeUnstageVolume failed: %s %s %s", mntInfo.Path, mntInfo.VolumeID, err)
					returnErr = err
				}
			}

			privTarget := Driver.GetDriverMountDir(mntInfo.VolumeID, mntInfo.PVName, podUID)
			err = gofsutil.Unmount(context.Background(), privTarget)
			if err != nil {
				log.WithFields(fields).Errorf("Could not Unmount private target: %s because: %s", privTarget, err.Error())
			}
			// Remove the private mount target to complete the cleanup.
			err = RemoveDir(privTarget)
			if err != nil && !os.IsNotExist(err) {
				log.WithFields(fields).Errorf("Could not remove private target: %s because: %s", privTarget, err.Error())
				returnErr = err
			}
			// Do final driver cleanup if any.
			err = Driver.FinalCleanup(false, mntInfo.VolumeID, mntInfo.PVName, podUID)
			if err != nil {
				log.WithFields(fields).Errorf("FinalCleanup failed: %s", err)
				returnErr = err
			}
		}
	}

	// Clean up raw block devices
	for _, devInfo := range podInfo.Devices {
		// Call Node unpublish for block device
		err := pm.callNodeUnpublishVolume(fields, devInfo.Path, devInfo.VolumeID)
		if err != nil && !Driver.NodeUnpublishExcludedError(err) {
			log.WithFields(fields).Errorf("NodeUnpublishVolume failed: %s %s %s", devInfo.Path, devInfo.VolumeID, err)
			returnErr = err
		} else {
			stagingDir := Driver.GetStagingBlockDir(devInfo.VolumeID, devInfo.PVName)
			if stagingDir != "" {
				err = pm.callNodeUnstageVolume(fields, stagingDir, devInfo.VolumeID)
				if err != nil && !Driver.NodeUnstageExcludedError(err) {
					log.WithFields(fields).Errorf("NodeUnstageVolume failed: %s %s %s", devInfo.Path, devInfo.VolumeID, err)
					returnErr = err
				}
			}

			privBlockDev := Driver.GetDriverBlockDev(devInfo.VolumeID, devInfo.PVName, podUID)
			err = utils.Unmount(privBlockDev, 0)
			if err != nil {
				log.WithFields(fields).Errorf("Could not Unmount private block device: %s because: %s", privBlockDev, err.Error())
			}
			// Remove the block device to complete the cleanup
			err = RemoveDev(privBlockDev)
			if err != nil && !os.IsNotExist(err) {
				log.WithFields(fields).Errorf("Could not remove block device: %s because: %s", privBlockDev, err.Error())
				returnErr = err
			}
			// Do final driver cleanup if any.
			err = Driver.FinalCleanup(true, devInfo.VolumeID, devInfo.PVName, podUID)
			if err != nil {
				log.WithFields(fields).Errorf("FinalCleanup failed: %s", err)
				returnErr = err
			}
		}
	}
	if returnErr != nil {
		log.WithFields(fields).Errorf("Pod cleanup failed, reason: %s", returnErr.Error())
	}

	if returnErr != nil {
		log.WithFields(fields).Errorf("Pod cleanup failed, reason: %s", returnErr.Error())
	} else {
		log.WithFields(fields).Infof("Pod cleanup complete")
	}

	return returnErr
}

// callNodeUnpublishVolume in the driver, log any messages, return error.
func (pm *PodMonitorType) callNodeUnpublishVolume(fields map[string]interface{}, targetPath, volumeID string) error {
	var err error
	for i := 0; i < CSIMaxRetries; i++ {
		log.WithFields(fields).Infof("Calling NodeUnpublishVolume path %s volume %s", targetPath, volumeID)
		req := &csi.NodeUnpublishVolumeRequest{
			TargetPath: targetPath,
			VolumeId:   volumeID,
		}
		_, err = CSIApi.NodeUnpublishVolume(context.Background(), req)
		if err == nil {
			break
		}
		log.WithFields(fields).Infof("Error calling NodeUnpublishVolume path %s volume %s: %s", targetPath, volumeID, err.Error())
		if !strings.HasSuffix(err.Error(), "pending") {
			break
		}
		time.Sleep(PendingRetryTime)
	}

	return err
}

// callNodeUnStageVolume in the driver, log any messages, return error.
func (pm *PodMonitorType) callNodeUnstageVolume(fields map[string]interface{}, targetPath, volumeID string) error {
	var err error
	for i := 0; i < CSIMaxRetries; i++ {
		log.WithFields(fields).Infof("Calling NodeUnstageVolume path %s volume %s", targetPath, volumeID)
		req := &csi.NodeUnstageVolumeRequest{
			StagingTargetPath: targetPath,
			VolumeId:          volumeID,
		}
		_, err = CSIApi.NodeUnstageVolume(context.Background(), req)
		if err == nil {
			break
		}
		log.WithFields(fields).Infof("Error calling NodeUnstageVolume path %s volume %s: %s", targetPath, volumeID, err.Error())
		if !strings.HasSuffix(err.Error(), "pending") {
			break
		}
		time.Sleep(PendingRetryTime)
	}
	return err
}

var getContainers = criapi.CRIClient.GetContainerInfo
