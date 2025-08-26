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
	"os"
	"sync"
	"testing"

	"podmon/internal/mocks"
	"podmon/internal/utils/constants"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

func TestPodMonitorType_controllerModePodHandler(t *testing.T) {
	kubeSystemNamespace := "kube-system"
	appPodName := "test-pod"
	driverPodName := "powerflex-node-abcdefg"
	nodeName := "worker-1-abcdefg.domain"

	type fields struct {
		Mode                          string
		PodKeyMap                     sync.Map
		PodKeyToControllerPodInfo     sync.Map
		PodKeyToCrashLoopBackOffCount sync.Map
		APIConnected                  bool
		ArrayConnected                bool
		SkipArrayConnectionValidation bool
		CSIExtensionsPresent          bool
		DriverPathStr                 string
		NodeNameToUID                 sync.Map
	}
	type args struct {
		pod       *corev1.Pod
		eventType watch.EventType
	}
	type test struct {
		name               string
		fields             fields
		args               args
		k8sAPI             func(*test) *mocks.K8sMock
		setDriverNamespace func() error
		wantErr            bool
	}
	tests := []test{
		{
			name: "a pod in the driver namespace that is not a driver pod should be ignored",
			fields: fields{
				Mode:                          "controller",
				PodKeyMap:                     sync.Map{},
				PodKeyToControllerPodInfo:     sync.Map{},
				PodKeyToCrashLoopBackOffCount: sync.Map{},
				APIConnected:                  true,
				ArrayConnected:                true,
				SkipArrayConnectionValidation: false,
				CSIExtensionsPresent:          true,
				DriverPathStr:                 "csi-vxflexos.dellemc.com",
				NodeNameToUID:                 sync.Map{},
			},
			args: args{
				// an application pod in the same namespace as the driver, but not a driver pod
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						// pod is conspicuously missing the constants.DriverPodLabelKey and constants.DriverPodLabelValue
						Labels:    map[string]string{},
						Name:      appPodName,
						Namespace: kubeSystemNamespace,
					},
					Spec: corev1.PodSpec{
						NodeName: nodeName,
					},
					Status: corev1.PodStatus{
						Message: "",
						Reason:  "",
					},
				},
				eventType: watch.Error,
			},
			k8sAPI: func(_ *test) *mocks.K8sMock {
				return &mocks.K8sMock{}
			},
			setDriverNamespace: func() error {
				return os.Setenv("MY_POD_NAMESPACE", kubeSystemNamespace)
			},
			wantErr: true,
		},
		{
			name: "a driver pod in the kube-system namespace with the expected label",
			fields: fields{
				Mode:                          "controller",
				PodKeyMap:                     sync.Map{},
				PodKeyToControllerPodInfo:     sync.Map{},
				PodKeyToCrashLoopBackOffCount: sync.Map{},
				APIConnected:                  true,
				ArrayConnected:                true,
				SkipArrayConnectionValidation: false,
				CSIExtensionsPresent:          true,
				DriverPathStr:                 "csi-vxflexos.dellemc.com",
				NodeNameToUID:                 sync.Map{},
			},
			args: args{
				// a driver pod in the driver namespace
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						// a driver pod should have the label key constants.DriverPodLabelKey and label value constants.DriverPodLabelValue
						Labels: map[string]string{constants.DriverPodLabelKey: constants.DriverPodLabelValue},
						Name:   driverPodName,
						// accidentally installed the driver in the kube-system namespace
						Namespace: kubeSystemNamespace,
					},
					Spec: corev1.PodSpec{
						NodeName: nodeName,
					},
					Status: corev1.PodStatus{
						Message: "",
						Reason:  "",
					},
				},
				eventType: watch.Error,
			},
			k8sAPI: func(tt *test) *mocks.K8sMock {
				k8sClient := &mocks.K8sMock{}
				k8sClient.AddPod(tt.args.pod)
				return k8sClient
			},
			setDriverNamespace: func() error {
				return os.Setenv("MY_POD_NAMESPACE", kubeSystemNamespace)
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			K8sAPI = tt.k8sAPI(&tt)

			if err := tt.setDriverNamespace(); err != nil {
				t.Errorf("failed to set driver namespace: %s", err.Error())
			}

			cm := &PodMonitorType{
				Mode:                          tt.fields.Mode,
				PodKeyMap:                     tt.fields.PodKeyMap,
				PodKeyToControllerPodInfo:     tt.fields.PodKeyToControllerPodInfo,
				PodKeyToCrashLoopBackOffCount: tt.fields.PodKeyToCrashLoopBackOffCount,
				APIConnected:                  tt.fields.APIConnected,
				ArrayConnected:                tt.fields.ArrayConnected,
				SkipArrayConnectionValidation: tt.fields.SkipArrayConnectionValidation,
				CSIExtensionsPresent:          tt.fields.CSIExtensionsPresent,
				DriverPathStr:                 tt.fields.DriverPathStr,
				NodeNameToUID:                 tt.fields.NodeNameToUID,
			}
			if err := cm.controllerModePodHandler(tt.args.pod, tt.args.eventType); (err != nil) != tt.wantErr {
				t.Errorf("PodMonitorType.controllerModePodHandler() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
