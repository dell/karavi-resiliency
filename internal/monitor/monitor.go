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
	nodeUnreachableTaint    = "node.kubernetes.io/unreachable"
	podReadyCondition       = "Ready"
	podInitializedCondition = "Initialized"
	podmon                  = "podmon"
)

var (
	// podmonTaintKey is the key for this driver's podmon taint.
	podmonTaintKey = "driver.podmon.dellemc.com"
	//ShortTimeout used for initial try
	ShortTimeout = 10 * time.Second
	//MediumTimeout is a wait-backoff after the ShortTimeout
	MediumTimeout = 30 * time.Second
	//LongTimeout is a longer wait-backoff period
	LongTimeout = 180 * time.Second
	//PendingRetryTime time between retry of certain CSI calls
	PendingRetryTime = 30 * time.Second
	//NodeAPIInterval time between NodeAPI checks
	NodeAPIInterval = 30 * time.Second
	//CSIMaxRetries max times to retry certain CSI calls
	CSIMaxRetries = 3
	//MonitorRestartTimeDelay time to wait before restarting monitor
	MonitorRestartTimeDelay = 10 * time.Second
	//LockSleepTimeDelay wait for lock retry
	LockSleepTimeDelay = 1 * time.Second
)

//PodMonitorType structure is tracking data for the pod monitor
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

//PodMonitor is a reference to tracking data for the pod monitor
var PodMonitor PodMonitorType

//K8sAPI is a reference to the internal k8s API wrapper
var K8sAPI k8sapi.K8sAPI

//CSIApi is a reference to the internal CSI API wrapper
var CSIApi csiapi.CSIApi

//CSIVolumePathFormat is a formatter string used for producing the full path of a volume mount
var CSIVolumePathFormat = "/var/lib/kubelet/pods/%s/volumes/kubernetes.io~csi"

//CSIDevicePathFormat is a formatter string used for producing the full path of a block volume
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

//Lock acquires a sync lock based on the pod reference and key
func Lock(podkey string, pod *v1.Pod, delay time.Duration) {
	_, loaded := PodMonitor.PodKeyMap.LoadOrStore(podkey, pod)
	for loaded {
		time.Sleep(delay)
		_, loaded = PodMonitor.PodKeyMap.LoadOrStore(podkey, pod)
	}
}

//Unlock returns a sync lock based on a pod key reference
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

//StartPodMonitor starts the PodMonitor so that it is processing pods which might have problems.
// The labelKey and labelValue are used for filtering.
func StartPodMonitor(api k8sapi.K8sAPI, client kubernetes.Interface, labelKey, labelValue string, restartDelay time.Duration) {
	log.Infof("attempting to start PodMonitor\n")
	podmonTaintKey = fmt.Sprintf("%s.podmon.storage.dell.com", Driver.GetDriverName())
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
		time.Sleep(restartDelay)
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

//StartNodeMonitor starts the NodeMonitor so that it is process nodes which might go offline.
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
		time.Sleep(restartDelay)
	}
}
