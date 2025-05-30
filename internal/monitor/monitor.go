// Copyright © 2021-2023 Dell Inc. or its subsidiaries. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package monitor

import (
	"context"
	"fmt"
	"podmon/internal/csiapi"
	"podmon/internal/k8sapi"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

const (
	nodeUnreachableTaint    = "node.kubernetes.io/unreachable"
	podReadyCondition       = "Ready"
	podInitializedCondition = "Initialized"
	podmon                  = "podmon"
	crashLoopBackOffReason  = "CrashLoopBackOff"
	// PodmonTaintKeySuffix is used for creating a driver specific podmon taint key
	PodmonTaintKeySuffix = "podmon.storage.dell.com"
	// PodmonDriverPodTaintKeySuffix is used for creating a driver node pod specific podmon taint key
	PodmonDriverPodTaintKeySuffix = "storage.dell.com"
)

var (
	// PodmonTaintKey is the key for this driver's podmon taint.
	PodmonTaintKey = ""
	// PodmonDriverPodTaintKey is the key for this driver's node pod taint.
	PodmonDriverPodTaintKey = ""
	// ShortTimeout used for initial try
	ShortTimeout = 10 * time.Second
	// MediumTimeout is a wait-backoff after the ShortTimeout
	MediumTimeout = 30 * time.Second
	// LongTimeout is a longer wait-backoff period
	LongTimeout = 180 * time.Second
	// PendingRetryTime time between retry of certain CSI calls
	PendingRetryTime = 30 * time.Second
	// NodeAPIInterval time between NodeAPI checks
	NodeAPIInterval = 30 * time.Second
	// CSIMaxRetries max times to retry certain CSI calls
	CSIMaxRetries = 3
	// MonitorRestartTimeDelay time to wait before restarting monitor
	MonitorRestartTimeDelay = 10 * time.Second
	// LockSleepTimeDelay wait for lock retry
	LockSleepTimeDelay = 1 * time.Second
	// dynamicConfigUpdateMutex protects concurrently running threads that could be affected by dynamic configuration parameters.
	dynamicConfigUpdateMutex sync.Mutex
	// arrayConnectivityPollRate is the rate it polls to check array connectivity to nodes.
	arrayConnectivityPollRate = ShortTimeout
	// IgnoreVolumelessPods when set will keep labeled pods with no volumes from being force deleted on node or connectivity failures.
	IgnoreVolumelessPods bool
)

// GetArrayConnectivityPollRate returns the array connectivity poll rate.
func GetArrayConnectivityPollRate() time.Duration {
	dynamicConfigUpdateMutex.Lock()
	defer dynamicConfigUpdateMutex.Unlock()
	return arrayConnectivityPollRate
}

// SetArrayConnectivityPollRate sets the array connectivity poll rate.
func SetArrayConnectivityPollRate(rate time.Duration) {
	dynamicConfigUpdateMutex.Lock()
	defer dynamicConfigUpdateMutex.Unlock()
	arrayConnectivityPollRate = rate
}

// PodMonitorType structure is tracking data for the pod monitor
type PodMonitorType struct {
	Mode                          string   // controller, node, or standalone
	PodKeyMap                     sync.Map // podkey to *v1.Pod in controller (temporal) or *NodePodInfo in node
	PodKeyToControllerPodInfo     sync.Map // podkey to *ControllerPodInfo in controller
	PodKeyToCrashLoopBackOffCount sync.Map // podkey to CrashLoopBackOffCount
	APIConnected                  bool     // connected to k8s API
	ArrayConnected                bool     // node is connected to array
	SkipArrayConnectionValidation bool     // skip validation array connection lost
	CSIExtensionsPresent          bool     // the CSI PodmonExtensions are present
	DriverPathStr                 string   // CSI Driver path string for parsing csi.volume.kubernetes.io/nodeid annotation
	NodeNameToUID                 sync.Map // Node.ObjectMeta.Name to Node.ObjectMeta.Uid
}

// PodMonitor is a reference to tracking data for the pod monitor
var PodMonitor PodMonitorType

// K8sAPI is a reference to the internal k8s API wrapper
var K8sAPI k8sapi.K8sAPI

// CSIApi is a reference to the internal CSI API wrapper
var CSIApi csiapi.CSIApi

// CSIVolumePathFormat is a formatter string used for producing the full path of a volume mount
var CSIVolumePathFormat = "/var/lib/kubelet/pods/%s/volumes/kubernetes.io~csi"

// CSIDevicePathFormat is a formatter string used for producing the full path of a block volume
var CSIDevicePathFormat = "/var/lib/kubelet/pods/%s/volumeDevices/kubernetes.io~csi"

func getPodKey(pod *v1.Pod) string {
	return pod.ObjectMeta.Namespace + "/" + pod.ObjectMeta.Name
}

// Returns the namespace, name from the pod key
func splitPodKey(podKey string) (string, string) {
	parts := strings.Split(podKey, "/")
	return parts[0], parts[1]
}

// WatchFunc will receive a callback if there is a Watch Event.
// eventType is Added, Modified, Deleted, Bookmark, or Error.
type WatchFunc func(eventType watch.EventType, object interface{}) error

// Monitor sets up a Watch on a Pod or other object by calling the Watch function.
type Monitor struct {
	Client   kubernetes.Interface
	StopChan chan bool
	Watcher  watch.Interface
}

// Lock acquires a sync lock based on the pod reference and key
func Lock(podkey string, pod *v1.Pod, delay time.Duration) {
	_, loaded := PodMonitor.PodKeyMap.LoadOrStore(podkey, pod)
	for loaded {
		time.Sleep(delay)
		_, loaded = PodMonitor.PodKeyMap.LoadOrStore(podkey, pod)
	}
}

// Unlock returns a sync lock based on a pod key reference
func Unlock(podkey string) {
	PodMonitor.PodKeyMap.Delete(podkey)
}

// StoreNodeUID store node name and UID
func (pm *PodMonitorType) StoreNodeUID(nodeName, uid string) {
	log.Debugf("StoreNodeUid added node: %s uid: %s", nodeName, uid)
	pm.NodeNameToUID.Store(nodeName, uid)
}

// GetNodeUID returns the node UID
func (pm *PodMonitorType) GetNodeUID(nodeName string) string {
	log.Debugf("GetNodeUid added node: %s", nodeName)
	value, ok := pm.NodeNameToUID.Load(nodeName)
	if !ok {
		return ""
	}
	uid := fmt.Sprintf("%v", value)
	return uid
}

// ClearNodeUID remove the node name UID on node
func (pm *PodMonitorType) ClearNodeUID(nodeName, oldNodeUID string) bool {
	return pm.NodeNameToUID.CompareAndSwap(nodeName, oldNodeUID, "")
}

// Watch watches a watcher result channel. Upon receiving an event, the fn WatchFunc is invoked.Load
func (pm *Monitor) Watch(fn WatchFunc) error {
	defer pm.Watcher.Stop()
	for {
		select {
		case event, more := <-pm.Watcher.ResultChan():
			log.Debugf("event received")
			if !more {
				return fmt.Errorf("watcher disconnected")
			}
			if event.Object != nil {
				log.Debugf("received object %+v", event.Object)
				err := fn(event.Type, event.Object)
				if err != nil {
					log.Error(err)
				}
			}
		case stop := <-pm.StopChan:
			if stop == true {
				return nil
			}
		}
	}
}

func podMonitorHandler(eventType watch.EventType, object interface{}) error {
	log.Debugf("podMonitorHandler %s eventType %+v object %+v", PodMonitor.Mode, eventType, object)
	pod, ok := object.(*v1.Pod)
	if !ok || pod == nil {
		log.Info("podMonitorHandler nil pod")
		return nil
	}
	pm := &PodMonitor
	switch PodMonitor.Mode {
	case "controller":
		if err := pm.controllerModePodHandler(pod, eventType); err != nil {
			return err
		}
	case "standalone":
		if err := pm.controllerModePodHandler(pod, eventType); err != nil {
			return err
		}
	case "node":
		if err := pm.nodeModePodHandler(pod, eventType); err != nil {
			return err
		}
	default:
		log.Error("PodMonitor.Mode not set")
	}
	return nil
}

// StartPodMonitor starts the PodMonitor so that it is processing pods which might have problems.
// The labelKey and labelValue are used for filtering.
func StartPodMonitor(api k8sapi.K8sAPI, client kubernetes.Interface, labelKey, labelValue string, restartDelay time.Duration) {
	log.Infof("attempting to start PodMonitor\n")
	PodmonTaintKey = fmt.Sprintf("%s.%s", Driver.GetDriverName(), PodmonTaintKeySuffix)
	PodmonDriverPodTaintKey = fmt.Sprintf("offline.%s.%s", Driver.GetDriverName(), PodmonDriverPodTaintKeySuffix)
	podMonitor := Monitor{Client: client}
	listOptions := metav1.ListOptions{
		Watch: true,
	}
	if labelKey != "" {
		labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{labelKey: labelValue}}
		log.Infof("labelSelector: %v\n", labelSelector)
		listOptions.LabelSelector = labels.Set(labelSelector.MatchLabels).String()
	}
	for {
		ctx := context.Background()
		watcher, err := api.SetupPodWatch(ctx, "", listOptions)
		if err != nil {
			// The following check excludes unit testing, to avoid polluting the log messages captured
			if restartDelay > 10*time.Millisecond {
				log.Errorf("Could not create PodWatcher: %s - will retry\n", err)
			}
			time.Sleep(restartDelay)
			continue
		}
		podMonitor.Watcher = watcher
		log.Infof("Setup of PodWatcher complete\n")
		err = podMonitor.Watch(podMonitorHandler)
		if err == nil {
			// requested stop
			return
		}
		log.Warnf("PodWatcher stopped... attempting restart: %s", err)
		time.Sleep(restartDelay)
	}
}

func nodeMonitorHandler(eventType watch.EventType, object interface{}) error {
	node, ok := object.(*v1.Node)
	if !ok {
		log.Info("nodeMonitorHandler nil node")
		return nil
	}
	oldUID := string(node.ObjectMeta.UID)
	if node != nil {
		pm := &PodMonitor
		switch eventType {
		case watch.Added:
			log.Debugf("Node created: %s %s", node.ObjectMeta.Name, node.ObjectMeta.UID)
			pm.StoreNodeUID(node.ObjectMeta.Name, string(node.ObjectMeta.UID))
		case watch.Modified:
			log.Debugf("Node updated: %s %s", node.ObjectMeta.Name, node.ObjectMeta.UID)
			pm.StoreNodeUID(node.ObjectMeta.Name, string(node.ObjectMeta.UID))
		case watch.Deleted:
			log.Debugf("Node deleted: %s previously %s", node.ObjectMeta.Name, oldUID)
			pm.ClearNodeUID(node.ObjectMeta.Name, oldUID)
		}
		// Get the CSI annotations for nodeID
		volumeIDs := make([]string, 0)
		// Print out whether the host is connected or not...
		_, _, _ = pm.callValidateVolumeHostConnectivity(node, volumeIDs, true)

		// Determine if the node is tainted
		taintnosched := nodeHasTaint(node, nodeUnreachableTaint, v1.TaintEffectNoSchedule)
		taintnoexec := nodeHasTaint(node, nodeUnreachableTaint, v1.TaintEffectNoExecute)

		log.Infof("node name: %s uid %s nosched %t noexec %t", node.ObjectMeta.Name, string(node.ObjectMeta.UID), taintnosched, taintnoexec)
	}
	return nil
}

// StartNodeMonitor starts the NodeMonitor so that it is process nodes which might go offline.
func StartNodeMonitor(api k8sapi.K8sAPI, client kubernetes.Interface, labelKey, labelValue string, restartDelay time.Duration) {
	log.Printf("attempting to start NodeMonitor\n")
	nodeMonitor := Monitor{Client: client}
	listOptions := metav1.ListOptions{
		Watch: true,
	}
	if labelKey != "" {
		labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{labelKey: labelValue}}
		log.Infof("labelSelector: %v\n", labelSelector)
		listOptions.LabelSelector = labels.Set(labelSelector.MatchLabels).String()
	}
	for {
		ctx := context.Background()
		watcher, err := api.SetupNodeWatch(ctx, listOptions)
		if err != nil {
			// The following check excludes unit testing, to avoid polluting the log messages captured
			if restartDelay > 10*time.Millisecond {
				log.Errorf("Could not create NodeWatcher: %s - will retry\n", err)
			}
			time.Sleep(restartDelay)
			continue
		}
		nodeMonitor.Watcher = watcher
		log.Infof("Setup of NodeWatcher complete\n")
		err = nodeMonitor.Watch(nodeMonitorHandler)

		if err == nil {
			// requested stop
			return
		}
		log.Errorf("NodeWatcher stopped... attempting restart: %s", err)
		time.Sleep(restartDelay)
	}
}
