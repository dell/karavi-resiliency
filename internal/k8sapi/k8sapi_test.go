package k8sapi

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
)

func createClient() *fake.Clientset {
	return fake.NewSimpleClientset()
}

func TestDeletePod(t *testing.T) {
	mockClient := createClient()
	api := &Client{
		Client: mockClient,
	}

	// Define the test namespace and name
	namespace := "test-namespace"
	name := "test-name"

	// Create a test pod to simulate an existing pod
	_, err := mockClient.CoreV1().Pods(namespace).Create(context.Background(), &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			UID:  types.UID("test-uid"),
		},
	}, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create pod: %s", err)
	}

	podUID := types.UID("test-uid")

	force := true

	// Call the DeletePod function
	err = api.DeletePod(context.Background(), namespace, name, podUID, force)

	assert.NoError(t, err, "DeletePod returned an error")
}

func TestGetPod(t *testing.T) {
	mockClient := createClient()
	api := &Client{
		Client: mockClient,
	}

	// Define the test namespace and name
	namespace := "test-namespace"
	name := "test-pod"

	// Create a test pod to simulate an existing pod
	expectedPod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	_, err := mockClient.CoreV1().Pods(namespace).Create(context.Background(), expectedPod, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test pod: %s", err)
	}

	// Call the GetPod function
	pod, err := api.GetPod(context.Background(), namespace, name)

	// Validate the results
	assert.NoError(t, err, "GetPod returned an error")
	assert.NotNil(t, pod, "GetPod returned nil pod")
	assert.Equal(t, expectedPod.Name, pod.Name, "Pod name does not match")
	assert.Equal(t, expectedPod.Namespace, pod.Namespace, "Pod namespace does not match")
}

func TestGetVolumeAttachments(t *testing.T) {
	mockClient := createClient()
	api := &Client{
		Client: mockClient,
	}

	// Create a test volume attachment to simulate existing volume attachments
	expectedVolumeAttachment1 := &storagev1.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "volume-attachment-1",
		},
	}
	expectedVolumeAttachment2 := &storagev1.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "volume-attachment-2",
		},
	}

	// Add volume attachments to the fake client
	_, err := mockClient.StorageV1().VolumeAttachments().Create(context.Background(), expectedVolumeAttachment1, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test volume attachment 1: %s", err)
	}
	_, err = mockClient.StorageV1().VolumeAttachments().Create(context.Background(), expectedVolumeAttachment2, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test volume attachment 2: %s", err)
	}

	// Call the GetVolumeAttachments function
	volumeAttachments, err := api.GetVolumeAttachments(context.Background())

	// Validate the results
	assert.NoError(t, err, "GetVolumeAttachments returned an error")
	assert.NotNil(t, volumeAttachments, "GetVolumeAttachments returned nil")
	assert.Len(t, volumeAttachments.Items, 2, "Expected 2 volume attachments")
	assert.Equal(t, "volume-attachment-1", volumeAttachments.Items[0].Name, "VolumeAttachment name does not match")
	assert.Equal(t, "volume-attachment-2", volumeAttachments.Items[1].Name, "VolumeAttachment name does not match")
}

func TestDeleteVolumeAttachment(t *testing.T) {
	mockClient := createClient()
	api := &Client{
		Client:                    mockClient,
		volumeAttachmentNameToKey: make(map[string]string),
		volumeAttachmentCache:     make(map[string]*storagev1.VolumeAttachment),
	}

	// Define the test volume attachment
	vaname := "test-volume-attachment"
	vaKey := "test-va-key"

	// Create a test volume attachment to simulate an existing one
	testVolumeAttachment := &storagev1.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name: vaname,
		},
	}
	_, err := mockClient.StorageV1().VolumeAttachments().Create(context.Background(), testVolumeAttachment, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test volume attachment: %s", err)
	}

	// Add the volume attachment to the client's cache
	api.volumeAttachmentNameToKey[vaname] = vaKey
	api.volumeAttachmentCache[vaKey] = testVolumeAttachment

	// Call the DeleteVolumeAttachment function
	err = api.DeleteVolumeAttachment(context.Background(), vaname)

	// Validate the results
	assert.NoError(t, err, "DeleteVolumeAttachment returned an error")
	_, err = mockClient.StorageV1().VolumeAttachments().Get(context.Background(), vaname, metav1.GetOptions{})
	assert.Error(t, err, "Expected an error when getting a deleted volume attachment")
	assert.Nil(t, api.volumeAttachmentCache[vaKey], "Expected volume attachment to be removed from cache")
	assert.Empty(t, api.volumeAttachmentNameToKey[vaname], "Expected volume attachment name to be removed from name to key map")
}

func TestGetPersistentVolumeClaimsInNamespace(t *testing.T) {
	mockClient := createClient()
	api := &Client{
		Client: mockClient,
	}

	// Define the test namespace
	namespace := "test-namespace"

	// Create test PVCs to simulate existing ones
	expectedPVC1 := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pvc-1",
			Namespace: namespace,
		},
	}
	expectedPVC2 := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pvc-2",
			Namespace: namespace,
		},
	}

	// Add the PVCs to the fake client
	_, err := mockClient.CoreV1().PersistentVolumeClaims(namespace).Create(context.Background(), expectedPVC1, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test PVC 1: %s", err)
	}
	_, err = mockClient.CoreV1().PersistentVolumeClaims(namespace).Create(context.Background(), expectedPVC2, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test PVC 2: %s", err)
	}

	// Call the GetPersistentVolumeClaimsInNamespace function
	pvcList, err := api.GetPersistentVolumeClaimsInNamespace(context.Background(), namespace)

	// Validate the results
	assert.NoError(t, err, "GetPersistentVolumeClaimsInNamespace returned an error")
	assert.NotNil(t, pvcList, "GetPersistentVolumeClaimsInNamespace returned nil")
	assert.Len(t, pvcList.Items, 2, "Expected 2 persistent volume claims")
	assert.Equal(t, "pvc-1", pvcList.Items[0].Name, "PVC name does not match")
	assert.Equal(t, "pvc-2", pvcList.Items[1].Name, "PVC name does not match")
}

func TestGetCachedVolumeAttachment(t *testing.T) {
	mockClient := createClient()
	api := &Client{
		Client:                    mockClient,
		volumeAttachmentCache:     make(map[string]*storagev1.VolumeAttachment),
		volumeAttachmentNameToKey: make(map[string]string),
	}

	// Define the test PV name and node name
	pvName := "test-pv"
	nodeName := "test-node"
	vaName := "test-va"
	key := fmt.Sprintf("%s/%s", pvName, nodeName)

	// Create a test volume attachment to simulate an existing one
	testVolumeAttachment := &storagev1.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name: vaName,
		},
		Spec: storagev1.VolumeAttachmentSpec{
			Source: storagev1.VolumeAttachmentSource{
				PersistentVolumeName: &pvName,
			},
			NodeName: nodeName,
		},
	}
	_, err := mockClient.StorageV1().VolumeAttachments().Create(context.Background(), testVolumeAttachment, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test volume attachment: %s", err)
	}

	// Test cache miss scenario (empty initial cache)
	va, err := api.GetCachedVolumeAttachment(context.Background(), pvName, nodeName)
	assert.NoError(t, err, "GetCachedVolumeAttachment returned an error on cache miss")
	assert.NotNil(t, va, "GetCachedVolumeAttachment returned nil on cache miss")
	assert.Equal(t, vaName, va.Name, "Volume attachment name does not match on cache miss")
	assert.Equal(t, key, api.volumeAttachmentNameToKey[vaName], "Volume attachment key does not match in cache")
	assert.Equal(t, testVolumeAttachment, api.volumeAttachmentCache[key], "Volume attachment does not match in cache after rebuild")

	// Test cache hit scenario
	va, err = api.GetCachedVolumeAttachment(context.Background(), pvName, nodeName)
	assert.NoError(t, err, "GetCachedVolumeAttachment returned an error on cache hit")
	assert.NotNil(t, va, "GetCachedVolumeAttachment returned nil on cache hit")
	assert.Equal(t, vaName, va.Name, "Volume attachment name does not match on cache hit")
}

func TestGetPersistentVolumeClaimsInPod(t *testing.T) {

	mockClient := createClient()
	api := &Client{
		Client: mockClient,
	}

	// Create a test pod
	testPod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-namespace",
		},
		Spec: v1.PodSpec{
			Volumes: []v1.Volume{
				{
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
							ClaimName: "test-pvc",
						},
					},
				},
			},
		},
	}

	// Create a test PVC
	testPVC := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-namespace",
			Name:      "test-pvc",
		},
	}

	// Add the test PVC to the fake client
	_, err := mockClient.CoreV1().PersistentVolumeClaims("test-namespace").Create(context.Background(), testPVC, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test PVC: %s", err)
	}

	// Call the function
	pvcs, err := api.GetPersistentVolumeClaimsInPod(context.Background(), testPod)

	// Validate the results
	assert.NoError(t, err, "GetPersistentVolumeClaimsInPod returned an error")
	assert.NotNil(t, pvcs, "GetPersistentVolumeClaimsInPod returned nil")
	assert.Len(t, pvcs, 1, "Expected 1 persistent volume claim")
	assert.Equal(t, "test-pvc", pvcs[0].ObjectMeta.Name, "PVC name does not match")
}

func TestUpdateTaint(t *testing.T) {
	taintKey := "key1"
	effect := v1.TaintEffectNoSchedule

	tests := []struct {
		name           string
		node           *v1.Node
		remove         bool
		expectedOp     string
		expectedPatch  bool
		expectedTaints []v1.Taint
	}{
		{
			name: "Add new taint",
			node: &v1.Node{
				Spec: v1.NodeSpec{
					Taints: []v1.Taint{},
				},
			},
			remove:        false,
			expectedOp:    taintAdd,
			expectedPatch: true,
			expectedTaints: []v1.Taint{
				{
					Key:    taintKey,
					Effect: effect,
				},
			},
		},
		{
			name: "Remove existing taint",
			node: &v1.Node{
				Spec: v1.NodeSpec{
					Taints: []v1.Taint{
						{
							Key:    taintKey,
							Effect: effect,
						},
					},
				},
			},
			remove:         true,
			expectedOp:     taintRemove,
			expectedPatch:  true,
			expectedTaints: []v1.Taint{},
		},
		{
			name: "Taint already exists",
			node: &v1.Node{
				Spec: v1.NodeSpec{
					Taints: []v1.Taint{
						{
							Key:    taintKey,
							Effect: effect,
						},
					},
				},
			},
			remove:        false,
			expectedOp:    taintAlreadyExists,
			expectedPatch: false,
			expectedTaints: []v1.Taint{
				{
					Key:    taintKey,
					Effect: effect,
				},
			},
		},
		{
			name: "Taint does not exist",
			node: &v1.Node{
				Spec: v1.NodeSpec{
					Taints: []v1.Taint{},
				},
			},
			remove:         true,
			expectedOp:     taintDoesNotExist,
			expectedPatch:  false,
			expectedTaints: []v1.Taint{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op, patch := updateTaint(tt.node, taintKey, effect, tt.remove)
			if op != tt.expectedOp {
				t.Errorf("expected operation %s, got %s", tt.expectedOp, op)
			}
			if patch != tt.expectedPatch {
				t.Errorf("expected patch %v, got %v", tt.expectedPatch, patch)
			}
			if len(tt.node.Spec.Taints) != len(tt.expectedTaints) {
				t.Errorf("expected taints %v, got %v", tt.expectedTaints, tt.node.Spec.Taints)
			}
		})
	}
}

func TestTaintExists(t *testing.T) {
	taintKey := "key1"
	effect := v1.TaintEffectNoSchedule

	tests := []struct {
		name     string
		node     *v1.Node
		expected bool
	}{
		{
			name: "Taint exists",
			node: &v1.Node{
				Spec: v1.NodeSpec{
					Taints: []v1.Taint{
						{
							Key:    taintKey,
							Effect: effect,
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "Taint does not exist",
			node: &v1.Node{
				Spec: v1.NodeSpec{
					Taints: []v1.Taint{},
				},
			},
			expected: false,
		},
		{
			name: "Different taint key",
			node: &v1.Node{
				Spec: v1.NodeSpec{
					Taints: []v1.Taint{
						{
							Key:    "differentKey",
							Effect: effect,
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "Different taint effect",
			node: &v1.Node{
				Spec: v1.NodeSpec{
					Taints: []v1.Taint{
						{
							Key:    taintKey,
							Effect: v1.TaintEffectPreferNoSchedule,
						},
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := taintExists(tt.node, taintKey, effect)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
