package monitor

import (
	"context"
	"errors"
	"fmt"
	"github.com/cucumber/godog"
	"github.com/dell/gofsutil"
	log "github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
	"os"
	"path/filepath"
	"podmon/internal/csiapi"
	"podmon/internal/k8sapi"
	"strconv"
	"strings"
	"time"
)

const (
	podns string = "podns"
)

type feature struct {
	// Logrus test hook
	loghook *logtest.Hook
	// Kubernetes objects
	pod  *v1.Pod
	node *v1.Node
	// PodmonMonitorType
	podmonMonitor *PodMonitorType
	// CSIMock
	csiapiMock *csiapi.CSIMock
	// K8SMock
	k8sapiMock               *k8sapi.K8sMock
	err                      error
	success                  bool
	podList                  []*v1.Pod   // For multi-pod tests
	podUID                   []types.UID // For multi-pod tests
	pvNames                  []string    // For multi-volume tests
	podCount                 int
	failCSIVolumePathDirRead bool
	failKubectlTaint         bool
	failRemoveDir            string
	maxNodeApiLoopTimes      int
	// If true and the test case has expected loghook.LastEntry set to
	//'none', it will validate if it indeed was a successful message.
	validateLastMessage bool
	badWatchObject      bool
}

func (f *feature) aControllerMonitor() error {
	if f.loghook == nil {
		f.loghook = logtest.NewGlobal()
	} else {
		fmt.Printf("loghook last-entry %+v\n", f.loghook.LastEntry())
	}
	f.k8sapiMock = new(k8sapi.K8sMock)
	f.k8sapiMock.Initialize()
	K8sApi = f.k8sapiMock
	f.csiapiMock = new(csiapi.CSIMock)
	CSIApi = f.csiapiMock
	f.podmonMonitor = &PodMonitorType{}
	f.podmonMonitor.CSIExtensionsPresent = true
	f.podmonMonitor.DriverPathStr = "csi-vxflexos.dellemc.com"
	gofsutil.UseMockFS()
	KubectlTaint = f.mockKubectlTaint
	RemoveDir = f.mockRemoveDir
	f.badWatchObject = false
	return nil
}

func (f *feature) mockKubectlTaint(operation, name, taint string) error {
	if f.failKubectlTaint {
		return fmt.Errorf("mock failure: operation %s against %s with taint %s failed", operation, name, taint)
	}
	return nil
}

func (f *feature) mockRemoveDir(_ string) error {
	if f.failRemoveDir != "" && f.failRemoveDir != "none" {
		return fmt.Errorf(f.failRemoveDir)
	}
	return nil
}

func (f *feature) aPodForNodeWithVolumesCondition(node string, nvolumes int, condition string) error {
	pod := f.createPod(node, nvolumes, condition)
	f.pod = pod
	f.k8sapiMock.AddPod(pod)
	return nil
}

func (f *feature) iHaveAPodsForNodeWithVolumesCondition(nPods int, nodeName string, nvolumes int, condition string) error {
	var err error
	f.podList = make([]*v1.Pod, nPods)
	mockPaths := make([]string, nPods)
	defer func() {
		for _, dirName := range mockPaths {
			os.RemoveAll(dirName)
		}
	}()
	for i := 0; i < nPods; i++ {
		pod := f.createPod(nodeName, nvolumes, condition)
		f.k8sapiMock.AddPod(pod)
		f.podList[i] = pod

		dir := os.TempDir()
		CSIVolumePathFormat = filepath.Join(dir, "node-mode-testPath-%s")
		mockCSIVolumePath := fmt.Sprintf(CSIVolumePathFormat, pod.UID)

		err = os.Mkdir(mockCSIVolumePath, 0700)
		if err != nil {
			return err
		}

		mockPaths = append(mockPaths, mockCSIVolumePath)
		for _, pvName := range f.pvNames {
			if err = os.Mkdir(filepath.Join(mockCSIVolumePath, pvName), 0700); err != nil {
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
	f.success = f.podmonMonitor.controllerCleanupPod(f.pod, node)
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
	case "KubectlTaint":
		f.failKubectlTaint = true
	case "RemoveDir":
		f.failRemoveDir = "Could not delete"
	case "BadWatchObject":
		f.badWatchObject = true
	case "Unmount":
		gofsutil.GOFSMock.InduceUnmountError = true
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
			Key:    podmonTaintKey,
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
			Key:    podmonTaintKey,
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

	// Wait on the go routine to finish
	time.Sleep(100 * time.Millisecond)
	podKey := getPodKey(f.pod)
	Lock(podKey, f.pod)
	Unlock(podKey)
	return nil
}

func (f *feature) thePodIsCleaned(boolean string) error {
	lastentry := f.loghook.LastEntry()
	switch boolean {
	case "true":
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
	ArrayConnectivityPollRate = 1 * time.Millisecond
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

func (f *feature) createPod(node string, nvolumes int, condition string) *v1.Pod {
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
	pod.Status.Message = "pod updated"
	pod.Status.Reason = "pod reason"
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

func (f *feature) theControllerCleanedUpPodsForNode(cleanedUpCount int, nodeName string) error {
	node, _ := f.k8sapiMock.GetNode(context.Background(), nodeName)
	for i := 0; i < cleanedUpCount; i++ {
		if success := f.podmonMonitor.controllerCleanupPod(f.podList[i], node); !success {
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

func (f *feature) iAllowNodeApiMonitorLoopToRun(maxLoopTimes int) error {
	f.maxNodeApiLoopTimes = maxLoopTimes
	return nil
}

func (f *feature) initApiLoopVariables() {
	f.validateLastMessage = true

	APICheckInterval = 30 * time.Millisecond
	APICheckRetryTimeout = 10 * time.Millisecond
	APICheckFirstTryTimeout = 5 * time.Millisecond

	loops := 0
	APIMonitorWait = func() bool {
		loops++
		if loops >= f.maxNodeApiLoopTimes {
			return true
		} else {
			time.Sleep(APICheckInterval)
			return false
		}
	}
}

func (f *feature) iCallStartAPIMonitor() error {
	f.initApiLoopVariables()
	StartAPIMonitor()
	time.Sleep(2 * APICheckInterval)
	return nil
}

func (f *feature) iCallApiMonitorLoop(nodeName string) error {
	f.initApiLoopVariables()
	f.podmonMonitor.apiMonitorLoop(nodeName)
	return nil
}

func (f *feature) iCallStartPodMonitorWithKeyAndValue(key, value string) error {
	MonitorRestartTimeDelay = 5 * time.Millisecond
	client := fake.NewSimpleClientset()
	go StartPodMonitor(client, key, value)
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
	go StartNodeMonitor(client, key, value)
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
	var counter int
	for i := 0; i < nlocks; i++ {
		go func() {
			Lock(podkey, f.pod)
			counter++
			time.Sleep(LockSleepTimeDelay)
			Unlock(podkey)
		}()
	}
	time.Sleep(10 * LockSleepTimeDelay)
	LockSleepTimeDelay = lockSleepTimeDelay
	if counter != nlocks {
		return fmt.Errorf("Error in Lock()/Unlock()")
	}
	return nil
}

func MonitorTestScenarioInit(context *godog.ScenarioContext) {
	f := &feature{}
	context.Step(`^a controller monitor$`, f.aControllerMonitor)
	context.Step(`^a pod for node "([^"]*)" with (\d+) volumes condition "([^"]*)"$`, f.aPodForNodeWithVolumesCondition)
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
	context.Step(`^I have a (\d+) pods for node "([^"]*)" with (\d+) volumes condition "([^"]*)"$`, f.iHaveAPodsForNodeWithVolumesCondition)
	context.Step(`^the controller cleaned up (\d+) pods for node "([^"]*)"$`, f.theControllerCleanedUpPodsForNode)
	context.Step(`^the unmount returns "([^"]*)"$`, f.theUnmountReturns)
	context.Step(`^node "([^"]*)" env vars set$`, f.nodeEnvVarsSet)
	context.Step(`^I allow nodeApiMonitor loop to run (\d+)$`, f.iAllowNodeApiMonitorLoopToRun)
	context.Step(`^I call StartAPIMonitor$`, f.iCallStartAPIMonitor)
	context.Step(`^I call apiMonitorLoop for "([^"]*)"$`, f.iCallApiMonitorLoop)
	context.Step(`^I induce error "([^"]*)" for "([^"]*)"$`, f.iInduceErrorForMaxTimes)
	context.Step(`^I call StartPodMonitor with key "([^"]*)" and value "([^"]*)"$`, f.iCallStartPodMonitorWithKeyAndValue)
	context.Step(`^I close the Watcher$`, f.iCloseTheWatcher)
	context.Step(`^I send a pod event type "([^"]*)"$`, f.iSendAPodEventType)
	context.Step(`^pod monitor mode "([^"]*)"$`, f.podMonitorMode)
	context.Step(`^I call StartNodeMonitor with key "([^"]*)" and value "([^"]*)"$`, f.iCallStartNodeMonitorWithKeyAndValue)
	context.Step(`^I send a node event type "([^"]*)"$`, f.iSendANodeEventType)
	context.Step(`^I call test lock and getPodKey$`, f.iCallTestLockAndGetPodKey)
}
