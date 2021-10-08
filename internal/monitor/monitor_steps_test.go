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
	"os"
	"path/filepath"
	"podmon/internal/criapi"
	"podmon/internal/csiapi"
	"podmon/internal/k8sapi"
	"podmon/internal/utils"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cucumber/godog"
	"github.com/dell/gofsutil"
	log "github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
	cri "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

const (
	podns       string = "podns"
	containerID        = "1234"
)

type feature struct {
	// Logrus test hook
	loghook *logtest.Hook
	// Kubernetes objects
	pod  *v1.Pod
	pod2 *v1.Pod
	node *v1.Node
	// PodmonMonitorType
	podmonMonitor *PodMonitorType
	// CSIMock
	csiapiMock *csiapi.CSIMock
	// K8SMock
	k8sapiMock *k8sapi.K8sMock
	// CRI mock
	criMock                  *criapi.MockClient
	err                      error
	success                  bool
	podList                  []*v1.Pod   // For multi-pod tests
	podUID                   []types.UID // For multi-pod tests
	pvNames                  []string    // For multi-volume tests
	podCount                 int
	failCSIVolumePathDirRead bool
	failRemoveDir            string
	maxNodeAPILoopTimes      int
	// If true and the test case has expected loghook.LastEntry set to
	//'none', it will validate if it indeed was a successful message.
	validateLastMessage bool
	badWatchObject      bool
}

func (f *feature) aControllerMonitorUnity() error {
	return f.aControllerMonitor("unity")
}

func (f *feature) aControllerMonitorVxflex() error {
	return f.aControllerMonitor("vxflex")
}

func (f *feature) aControllerMonitor(driver string) error {
	if f.loghook == nil {
		f.loghook = logtest.NewGlobal()
	} else {
		fmt.Printf("loghook last-entry %+v\n", f.loghook.LastEntry())
	}
	switch driver {
	case "vxflex":
		Driver = new(VxflexDriver)
	case "unity":
		Driver = new(UnityDriver)
	default:
		Driver = new(VxflexDriver)
	}
	f.k8sapiMock = new(k8sapi.K8sMock)
	f.k8sapiMock.Initialize()
	K8sAPI = f.k8sapiMock
	f.csiapiMock = new(csiapi.CSIMock)
	CSIApi = f.csiapiMock
	f.criMock = new(criapi.MockClient)
	f.criMock.Initialize()
	getContainers = f.criMock.GetContainerInfo
	f.podmonMonitor = &PodMonitorType{}
	f.podmonMonitor.CSIExtensionsPresent = true
	f.podmonMonitor.DriverPathStr = "csi-vxflexos.dellemc.com"
	gofsutil.UseMockFS()
	RemoveDir = f.mockRemoveDir
	f.badWatchObject = false
	f.pod2 = nil
	return nil
}

func (f *feature) mockRemoveDir(_ string) error {
	if f.failRemoveDir != "" && f.failRemoveDir != "none" {
		return fmt.Errorf(f.failRemoveDir)
	}
	return nil
}

func (f *feature) aPodForNodeWithVolumesCondition(node string, nvolumes int, condition string) error {
	pod := f.createPod(node, nvolumes, condition, "false")
	f.pod = pod
	f.k8sapiMock.AddPod(pod)
	return nil
}

func (f *feature) aPodForNodeWithVolumesConditionAffinity(node string, nvolumes int, condition, affinity string) error {
	pod := f.createPod(node, nvolumes, condition, affinity)
	f.pod = pod
	f.k8sapiMock.AddPod(pod)
	// If affinity, create a second pod with affinity to the first
	if affinity == "true" {
		f.pod2 = f.createPod(node, nvolumes, condition, affinity)
		f.pod2.ObjectMeta.Name = "affinityPod"
		f.k8sapiMock.AddPod(f.pod2)
		fmt.Printf("Added affinitPod\n")
	}
	return nil
}

func (f *feature) iHaveAPodsForNodeWithVolumesDevicesCondition(nPods int, nodeName string, nvolumes, ndevices int, condition string) error {
	var err error
	f.podList = make([]*v1.Pod, nPods)
	mockPaths := make([]string, nPods)
	defer func() {
		for _, dirName := range mockPaths {
			os.RemoveAll(dirName)
		}
	}()
	for i := 0; i < nPods; i++ {
		pod := f.createPod(nodeName, nvolumes, condition, "false")
		f.k8sapiMock.AddPod(pod)
		f.podList[i] = pod

		dir := os.TempDir()
		CSIVolumePathFormat = filepath.Join(dir, "node-mode-testPath-%s")
		mockCSIVolumePath := fmt.Sprintf(CSIVolumePathFormat, pod.UID)
		mockCSIDevicePath := fmt.Sprintf(CSIDevicePathFormat, pod.UID)

		err = os.Mkdir(mockCSIVolumePath, 0700)
		if err != nil {
			return err
		}

		err = os.MkdirAll(mockCSIDevicePath, 0700)
		if err != nil {
			err = fmt.Errorf("Mkdir mockCSIDevicePath failed: %s", err)
			return err
		}

		mockPaths = append(mockPaths, mockCSIVolumePath)
		for _, pvName := range f.pvNames {
			if err = os.Mkdir(filepath.Join(mockCSIVolumePath, pvName), 0700); err != nil {
				return err
			}
		}
		mockPaths = append(mockPaths, mockCSIDevicePath)
		for _, pvName := range f.pvNames {
			if _, err = utils.Creat(filepath.Join(mockCSIDevicePath, pvName), 060700); err != nil {
				err = fmt.Errorf("Create mockCSIDevicePath failed: %s", err)
				return err
			}
		}
		if err = f.podmonMonitor.nodeModePodHandler(pod, watch.Added); err != nil {
			return err
		}
	}
	return nil
}

func (f *feature) iCallControllerCleanupPodForNode(nodeName string) error {
	node, _ := f.k8sapiMock.GetNode(context.Background(), nodeName)
	f.node = node
	f.success = f.podmonMonitor.controllerCleanupPod(f.pod, node, "Unit Test", false, false)
	return nil
}

func (f *feature) iInduceError(induced string) error {
	switch induced {
	case "none":
		break
	case "Connect":
		f.k8sapiMock.InducedErrors.Connect = true
	case "DeletePod":
		f.k8sapiMock.InducedErrors.DeletePod = true
	case "GetPod":
		f.k8sapiMock.InducedErrors.GetPod = true
	case "GetVolumeAttachments":
		f.k8sapiMock.InducedErrors.GetVolumeAttachments = true
	case "DeleteVolumeAttachment":
		f.k8sapiMock.InducedErrors.DeleteVolumeAttachment = true
	case "GetPersistentVolumeClaimsInNamespace":
		f.k8sapiMock.InducedErrors.GetPersistentVolumeClaimsInNamespace = true
	case "GetPersistentVolumeClaimsInPod":
		f.k8sapiMock.InducedErrors.GetPersistentVolumeClaimsInPod = true
	case "GetPersistentVolumesInPod":
		f.k8sapiMock.InducedErrors.GetPersistentVolumesInPod = true
	case "IsVolumeAttachmentToPod":
		f.k8sapiMock.InducedErrors.IsVolumeAttachmentToPod = true
	case "GetPersistentVolumeClaimName":
		f.k8sapiMock.InducedErrors.GetPersistentVolumeClaimName = true
	case "GetPersistentVolume":
		f.k8sapiMock.InducedErrors.GetPersistentVolume = true
	case "GetPersistentVolumeClaim":
		f.k8sapiMock.InducedErrors.GetPersistentVolumeClaim = true
	case "GetNode":
		f.k8sapiMock.InducedErrors.GetNode = true
	case "GetNodeWithTimeout":
		f.k8sapiMock.InducedErrors.GetNodeWithTimeout = true
	case "GetVolumeHandleFromVA":
		f.k8sapiMock.InducedErrors.GetVolumeHandleFromVA = true
	case "GetPVNameFromVA":
		f.k8sapiMock.InducedErrors.GetPVNameFromVA = true
	case "Watch":
		f.k8sapiMock.InducedErrors.Watch = true
	case "ControllerUnpublishVolume":
		f.csiapiMock.InducedErrors.ControllerUnpublishVolume = true
	case "NodeUnpublishVolume":
		f.csiapiMock.InducedErrors.NodeUnpublishVolume = true
	case "NodeUnstageVolume":
		f.csiapiMock.InducedErrors.NodeUnstageVolume = true
	case "ValidateVolumeHostConnectivity":
		f.csiapiMock.InducedErrors.ValidateVolumeHostConnectivity = true
	case "NodeConnected":
		f.csiapiMock.ValidateVolumeHostConnectivityResponse.Connected = true
	case "NodeNotConnected":
		f.csiapiMock.ValidateVolumeHostConnectivityResponse.Connected = false
	case "CSIExtensionsNotPresent":
		f.podmonMonitor.CSIExtensionsPresent = false
	case "CSIVolumePathDirRead":
		f.failCSIVolumePathDirRead = true
	case "K8sTaint":
		f.k8sapiMock.InducedErrors.TaintNode = true
	case "RemoveDir":
		f.failRemoveDir = "Could not delete"
	case "BadWatchObject":
		f.badWatchObject = true
	case "Unmount":
		gofsutil.GOFSMock.InduceUnmountError = true
	case "CreateEvent":
		f.k8sapiMock.InducedErrors.CreateEvent = true
	case "GetContainerInfo":
		f.criMock.InducedErrors.GetContainerInfo = true
	case "ContainerRunning":
		containerInfo := &criapi.ContainerInfo{
			ID:    containerID,
			Name:  "running-container",
			State: cri.ContainerState_CONTAINER_RUNNING,
		}
		f.criMock.MockContainerInfos[containerID] = containerInfo
	case "NodeUnpublishNFSShareNotFound":
		f.csiapiMock.InducedErrors.NodeUnpublishNFSShareNotFound = true
	case "NodeUnstageNFSShareNotFound":
		f.csiapiMock.InducedErrors.NodeUnstageNFSShareNotFound = true
	default:
		return fmt.Errorf("Unknown induced error: %s", induced)
	}
	return nil
}

func (f *feature) iInduceErrorForMaxTimes(error, wantFailCount string) error {
	f.k8sapiMock.WantFailCount, _ = strconv.Atoi(wantFailCount)
	err := f.iInduceError(error)
	return err
}

func (f *feature) theLastLogMessageContains(errormsg string) error {
	lastEntry := f.loghook.LastEntry()
	if errormsg == "none" {
		if f.validateLastMessage && lastEntry != nil &&
			!strings.Contains(lastEntry.Message, "Cleanup of pods complete:") {
			return fmt.Errorf("expected no error for test case, but got: %s", lastEntry.Message)
		}
		return nil
	}
	if lastEntry == nil {
		return fmt.Errorf("expected error message to contain: %s, but last log entry was nil", errormsg)
	} else if strings.Contains(lastEntry.Message, errormsg) {
		return nil
	}
	return fmt.Errorf("expected error message to contain: %s, but it was %s", errormsg, lastEntry.Message)
}

func (f *feature) theReturnStatusIs(boolean string) error {
	if boolean == "true" {
		if f.success != true {
			return errors.New("Expected true status but was false")
		}
	} else {
		if f.success != false {
			return errors.New("Expected false status but was true")
		}
	}
	return nil
}

func (f *feature) aControllerPodInfoIsPresent(boolean string) error {
	if boolean == "" {
		return nil
	}
	_, loaded := f.podmonMonitor.PodKeyToControllerPodInfo.Load(getPodKey(f.pod))
	if boolean == "true" && !loaded {
		return fmt.Errorf("Expect ControllerPodInfo for pod %s but wasn't there", getPodKey(f.pod))
	}
	if boolean == "false" && loaded {
		return fmt.Errorf("Expect no ControllerPodInfo for pod %s but was there", getPodKey(f.pod))
	}
	return nil
}

func (f *feature) aNodeWithTaint(nodeName, taint string) error {
	node, _ := f.k8sapiMock.GetNode(context.Background(), nodeName)
	node.Spec.Taints = make([]v1.Taint, 0)
	switch taint {
	case "none":
	case "noexec":
		taint := v1.Taint{
			Key:    nodeUnreachableTaint,
			Effect: v1.TaintEffectNoExecute,
		}
		node.Spec.Taints = append(node.Spec.Taints, taint)
	case "podmon-noexec":
		taint := v1.Taint{
			Key:    PodmonTaintKey,
			Effect: v1.TaintEffectNoExecute,
		}
		node.Spec.Taints = append(node.Spec.Taints, taint)
	case "nosched":
		taint := v1.Taint{
			Key:    nodeUnreachableTaint,
			Effect: v1.TaintEffectNoSchedule,
		}
		node.Spec.Taints = append(node.Spec.Taints, taint)
	case "podmon-nosched":
		taint := v1.Taint{
			Key:    PodmonTaintKey,
			Effect: v1.TaintEffectNoSchedule,
		}
		node.Spec.Taints = append(node.Spec.Taints, taint)
	}
	f.k8sapiMock.AddNode(node)
	return nil
}

func (f *feature) iCallControllerModePodHandlerWithEvent(event string) error {
	var eventType watch.EventType
	switch event {
	case "Added":
		eventType = watch.Added
	case "Modified":
		eventType = watch.Modified
	case "Deleted":
		eventType = watch.Deleted
	default:
		eventType = watch.Error
	}
	f.err = f.podmonMonitor.controllerModePodHandler(f.pod, eventType)
	if f.pod2 != nil {
		f.podmonMonitor.controllerModePodHandler(f.pod2, eventType)
	}

	// Wait on the go routine to finish
	time.Sleep(100 * time.Millisecond)
	podKey := getPodKey(f.pod)
	Lock(podKey, f.pod, LockSleepTimeDelay)
	Unlock(podKey)
	return nil
}

func (f *feature) thePodIsCleaned(boolean string) error {
	lastentry := f.loghook.LastEntry()
	switch boolean {
	case "true":
		if strings.Contains(lastentry.Message, "End Processing pods with affinity map") && f.pod2 != nil {
			return nil
		}
		if !strings.Contains(lastentry.Message, "Successfully cleaned up pod") {
			return fmt.Errorf("Expected pod to be cleaned up but it was not, last message: %s", lastentry.Message)
		}
	default:
		if strings.Contains(lastentry.Message, "Successfully cleaned up pod") {
			return fmt.Errorf("Expected pod not to be cleaned up, but it was")
		}
	}
	return nil
}

func (f *feature) iCallArrayConnectivityMonitor() error {
	ArrayConnectivityConnectionLossThreshold = 1
	SetArrayConnectivityPollRate(1 * time.Millisecond)
	f.podmonMonitor.ArrayConnectivityMonitor()
	return nil
}

func (f *feature) iCallNodeModePodHandlerForNodeWithEvent(nodeName, eventType string) error {
	node, _ := f.k8sapiMock.GetNode(context.Background(), nodeName)
	f.node = node

	var err error
	dir := os.TempDir()
	CSIVolumePathFormat = filepath.Join(dir, "node-mode-testPath-%s")
	mockCSIVolumePath := fmt.Sprintf(CSIVolumePathFormat, f.pod.UID)

	if !f.failCSIVolumePathDirRead {
		if err = os.Mkdir(mockCSIVolumePath, 0700); err != nil {
			return err
		}
		defer os.RemoveAll(mockCSIVolumePath)
		for _, pvName := range f.pvNames {
			if err = os.Mkdir(filepath.Join(mockCSIVolumePath, pvName), 0700); err != nil {
				return err
			}
		}
	}

	f.err = f.podmonMonitor.nodeModePodHandler(f.pod, watch.EventType(eventType))
	f.success = f.err != nil
	return nil
}

func (f *feature) iExpectPodMonitorToHaveMounts(nMounts int) error {
	val, ok := f.podmonMonitor.PodKeyMap.Load(getPodKey(f.pod))
	if !ok && nMounts != 0 {
		return fmt.Errorf("could not find pod, but was expected")
	}
	actualMounts := 0
	if val != nil {
		podInfo := val.(*NodePodInfo)
		actualMounts = len(podInfo.Mounts)
	}
	return AssertExpectedAndActual(assert.Equal, nMounts, actualMounts,
		"Expected %d mounts, but there were %d", nMounts, actualMounts)
}

func (f *feature) iCallNodeModeCleanupPodsForNode(nodeName string) error {
	f.validateLastMessage = true
	node, _ := f.k8sapiMock.GetNode(context.Background(), nodeName)
	f.node = node

	f.podmonMonitor.nodeModeCleanupPods(node)

	return nil
}

func (f *feature) createPod(node string, nvolumes int, condition, affinity string) *v1.Pod {
	pod := &v1.Pod{}
	pod.ObjectMeta.UID = uuid.NewUUID()
	if len(f.podUID) == 0 {
		f.podUID = make([]types.UID, 0)
	}
	f.podCount++
	f.podUID = append(f.podUID, pod.ObjectMeta.UID)
	podIndex := f.podCount - 1
	pod.ObjectMeta.Namespace = podns
	pod.ObjectMeta.Name = fmt.Sprintf("podname-%s", pod.ObjectMeta.UID)
	pod.Spec.NodeName = node
	pod.Spec.Volumes = make([]v1.Volume, 0)
	if affinity == "true" {
		f.addAffinityToPod(pod)
	}
	pod.Status.Message = "pod updated"
	pod.Status.Reason = "pod reason"
	pod.Status.ContainerStatuses = make([]v1.ContainerStatus, 0)
	containerStatus := v1.ContainerStatus{
		ContainerID: "//" + containerID,
	}
	containerInfo := &criapi.ContainerInfo{
		ID:    containerID,
		Name:  "running-container",
		State: cri.ContainerState_CONTAINER_EXITED,
	}
	f.criMock.MockContainerInfos["1234"] = containerInfo
	pod.Status.ContainerStatuses = append(pod.Status.ContainerStatuses, containerStatus)
	if pod.Status.Conditions == nil {
		pod.Status.Conditions = make([]v1.PodCondition, 0)
	}
	switch condition {
	case "Ready":
		condition := v1.PodCondition{
			Type:    "Ready",
			Status:  "True",
			Reason:  condition,
			Message: condition,
		}
		pod.Status.Conditions = append(pod.Status.Conditions, condition)
	case "NotReady":
		condition := v1.PodCondition{
			Type:    "Ready",
			Status:  "False",
			Reason:  condition,
			Message: condition,
		}
		pod.Status.Conditions = append(pod.Status.Conditions, condition)
	case "Initialized":
		condition := v1.PodCondition{
			Type:    "Initialized",
			Status:  "True",
			Reason:  condition,
			Message: condition,
		}
		pod.Status.Conditions = append(pod.Status.Conditions, condition)
	case "CrashLoop":
		waiting := &v1.ContainerStateWaiting{
			Reason:  crashLoopBackOffReason,
			Message: "unit test condition",
		}
		state := v1.ContainerState{
			Waiting: waiting,
		}
		containerStatus := v1.ContainerStatus{
			State: state,
		}
		pod.Status.ContainerStatuses = append(pod.Status.ContainerStatuses, containerStatus)
		// PodCondition is Ready=false
		condition := v1.PodCondition{
			Type:    "Ready",
			Status:  "False",
			Reason:  condition,
			Message: condition,
		}
		pod.Status.Conditions = append(pod.Status.Conditions, condition)
	}
	// add a number of volumes to the pod
	for i := 0; i < nvolumes; i++ {
		// Create a PV
		pv := &v1.PersistentVolume{}
		pv.ObjectMeta.Name = fmt.Sprintf("pv-%s-%d", f.podUID[podIndex], i)
		f.pvNames = append(f.pvNames, pv.ObjectMeta.Name)
		claimRef := &v1.ObjectReference{}
		claimRef.Kind = "PersistentVolumeClaim"
		claimRef.Namespace = podns
		claimRef.Name = fmt.Sprintf("pvc-%s-%d", f.podUID[podIndex], i)
		pv.Spec.ClaimRef = claimRef
		log.Infof("claimRef completed")
		csiPVSource := &v1.CSIPersistentVolumeSource{}
		csiPVSource.Driver = "csi-vxflexos.dellemc.com"
		csiPVSource.VolumeHandle = fmt.Sprintf("vhandle%d", i)
		pv.Spec.CSI = csiPVSource
		// Create a PVC
		pvc := &v1.PersistentVolumeClaim{}
		pvc.ObjectMeta.Namespace = podns
		pvc.ObjectMeta.Name = fmt.Sprintf("pvc-%s-%d", f.podUID[podIndex], i)
		pvc.Spec.VolumeName = pv.ObjectMeta.Name
		pvc.Status.Phase = "Bound"
		// Create a VolumeAttachment
		va := &storagev1.VolumeAttachment{}
		va.ObjectMeta.Name = fmt.Sprintf("va%d", i)
		va.Spec.NodeName = node
		va.Spec.Source.PersistentVolumeName = &pv.ObjectMeta.Name
		// Add the objects to the mock engine.
		f.k8sapiMock.AddPV(pv)
		f.k8sapiMock.AddPVC(pvc)
		f.k8sapiMock.AddVA(va)
		// Add a volume to the pod
		vol := v1.Volume{}
		vol.Name = fmt.Sprintf("pv-%s-%d", f.podUID[podIndex], i)
		pvcSource := &v1.PersistentVolumeClaimVolumeSource{}
		pvcSource.ClaimName = pvc.ObjectMeta.Name
		volSource := v1.VolumeSource{}
		volSource.PersistentVolumeClaim = pvcSource
		vol.VolumeSource = volSource
		pod.Spec.Volumes = append(pod.Spec.Volumes, vol)
	}
	return pod
}

// Adds a pod affinity specification based on hostname to the pod
func (f *feature) addAffinityToPod(pod *v1.Pod) {
	matchLabels := make(map[string]string)
	matchLabels["affinityLabel1"] = "affinityLabelValue1"
	values := make([]string, 1)
	values[0] = "affinityValue1"
	matchExpr := metav1.LabelSelectorRequirement{
		Operator: "In",
		Key:      "affinityLabel2",
		Values:   values,
	}
	matchExprs := make([]metav1.LabelSelectorRequirement, 1)
	matchExprs[0] = matchExpr
	labelSelector := metav1.LabelSelector{
		MatchLabels:      matchLabels,
		MatchExpressions: matchExprs,
	}
	namespaces := make([]string, 1)
	namespaces[0] = podns
	podAffinityTerm := v1.PodAffinityTerm{
		LabelSelector: &labelSelector,
		Namespaces:    namespaces,
		TopologyKey:   hostNameTopologyKey,
	}
	podAffinityTerms := make([]v1.PodAffinityTerm, 1)
	podAffinityTerms[0] = podAffinityTerm
	podAffinity := v1.PodAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: podAffinityTerms,
	}
	affinity := v1.Affinity{
		PodAffinity: &podAffinity,
	}
	pod.Spec.Affinity = &affinity
}

func (f *feature) theControllerCleanedUpPodsForNode(cleanedUpCount int, nodeName string) error {
	node, _ := f.k8sapiMock.GetNode(context.Background(), nodeName)
	for i := 0; i < cleanedUpCount; i++ {
		if success := f.podmonMonitor.controllerCleanupPod(f.podList[i], node, "Unit Test", false, false); !success {
			return fmt.Errorf("controllerCleanPod was not successful")
		}
	}
	return nil
}

func (f *feature) theUnmountReturns(err string) error {
	gofsutil.GOFSMock.InduceUnmountError = strings.ToLower(err) == "true"
	return nil
}

func (f *feature) nodeEnvVarsSet(nodeName string) error {
	if err := os.Setenv("KUBE_NODE_NAME", nodeName); err != nil {
		return err
	}
	if err := os.Setenv("X_CSI_PRIVATE_MOUNT_DIR", "/test/mock/mount"); err != nil {
		return err
	}
	return nil
}

func (f *feature) iAllowNodeAPIMonitorLoopToRun(maxLoopTimes int) error {
	f.maxNodeAPILoopTimes = maxLoopTimes
	return nil
}

func (f *feature) initAPILoopVariables() {
	f.validateLastMessage = true

	APICheckInterval = 30 * time.Millisecond
	APICheckRetryTimeout = 10 * time.Millisecond
	APICheckFirstTryTimeout = 5 * time.Millisecond
	TaintCountDelay = 0

	loops := 0
	APIMonitorWait = func(interval time.Duration) bool {
		loops++
		if loops >= f.maxNodeAPILoopTimes {
			return true
		}
		time.Sleep(interval)
		return false
	}
}

func (f *feature) iCallStartAPIMonitor() error {
	f.initAPILoopVariables()
	StartAPIMonitor(K8sAPI, APICheckFirstTryTimeout, APICheckRetryTimeout, APICheckInterval, APIMonitorWait)
	time.Sleep(2 * APICheckInterval)
	return nil
}

func (f *feature) iCallAPIMonitorLoop(nodeName string) error {
	f.initAPILoopVariables()
	f.podmonMonitor.apiMonitorLoop(K8sAPI, nodeName, APICheckFirstTryTimeout, APICheckRetryTimeout, APICheckInterval, APIMonitorWait)
	return nil
}

func (f *feature) iCallStartPodMonitorWithKeyAndValue(key, value string) error {
	MonitorRestartTimeDelay = 5 * time.Millisecond
	client := fake.NewSimpleClientset()
	go StartPodMonitor(K8sAPI, client, key, value, MonitorRestartTimeDelay)
	return nil
}

func (f *feature) iCloseTheWatcher() error {
	time.Sleep(7 * time.Millisecond)
	f.k8sapiMock.Watcher.Reset()
	return nil
}

func (f *feature) iSendAPodEventType(eventType string) error {
	fakeWatcher := f.k8sapiMock.Watcher
	switch eventType {
	case "None":
	case "Add":
		if f.badWatchObject {
			fakeWatcher.Add(f.node)
		} else {
			fakeWatcher.Add(f.pod)
		}
	case "Modify":
		fakeWatcher.Modify(f.pod)
	case "Delete":
		fakeWatcher.Delete(f.pod)
	case "Error":
		fakeWatcher.Error(nil)
	case "Stop":
		fakeWatcher.Stop()
	}
	return nil
}

func (f *feature) podMonitorMode(mode string) error {
	PodMonitor.Mode = mode
	return nil
}

func (f *feature) iCallStartNodeMonitorWithKeyAndValue(key, value string) error {
	MonitorRestartTimeDelay = 5 * time.Millisecond
	client := fake.NewSimpleClientset()
	go StartNodeMonitor(K8sAPI, client, key, value, MonitorRestartTimeDelay)
	return nil
}

func (f *feature) iSendANodeEventType(eventType string) error {
	node, _ := f.k8sapiMock.GetNode(context.Background(), "node1")
	f.node = node
	fakeWatcher := f.k8sapiMock.Watcher
	switch eventType {
	case "None":
	case "Add":
		if f.badWatchObject {
			fakeWatcher.Add(f.pod)
		} else {
			fakeWatcher.Add(f.node)
		}
	case "Modify":
		fakeWatcher.Modify(f.node)
	case "Delete":
		fakeWatcher.Delete(f.node)
	case "Error":
		fakeWatcher.Error(nil)
	case "Stop":
		fakeWatcher.Stop()
	}
	return nil
}

type safeCount struct {
	lock  sync.Mutex
	value int
}

func (c *safeCount) inc() {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.value++
}

func (c *safeCount) equals(compareTo int) bool {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.value == compareTo
}

func (c *safeCount) dump() {
	c.lock.Lock()
	defer c.lock.Unlock()
	fmt.Printf("value = %d", c.value)
}

func (f *feature) iCallTestLockAndGetPodKey() error {
	// Test getPodKey and splitPodKey
	podkey := getPodKey(f.pod)
	ns, name := splitPodKey(podkey)
	if podkey != fmt.Sprintf("%s/%s", ns, name) {
		return fmt.Errorf("Error in getPodKey/splitPodKey %s %s/%s", podkey, ns, name)
	}
	lockSleepTimeDelay := LockSleepTimeDelay
	LockSleepTimeDelay = 100 * time.Millisecond
	// Test Lock and Unlock
	const nlocks = 5
	var counter safeCount
	for i := 0; i < nlocks; i++ {
		go func(duration time.Duration) {
			Lock(podkey, f.pod, duration)
			counter.inc()
			time.Sleep(duration)
			Unlock(podkey)
		}(LockSleepTimeDelay)
	}
	time.Sleep(10 * LockSleepTimeDelay)
	LockSleepTimeDelay = lockSleepTimeDelay
	counter.dump()
	if !counter.equals(nlocks) {
		return fmt.Errorf("Error in Lock()/Unlock()")
	}
	return nil
}

func (f *feature) createPodErrorCase(node string, nvolumes int, condition, affinity string, errorcase string) *v1.Pod {
	pod := &v1.Pod{}
	pod.ObjectMeta.UID = uuid.NewUUID()
	if len(f.podUID) == 0 {
		f.podUID = make([]types.UID, 0)
	}
	f.podCount++
	f.podUID = append(f.podUID, pod.ObjectMeta.UID)
	podIndex := f.podCount - 1
	pod.ObjectMeta.Namespace = podns
	pod.ObjectMeta.Name = fmt.Sprintf("podname-%s", pod.ObjectMeta.UID)
	pod.Spec.NodeName = node
	pod.Spec.Volumes = make([]v1.Volume, 0)
	if affinity == "true" {
		f.addAffinityToPodErrorCase(pod, errorcase)
	}
	pod.Status.Message = "pod updated"
	pod.Status.Reason = "pod reason"
	pod.Status.ContainerStatuses = make([]v1.ContainerStatus, 0)
	containerStatus := v1.ContainerStatus{
		ContainerID: "//" + containerID,
	}
	containerInfo := &criapi.ContainerInfo{
		ID:    containerID,
		Name:  "running-container",
		State: cri.ContainerState_CONTAINER_EXITED,
	}
	f.criMock.MockContainerInfos["1234"] = containerInfo
	pod.Status.ContainerStatuses = append(pod.Status.ContainerStatuses, containerStatus)
	if pod.Status.Conditions == nil {
		pod.Status.Conditions = make([]v1.PodCondition, 0)
	}
	switch condition {
	case "Ready":
		condition := v1.PodCondition{
			Type:    "Ready",
			Status:  "True",
			Reason:  condition,
			Message: condition,
		}
		pod.Status.Conditions = append(pod.Status.Conditions, condition)
	case "NotReady":
		condition := v1.PodCondition{
			Type:    "Ready",
			Status:  "False",
			Reason:  condition,
			Message: condition,
		}
		pod.Status.Conditions = append(pod.Status.Conditions, condition)
	case "Initialized":
		condition := v1.PodCondition{
			Type:    "Initialized",
			Status:  "True",
			Reason:  condition,
			Message: condition,
		}
		pod.Status.Conditions = append(pod.Status.Conditions, condition)
	case "CrashLoop":
		waiting := &v1.ContainerStateWaiting{
			Reason:  crashLoopBackOffReason,
			Message: "unit test condition",
		}
		state := v1.ContainerState{
			Waiting: waiting,
		}
		containerStatus := v1.ContainerStatus{
			State: state,
		}
		pod.Status.ContainerStatuses = append(pod.Status.ContainerStatuses, containerStatus)
		// PodCondition is Ready=false
		condition := v1.PodCondition{
			Type:    "Ready",
			Status:  "False",
			Reason:  condition,
			Message: condition,
		}
		pod.Status.Conditions = append(pod.Status.Conditions, condition)
	}
	// add a number of volumes to the pod
	for i := 0; i < nvolumes; i++ {
		// Create a PV
		pv := &v1.PersistentVolume{}
		pv.ObjectMeta.Name = fmt.Sprintf("pv-%s-%d", f.podUID[podIndex], i)
		f.pvNames = append(f.pvNames, pv.ObjectMeta.Name)
		claimRef := &v1.ObjectReference{}
		claimRef.Kind = "PersistentVolumeClaim"
		claimRef.Namespace = podns
		claimRef.Name = fmt.Sprintf("pvc-%s-%d", f.podUID[podIndex], i)
		pv.Spec.ClaimRef = claimRef
		log.Infof("claimRef completed")
		csiPVSource := &v1.CSIPersistentVolumeSource{}
		csiPVSource.Driver = "csi-vxflexos.dellemc.com"
		csiPVSource.VolumeHandle = fmt.Sprintf("vhandle%d", i)
		pv.Spec.CSI = csiPVSource
		// Create a PVC
		pvc := &v1.PersistentVolumeClaim{}
		pvc.ObjectMeta.Namespace = podns
		pvc.ObjectMeta.Name = fmt.Sprintf("pvc-%s-%d", f.podUID[podIndex], i)
		pvc.Spec.VolumeName = pv.ObjectMeta.Name
		pvc.Status.Phase = "Bound"
		// Create a VolumeAttachment
		va := &storagev1.VolumeAttachment{}
		va.ObjectMeta.Name = fmt.Sprintf("va%d", i)
		va.Spec.NodeName = node
		va.Spec.Source.PersistentVolumeName = &pv.ObjectMeta.Name
		// Add the objects to the mock engine.
		f.k8sapiMock.AddPV(pv)
		f.k8sapiMock.AddPVC(pvc)
		f.k8sapiMock.AddVA(va)
		// Add a volume to the pod
		vol := v1.Volume{}
		vol.Name = fmt.Sprintf("pv-%s-%d", f.podUID[podIndex], i)
		pvcSource := &v1.PersistentVolumeClaimVolumeSource{}
		pvcSource.ClaimName = pvc.ObjectMeta.Name
		volSource := v1.VolumeSource{}
		volSource.PersistentVolumeClaim = pvcSource
		vol.VolumeSource = volSource
		pod.Spec.Volumes = append(pod.Spec.Volumes, vol)
	}
	return pod
}

// Adds a pod affinity specification based on error condition
func (f *feature) addAffinityToPodErrorCase(pod *v1.Pod, errorcase string) {
	matchLabels := make(map[string]string)
	matchLabels["affinityLabel1"] = "affinityLabelValue1"
	values := make([]string, 1)
	values[0] = "affinityValue1"
	matchExpr := metav1.LabelSelectorRequirement{
		Operator: "In",
		Key:      "affinityLabel2",
		Values:   values,
	}
	if errorcase == "operator" {
		matchExpr.Operator = "Out"
	}
	matchExprs := make([]metav1.LabelSelectorRequirement, 1)
	matchExprs[0] = matchExpr
	labelSelector := metav1.LabelSelector{
		MatchLabels:      matchLabels,
		MatchExpressions: matchExprs,
	}
	namespaces := make([]string, 1)
	namespaces[0] = podns
	podAffinityTerm := v1.PodAffinityTerm{
		LabelSelector: &labelSelector,
		Namespaces:    namespaces,
		TopologyKey:   hostNameTopologyKey,
	}
	if errorcase == "topology" {
		podAffinityTerm.TopologyKey = "unknown/hostname"
	}
	if errorcase == "labelselector" {
		podAffinityTerm.LabelSelector = nil
	}
	podAffinityTerms := make([]v1.PodAffinityTerm, 1)
	podAffinityTerms[0] = podAffinityTerm
	podAffinity := v1.PodAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: podAffinityTerms,
	}
	if errorcase == "required" {
		podAffinity.RequiredDuringSchedulingIgnoredDuringExecution = nil
	}
	affinity := v1.Affinity{
		PodAffinity: &podAffinity,
	}

	if errorcase == "podaffinity" {
		affinity.PodAffinity = nil
	}
	pod.Spec.Affinity = &affinity
}

func (f *feature) aControllerPodWithPodaffinitylabels() error {
	if f.loghook == nil {
		f.loghook = logtest.NewGlobal()
	} else {
		fmt.Printf("loghook last-entry %+v\n", f.loghook.LastEntry())
	}
	// This test is for error condition of func getPodAffinityLabels
	// testing only for VxflexosDriver
	Driver = new(VxflexDriver)

	f.k8sapiMock = new(k8sapi.K8sMock)
	f.k8sapiMock.Initialize()
	K8sAPI = f.k8sapiMock
	f.csiapiMock = new(csiapi.CSIMock)
	CSIApi = f.csiapiMock
	f.criMock = new(criapi.MockClient)
	f.criMock.Initialize()
	getContainers = f.criMock.GetContainerInfo
	f.podmonMonitor = &PodMonitorType{}
	f.podmonMonitor.CSIExtensionsPresent = true
	f.podmonMonitor.DriverPathStr = "csi-vxflexos.dellemc.com"
	gofsutil.UseMockFS()
	RemoveDir = f.mockRemoveDir
	f.badWatchObject = false
	f.pod2 = nil
	return nil
}

func (f *feature) createAPodForNodeWithVolumesConditionAffinityErrorcase(node string, nvolumes int, condition, affinity, errorcase string) error {
	pod := f.createPodErrorCase(node, nvolumes, condition, affinity, errorcase)
	f.pod = pod
	f.k8sapiMock.AddPod(pod)
	return nil
}

func (f *feature) iCallGetPodAffinityLabels() error {
	f.podmonMonitor.getPodAffinityLabels(f.pod)
	return nil
}

func MonitorTestScenarioInit(context *godog.ScenarioContext) {
	f := &feature{}
	context.Step(`^a controller monitor "([^"]*)"$`, f.aControllerMonitor)
	context.Step(`^a controller monitor unity$`, f.aControllerMonitorUnity)
	context.Step(`^a controller monitor vxflex$`, f.aControllerMonitorVxflex)
	//context.Step(`^a pod for node "([^"]*)" with (\d+) volumes condition "([^"]*)"$`, f.aPodForNodeWithVolumesCondition)
	context.Step(`^a pod for node "([^"]*)" with (\d+) volumes condition "([^"]*)"$`, f.aPodForNodeWithVolumesCondition)
	context.Step(`^a pod for node "([^"]*)" with (\d+) volumes condition "([^"]*)" affinity "([^"]*)"$`, f.aPodForNodeWithVolumesConditionAffinity)
	context.Step(`^I call controllerCleanupPod for node "([^"]*)"$`, f.iCallControllerCleanupPodForNode)
	context.Step(`^I induce error "([^"]*)"$`, f.iInduceError)
	context.Step(`^the last log message contains "([^"]*)"$`, f.theLastLogMessageContains)
	context.Step(`^the return status is "([^"]*)"$`, f.theReturnStatusIs)
	context.Step(`^a controllerPodInfo is present "([^"]*)"$`, f.aControllerPodInfoIsPresent)
	context.Step(`^a node "([^"]*)" with taint "([^"]*)"$`, f.aNodeWithTaint)
	context.Step(`^I call controllerModePodHandler with event "([^"]*)"$`, f.iCallControllerModePodHandlerWithEvent)
	context.Step(`^the pod is cleaned "([^"]*)"$`, f.thePodIsCleaned)
	context.Step(`^I call ArrayConnectivityMonitor$`, f.iCallArrayConnectivityMonitor)
	context.Step(`^I call nodeModePodHandler for node "([^"]*)" with event "([^"]*)"$`, f.iCallNodeModePodHandlerForNodeWithEvent)
	context.Step(`^I call nodeModeCleanupPods for node "([^"]*)"$`, f.iCallNodeModeCleanupPodsForNode)
	context.Step(`^I expect podMonitor to have (\d+) mounts$`, f.iExpectPodMonitorToHaveMounts)
	context.Step(`^I have a (\d+) pods for node "([^"]*)" with (\d+) volumes (\d+) devices condition "([^"]*)"$`, f.iHaveAPodsForNodeWithVolumesDevicesCondition)
	context.Step(`^the controller cleaned up (\d+) pods for node "([^"]*)"$`, f.theControllerCleanedUpPodsForNode)
	context.Step(`^the unmount returns "([^"]*)"$`, f.theUnmountReturns)
	context.Step(`^node "([^"]*)" env vars set$`, f.nodeEnvVarsSet)
	context.Step(`^I allow nodeApiMonitor loop to run (\d+)$`, f.iAllowNodeAPIMonitorLoopToRun)
	context.Step(`^I call StartAPIMonitor$`, f.iCallStartAPIMonitor)
	context.Step(`^I call apiMonitorLoop for "([^"]*)"$`, f.iCallAPIMonitorLoop)
	context.Step(`^I induce error "([^"]*)" for "([^"]*)"$`, f.iInduceErrorForMaxTimes)
	context.Step(`^I call StartPodMonitor with key "([^"]*)" and value "([^"]*)"$`, f.iCallStartPodMonitorWithKeyAndValue)
	context.Step(`^I close the Watcher$`, f.iCloseTheWatcher)
	context.Step(`^I send a pod event type "([^"]*)"$`, f.iSendAPodEventType)
	context.Step(`^pod monitor mode "([^"]*)"$`, f.podMonitorMode)
	context.Step(`^I call StartNodeMonitor with key "([^"]*)" and value "([^"]*)"$`, f.iCallStartNodeMonitorWithKeyAndValue)
	context.Step(`^I send a node event type "([^"]*)"$`, f.iSendANodeEventType)
	context.Step(`^I call test lock and getPodKey$`, f.iCallTestLockAndGetPodKey)
	context.Step(`^a controller pod with podaffinitylabels$`, f.aControllerPodWithPodaffinitylabels)
	context.Step(`^create a pod for node "([^"]*)" with (\d+) volumes condition "([^"]*)" affinity "([^"]*)" errorcase "([^"]*)"$`, f.createAPodForNodeWithVolumesConditionAffinityErrorcase)
	context.Step(`^I call getPodAffinityLabels$`, f.iCallGetPodAffinityLabels)
}
