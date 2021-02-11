package monitor

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"podmon/internal/csiapi"
	"podmon/internal/k8sapi"
	"strings"
	"sync"
	"time"
)

const (
	podmonTaintKey          = "podmon.dellemc.com"
	podmonTaint             = "podmon.dellemc.com:%s%s" // typically podmon.dellemc.com:NoExecute
	nodeUnreachableTaint    = "node.kubernetes.io/unreachable"
	podReadyCondition       = "Ready"
	podInitializedCondition = "Initialized"
)

var (
	ShortTimeout            = 10 * time.Second
	MediumTimeout           = 30 * time.Second
	LongTimeout             = 180 * time.Second
	PendingRetryTime        = 30 * time.Second
	NodeAPIInterval         = 30 * time.Second
	CSIMaxRetries           = 3
	MonitorRestartTimeDelay = 10 * time.Second
	LockSleepTimeDelay      = 1 * time.Second
)

type PodMonitorType struct {
	Mode                          string   // controller, node, or standalone
	PodKeyMap                     sync.Map // podkey to *v1.Pod in controller (temporal) or *NodePodInfo in node
	PodKeyToControllerPodInfo     sync.Map // podkey to *ControllerPodInfo in controller
	APIConnected                  bool     // connected to k8s API
	ArrayConnected                bool     // node is connected to array
	SkipArrayConnectionValidation bool     // skip validation array connection lost
	CSIExtensionsPresent          bool     // the CSI PodmonExtensions are present
	DriverPathStr                 string   // CSI Driver path string for parsing csi.volume.kubernetes.io/nodeid annotation
}

var PodMonitor PodMonitorType
var K8sApi k8sapi.K8sApi
var CSIApi csiapi.CSIApi
var CSIVolumePathFormat = "/var/lib/kubelet/pods/%s/volumes/kubernetes.io~csi"

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

func Lock(podkey string, pod *v1.Pod) {
	_, loaded := PodMonitor.PodKeyMap.LoadOrStore(podkey, pod)
	for loaded {
		time.Sleep(LockSleepTimeDelay)
		_, loaded = PodMonitor.PodKeyMap.LoadOrStore(podkey, pod)
	}
}

func Unlock(podkey string) {
	PodMonitor.PodKeyMap.Delete(podkey)
}

// Watch watches a watcher result channel. Upon receiving an event, the fn WatchFunc is invoked.
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
		pm.controllerModePodHandler(pod, eventType)
	case "standalone":
		pm.controllerModePodHandler(pod, eventType)
	case "node":
		pm.nodeModePodHandler(pod, eventType)
	default:
		log.Error("PodMonitor.Mode not set")
	}
	return nil
}

// Start the PodMonitor so that it is processing pods which might have problems.
// The labelKey and labelValue are used for filtering.
func StartPodMonitor(client kubernetes.Interface, labelKey, labelValue string) {
	log.Infof("attempting to start PodMonitor\n")
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
		watcher, err := K8sApi.SetupPodWatch(ctx, "", listOptions)
		if err != nil {
			log.Errorf("Could not create PodWatcher: %s\n", err)
			return
		}
		podMonitor.Watcher = watcher
		log.Infof("Setup of PodWatcher complete\n")
		err = podMonitor.Watch(podMonitorHandler)
		if err == nil {
			// requested stop
			return
		}
		log.Errorf("PodWatcher stopped... attempting restart: %s", err)
		time.Sleep(MonitorRestartTimeDelay)
	}
}

func nodeMonitorHandler(eventType watch.EventType, object interface{}) error {
	node, ok := object.(*v1.Node)
	if !ok {
		log.Info("nodeMonitorHandler nil node")
		return nil
	}
	if node != nil {
		pm := &PodMonitor
		// Get the CSI annotations for nodeID
		volumeIDs := make([]string, 0)
		// Print out whether the host is connected or not...
		_, _, _ = pm.callValidateVolumeHostConnectivity(node, volumeIDs, true)

		// Determine if the node is tainted
		taintnosched := nodeHasTaint(node, nodeUnreachableTaint, v1.TaintEffectNoSchedule)
		taintnoexec := nodeHasTaint(node, nodeUnreachableTaint, v1.TaintEffectNoExecute)

		log.Infof("node name: %s nodsched %t noexec %t", node.ObjectMeta.Name, taintnosched, taintnoexec)
	}
	return nil
}

// Start the NodeMonitor so that it is process nodes which might go offline.
func StartNodeMonitor(client kubernetes.Interface, labelKey, labelValue string) {
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
		watcher, err := K8sApi.SetupNodeWatch(ctx, listOptions)
		if err != nil {
			log.Errorf("Could not create NodeWatcher: %s", err)
			return
		}
		nodeMonitor.Watcher = watcher
		log.Infof("Setup of NodeWatcher complete\n")
		err = nodeMonitor.Watch(nodeMonitorHandler)

		if err == nil {
			// requested stop
			return
		}
		log.Errorf("NodeWatcher stopped... attempting restart: %s", err)
		time.Sleep(MonitorRestartTimeDelay)
	}
}
