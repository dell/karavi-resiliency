// Copyright © 2021-2022 Dell Inc. or its subsidiaries. All Rights Reserved.
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
package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"podmon/internal/csiapi"
	"podmon/internal/k8sapi"
	"podmon/internal/mocks"
	"podmon/internal/monitor"
	"strings"
	"sync"
	"time"

	"github.com/dell/csmlog"
	"github.com/dell/gofsutil"
	"github.com/cucumber/godog"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"k8s.io/client-go/kubernetes"
)

type mainFeature struct {
	// Buffer to capture csmlog output
	logbuf              *bytes.Buffer
	k8sapiMock          *mocks.K8sMock
	csiapiMock          *mocks.CSIMock
	leaderElect         *mockLeaderElect
	failStartAPIMonitor bool
}

var (
	saveOriginalArgs sync.Once
	originalArgs     []string
)

func (m *mainFeature) aPodmonInstance() error {
	if m.logbuf == nil {
		m.logbuf = &bytes.Buffer{}
	} else {
		m.logbuf.Reset()
	}
	csmlog.SetOutput(m.logbuf)
	monitor.PodMonitor.CSIExtensionsPresent = false
	m.csiapiMock = new(mocks.CSIMock)
	m.k8sapiMock = new(mocks.K8sMock)
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

func (le *mockLeaderElect) WithNamespace(_ string) {
}

func (m *mainFeature) mockGetCSIClient(_ string, _ ...grpc.DialOption) (csiapi.CSIApi, error) {
	return m.csiapiMock, nil
}

func (m *mainFeature) mockStartPodMonitor(_ k8sapi.K8sAPI, _ kubernetes.Interface, _, _ string, _ time.Duration) {
}

func (m *mainFeature) mockStartNodeMonitor(_ k8sapi.K8sAPI, _ kubernetes.Interface, _, _ string, _ time.Duration) {
}

func (m *mainFeature) mockStartAPIMonitor(_ k8sapi.K8sAPI, _, _, _ time.Duration, _ func(interval time.Duration) bool) error {
	if m.failStartAPIMonitor {
		return fmt.Errorf("induced StorageAPIMonitor failure")
	}
	return nil
}

func (m *mainFeature) mockPodMonWait() bool {
	return true
}

func (m *mainFeature) mockLeaderElection(_ func(ctx context.Context)) leaderElection {
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
	output := m.logbuf.String()
	if errormsg == "none" {
		if len(output) > 0 {
			return nil
		}
		return nil
	}
	if strings.Contains(output, errormsg) {
		return nil
	}
	return fmt.Errorf("expected error message to contain: %s, but output was: %s", errormsg, output)
}

func (m *mainFeature) csiExtensionsPresentIsFalse(expectedStr string) error {
	expected := strings.ToLower(expectedStr) == "true"
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
