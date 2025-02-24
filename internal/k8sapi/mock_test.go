package k8sapi

import (
	"context"
	"podmon/internal/mocks"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

func TestMockInitialize(t *testing.T) {
	mock := &mocks.K8sMock{}

	mock.Initialize()

	assert.NotNil(t, mock.Watcher)
	assert.IsType(t, &watch.RaceFreeFakeWatcher{}, mock.Watcher)
}

func TestMockAddPV(t *testing.T) {
	mock := &mocks.K8sMock{}

	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pv",
		},
	}

	mock.AddPV(pv)

	assert.Equal(t, pv, mock.NameToPV[pv.ObjectMeta.Name])
}

func TestMockAddVA(t *testing.T) {
	mock := &mocks.K8sMock{}

	va := &storagev1.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-va",
		},
	}

	mock.AddVA(va)

	assert.Equal(t, va, mock.NameToVolumeAttachment[va.ObjectMeta.Name])
}

func TestMockConnect(t *testing.T) {
	mock := &K8sMock{}

	// Test case 1: No induced error
	err := mock.Connect(nil)
	assert.NoError(t, err)

	// Test case 2: Induced error
	mock.InducedErrors.Connect = true
	err = mock.Connect(nil)
	assert.Error(t, err)
	assert.Equal(t, "induced Connect error", err.Error())
}

func TestMockGetClient(t *testing.T) {
	mock := &K8sMock{}

	clientset := mock.GetClient()

	assert.Nil(t, clientset)
}

func TestMockDeletePod(t *testing.T) {
	mock := &K8sMock{}

	namespace := "test-namespace"
	name := "test-pod"
	key := mock.getKey(namespace, name)

	// Add a pod to the mock
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
	mock.AddPod(pod)

	mock.InducedErrors.DeletePod = true
	err := mock.DeletePod(context.Background(), namespace, name, "", false)
	assert.Error(t, err)
	assert.Equal(t, "induced DeletePod error", err.Error())
	assert.Contains(t, mock.KeyToPod, key)
}
