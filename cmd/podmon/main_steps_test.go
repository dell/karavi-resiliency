/*
* Copyright (c) 2021-2022 Dell Inc., or its subsidiaries. All Rights Reserved.
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

package main

import (
	"context"
	"fmt"
	"os"
	"podmon/internal/csiapi"
	"podmon/internal/k8sapi"
	"podmon/internal/monitor"
	"strings"
	"sync"
	"time"

	"github.com/cucumber/godog"
	"github.com/dell/gofsutil"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"k8s.io/client-go/kubernetes"
)

type mainFeature struct {
	// Logrus test hook
	loghook             *logtest.Hook
	k8sapiMock          *k8sapi.K8sMock
	csiapiMock          *csiapi.CSIMock
	leaderElect         *mockLeaderElect
	failStartAPIMonitor bool
}

var saveOriginalArgs sync.Once
var originalArgs []string

func (m *mainFeature) aPodmonInstance() error {
	if m.loghook == nil {
		m.loghook = logtest.NewGlobal()
	} else {
		fmt.Printf("loghook last-entry %+v\n", m.loghook.LastEntry())
	}
	monitor.PodMonitor.CSIExtensionsPresent = false
	m.csiapiMock = new(csiapi.CSIMock)
	m.k8sapiMock = new(k8sapi.K8sMock)
	GetCSIClient = m.mockGetCSIClient
	K8sAPI = m.k8sapiMock
	m.leaderElect = &mockLeaderElect{}
	LeaderElection = m.mockLeaderElection
	StartAPIMonitorFn = m.mockStartAPIMonitor
	StartPodMonitorFn = m.mockStartPodMonitor
	StartNodeMonitorFn = m.mockStartNodeMonitor
	monitor.K8sAPI = m.k8sapiMock
	gofsutil.UseMockFS()
	PodMonWait = m.mockPodMonWait
	saveOriginalArgs.Do(func() {
		originalArgs = os.Args
	})

	return nil
}

type mockLeaderElect struct {
	failLeaderElection bool
}

func (le *mockLeaderElect) Run() error {
	if le.failLeaderElection {
		return fmt.Errorf("induced leaderElection failure")
	}
	return nil
}

func (le *mockLeaderElect) WithNamespace(namespace string) {

}

func (m *mainFeature) mockGetCSIClient(csiSock string, clientOpts ...grpc.DialOption) (csiapi.CSIApi, error) {
	return m.csiapiMock, nil
}

func (m *mainFeature) mockStartPodMonitor(api k8sapi.K8sAPI, client kubernetes.Interface, labelKey, labelValue string, duration time.Duration) {
}

func (m *mainFeature) mockStartNodeMonitor(api k8sapi.K8sAPI, client kubernetes.Interface, labelKey, labelValue string, duration time.Duration) {
}

func (m *mainFeature) mockStartAPIMonitor(api k8sapi.K8sAPI, first, retry, interval time.Duration, waiter func(interval time.Duration) bool) error {
	if m.failStartAPIMonitor {
		return fmt.Errorf("induced StorageAPIMonitor failure")
	}
	return nil
}

func (m *mainFeature) mockPodMonWait() bool {
	return true
}

func (m *mainFeature) mockLeaderElection(runFunc func(ctx context.Context)) leaderElection {
	return m.leaderElect
}

func (m *mainFeature) podmonEnvVarsSetTo(k8sSvc, k8sSvcPort string) error {
	os.Setenv("KUBERNETES_SERVICE_HOST", k8sSvc)
	os.Setenv("KUBERNETES_SERVICE_PORT", k8sSvcPort)
	return nil
}

func (m *mainFeature) invokeMainFunction(args string) error {
	os.Args = append(originalArgs, strings.Split(args, " ")...)
	main()
	return nil
}

func (m *mainFeature) theLastLogMessageContains(errormsg string) error {
	lastEntry := m.loghook.LastEntry()
	if errormsg == "none" {
		if lastEntry != nil && len(lastEntry.Message) > 0 {
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

func (m *mainFeature) csiExtensionsPresentIsFalse(expectedStr string) error {
	var expected = strings.ToLower(expectedStr) == "true"
	return monitor.AssertExpectedAndActual(assert.Equal, expected, monitor.PodMonitor.CSIExtensionsPresent,
		fmt.Sprintf("Expected CSIExtensionsPresent flag to be %s, but was %v",
			expectedStr, monitor.PodMonitor.CSIExtensionsPresent))
}

func (m *mainFeature) iInduceError(induced string) error {
	switch induced {
	case "none":
		break
	case "Connect":
		m.k8sapiMock.InducedErrors.Connect = true
	case "DeletePod":
		m.k8sapiMock.InducedErrors.DeletePod = true
	case "GetPod":
		m.k8sapiMock.InducedErrors.GetPod = true
	case "GetVolumeAttachments":
		m.k8sapiMock.InducedErrors.GetVolumeAttachments = true
	case "DeleteVolumeAttachment":
		m.k8sapiMock.InducedErrors.DeleteVolumeAttachment = true
	case "GetPersistentVolumeClaimsInNamespace":
		m.k8sapiMock.InducedErrors.GetPersistentVolumeClaimsInNamespace = true
	case "GetPersistentVolumeClaimsInPod":
		m.k8sapiMock.InducedErrors.GetPersistentVolumeClaimsInPod = true
	case "GetPersistentVolumesInPod":
		m.k8sapiMock.InducedErrors.GetPersistentVolumesInPod = true
	case "IsVolumeAttachmentToPod":
		m.k8sapiMock.InducedErrors.IsVolumeAttachmentToPod = true
	case "GetPersistentVolumeClaimName":
		m.k8sapiMock.InducedErrors.GetPersistentVolumeClaimName = true
	case "GetPersistentVolume":
		m.k8sapiMock.InducedErrors.GetPersistentVolume = true
	case "GetPersistentVolumeClaim":
		m.k8sapiMock.InducedErrors.GetPersistentVolumeClaim = true
	case "GetNode":
		m.k8sapiMock.InducedErrors.GetNode = true
	case "GetNodeWithTimeout":
		m.k8sapiMock.InducedErrors.GetNodeWithTimeout = true
	case "GetVolumeHandleFromVA":
		m.k8sapiMock.InducedErrors.GetVolumeHandleFromVA = true
	case "GetPVNameFromVA":
		m.k8sapiMock.InducedErrors.GetPVNameFromVA = true
	case "ControllerUnpublishVolume":
		m.csiapiMock.InducedErrors.ControllerUnpublishVolume = true
	case "NodeUnpublishVolume":
		m.csiapiMock.InducedErrors.NodeUnpublishVolume = true
	case "NodeUnstageVolume":
		m.csiapiMock.InducedErrors.NodeUnstageVolume = true
	case "ValidateVolumeHostConnectivity":
		m.csiapiMock.InducedErrors.ValidateVolumeHostConnectivity = true
	case "NodeConnected":
		m.csiapiMock.ValidateVolumeHostConnectivityResponse.Connected = true
	case "NodeNotConnected":
		m.csiapiMock.ValidateVolumeHostConnectivityResponse.Connected = false
	case "Unmount":
		gofsutil.GOFSMock.InduceUnmountError = true
	case "LeaderElection":
		m.leaderElect.failLeaderElection = true
	case "StartAPIMonitor":
		m.failStartAPIMonitor = true
	case "CSIClientClose":
		m.csiapiMock.InducedErrors.Close = true
	default:
		return fmt.Errorf("unknown induced error: %s", induced)
	}
	return nil
}

func ScenarioInit(context *godog.ScenarioContext) {
	m := &mainFeature{}
	context.Step(`^a podmon instance$`, m.aPodmonInstance)
	context.Step(`^Podmon env vars set to "([^"]*)":"([^"]*)"$`, m.podmonEnvVarsSetTo)
	context.Step(`^I invoke main with arguments "([^"]*)"$`, m.invokeMainFunction)
	context.Step(`^the last log message contains "([^"]*)"$`, m.theLastLogMessageContains)
	context.Step(`^I induce error "([^"]*)"$`, m.iInduceError)
	context.Step(`^CSIExtensionsPresent is "([^"]*)"`, m.csiExtensionsPresentIsFalse)
}
