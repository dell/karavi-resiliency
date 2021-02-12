package monitor

import (
	"context"
	"errors"
	"fmt"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/dell/gofsutil"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
	"os"
	"strings"
	"time"
)

//APICheckInterval interval to wait before calling node API after successful call
var APICheckInterval = NodeAPIInterval

//APICheckRetryTimeout retry wait after failure
var APICheckRetryTimeout = ShortTimeout

//APICheckFirstTryTimeout retry wait after the first failure
var APICheckFirstTryTimeout = MediumTimeout

//APIMonitorWait a function reference that can control the API monitor loop
var APIMonitorWait = internalAPIMonitorWait

// StartAPIMonitor checks API connectivity by pinging the indicated (self) node
func StartAPIMonitor() error {
	nodeName := os.Getenv("KUBE_NODE_NAME")
	if nodeName == "" {
		err := errors.New("KUBE_NODE_NAME environment variable must be set")
		log.Errorf("%s", err.Error())
		return err
	}

	pm := &PodMonitor
	fn := func() {
		pm.apiMonitorLoop(nodeName)
	}
	// Start a thread for the API monitor
	go fn()
	return nil
}

func (pm *PodMonitorType) apiMonitorLoop(nodeName string) {
	pm.APIConnected = true
	for {
		// Retrieve our Node's state
		node, err := K8sAPI.GetNodeWithTimeout(APICheckFirstTryTimeout, nodeName)
		if err != nil {
			for i := 0; i < 3; i++ {
				time.Sleep(APICheckRetryTimeout)
				_, err = K8sAPI.GetNodeWithTimeout(APICheckRetryTimeout, nodeName)
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
			if nodeHasTaint(node, podmonTaintKey, v1.TaintEffectNoSchedule) {
				pm.nodeModeCleanupPods(node)
			}
		}
		if stopLoop := APIMonitorWait(); stopLoop {
			break
		}
	}
}

func internalAPIMonitorWait() bool {
	time.Sleep(APICheckInterval)
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
	fields := make(map[string]interface{})
	fields["Namespace"] = pod.ObjectMeta.Namespace
	fields["PodName"] = pod.ObjectMeta.Name
	fields["PodUID"] = string(pod.ObjectMeta.UID)
	fields["Node"] = nodeName
	log.WithFields(fields).Infof("nodeModePodHandler")
	if nodeName == pod.Spec.NodeName {
		if eventType == watch.Added || eventType == watch.Modified {
			// If so, record the pod watch object so later we can check status of the mounts
			podInfo := &NodePodInfo{
				Pod:    pod,
				PodUID: string(pod.ObjectMeta.UID),
				Mounts: make([]MountPathVolumeInfo, 0),
			}
			log.WithFields(fields).Infof("podMonitorHandler-node:  message %s reason %s event %v",
				pod.Status.Message, pod.Status.Reason, eventType)
			csiVolumesPath := fmt.Sprintf(CSIVolumePathFormat, string(pod.ObjectMeta.UID))
			log.Infof("csiVolumesPath: %s", csiVolumesPath)
			volumeEntries, err := ioutil.ReadDir(csiVolumesPath)
			if err != nil {
				log.WithFields(fields).Errorf("Couldn't read directory %s: %s", csiVolumesPath, err.Error())
				return err
			}

			for _, volumeEntry := range volumeEntries {
				pvName := volumeEntry.Name()
				log.Debugf("pvName %s", pvName)
				pv, err := K8sAPI.GetPersistentVolume(ctx, pvName)
				if err != nil {
					log.Errorf("Couldn't read PV %s: %s", pvName, err.Error())
				} else {
					volumeID := pv.Spec.CSI.VolumeHandle
					mountPath := csiVolumesPath + "/" + pvName + "/mount"
					mountPathVolumeInfo := MountPathVolumeInfo{
						Path:     mountPath,
						VolumeID: volumeID,
					}
					log.WithFields(fields).Infof("Adding mountPathVolumeInfo %v", mountPathVolumeInfo)
					podInfo.Mounts = append(podInfo.Mounts, mountPathVolumeInfo)
				}
			}

			// Save the podname key to NodePodInfo object. These are used to eventually cleanup.
			pm.PodKeyMap.LoadOrStore(podKey, podInfo)
		}
		if eventType == watch.Deleted {
			// Do not delete a NodePodInfo structure (which is used to cleanup pods)
			// if our node is currently tainted. We could be in a situation where
			// the pod force delete finished and the event propogated while we were cleaning up.
			node, err := K8sAPI.GetNodeWithTimeout(MediumTimeout, nodeName)
			if err == nil && !nodeHasTaint(node, podmonTaintKey, v1.TaintEffectNoSchedule) {
				pm.PodKeyMap.Delete(podKey)
			}
		}
	}
	return nil
}

//MountPathVolumeInfo composes the mount path and volume
type MountPathVolumeInfo struct {
	Path     string
	VolumeID string
}

//NodePodInfo information used for monitoring a node
type NodePodInfo struct { // information we keep on hand about a pod
	Pod    *v1.Pod               // Copy of the pod itself
	PodUID string                // Pod user id
	Mounts []MountPathVolumeInfo // information about a mount
}

// nodeModeCleanupPods attempts cleanup of all the pods that were registered from the pod Watcher nodeModePodHandler
func (pm *PodMonitorType) nodeModeCleanupPods(node *v1.Node) {
	ctx, cancel := K8sAPI.GetContext(MediumTimeout)
	defer cancel()
	// Retrieve the podKeys we've been watching for our node
	removeTaint := true
	podKeys := make([]string, 0)
	podKeysSkipped := make([]string, 0)
	podInfos := make([]*NodePodInfo, 0)
	fn := func(key, value interface{}) bool {
		podKey := key.(string)
		podInfo := value.(*NodePodInfo)

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
	log.Infof("pods skipped for cleanup because still present: %v", podKeysSkipped)
	log.Infof("pods to be cleaned up: %v", podKeys)
	for i := 0; i < len(podKeys); i++ {
		err := pm.nodeModeCleanupPod(podKeys[i], podInfos[i])
		if err != nil {
			// Abort removing the taint since we didn't clean up
			removeTaint = false
		} else {
			// Remove the NodePodInfo structure as it was successfully cleaned up
			pm.PodKeyMap.Delete(podKeys[i])
		}
	}
	// Don't remove the taint if we had an error cleaning up a pod, or we skipped a pod because
	// it was still present. Instead we will do another cleanup cycle.
	if removeTaint && len(podKeysSkipped) == 0 {
		taintNode(node.ObjectMeta.Name, true)
		log.Infof("Cleanup of pods complete: %v", podKeys)
	} else {
		log.Info("Couldn't completely cleanup node- taint not removed- cleanup will be retried, or a manual reboot is advised advised")
	}
}

//RemoveDir reference to a function used to clean up directories
var RemoveDir = os.Remove

func (pm *PodMonitorType) nodeModeCleanupPod(podKey string, podInfo *NodePodInfo) error {
	var returnErr error
	privateMountDir := os.Getenv("X_CSI_PRIVATE_MOUNT_DIR")
	fields := make(map[string]interface{})
	fields["podKey"] = podKey
	podUID := podInfo.PodUID
	fields["podUid"] = podUID
	log.WithFields(fields).Infof("Cleaning up pod")
	for _, mntInfo := range podInfo.Mounts {
		// TODO Add check if path exists, if not skip
		// Call NodeUnpublish volume for mount
		err := pm.callNodeUnpublishVolume(mntInfo.Path, mntInfo.VolumeID)
		if err != nil {
			returnErr = err
		} else if privateMountDir != "" {
			privTarget := privateMountDir + "/" + mntInfo.VolumeID
			err = gofsutil.Unmount(context.Background(), privTarget)
			if err != nil {
				log.Errorf("Could not Umount private target: %s because: %s", privTarget, err.Error())
			}
			// Remove the private mount target to complete the cleanup.
			err = RemoveDir(privTarget)
			if err != nil && !os.IsNotExist(err) {
				log.Errorf("Could not remove private target: %s because: %s", privTarget, err.Error())
				returnErr = err
			}
		}
	}
	return returnErr
}

// callNodeUnpublishVolume in the driver, log any messages, return error.
func (pm *PodMonitorType) callNodeUnpublishVolume(targetPath, volumeID string) error {
	var err error
	for i := 0; i < CSIMaxRetries; i++ {
		log.Infof("Calling NodeUnpublishVolume path %s volume %s", targetPath, volumeID)
		req := &csi.NodeUnpublishVolumeRequest{
			TargetPath: targetPath,
			VolumeId:   volumeID,
		}
		_, err = CSIApi.NodeUnpublishVolume(context.Background(), req)
		if err == nil {
			break
		}
		log.Infof("Error calling NodeUnpublishVolume path %s volume %s: %s", targetPath, volumeID, err.Error())
		if !strings.HasSuffix(err.Error(), "pending") {
			break
		}
		time.Sleep(PendingRetryTime)
	}

	return err
}
