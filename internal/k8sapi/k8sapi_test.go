package k8sapi

import (
	"context"
	"fmt"
	"testing"
	"time"

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

func TestGetPersistentVolumesInPod(t *testing.T) {
	mockClient := createClient()
	api := &Client{
		Client: mockClient,
	}

	// Define the test namespace, PVC, and PV names
	namespace := "test-namespace"
	pvcName := "test-pvc"
	pvName := "test-pv"

	// Create a test pod
	testPod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: namespace,
		},
		Spec: v1.PodSpec{
			Volumes: []v1.Volume{
				{
					Name: "test-volume",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvcName,
						},
					},
				},
			},
		},
	}

	// Create a test PVC to simulate an existing one bound to a PV
	testPVC := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: namespace,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			VolumeName: pvName,
		},
		Status: v1.PersistentVolumeClaimStatus{
			Phase: v1.ClaimBound,
		},
	}

	// Create a test PV
	testPV := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: pvName,
		},
	}

	// Add the resources to the fake client
	_, err := mockClient.CoreV1().Pods(namespace).Create(context.Background(), testPod, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test pod: %s", err)
	}
	_, err = mockClient.CoreV1().PersistentVolumeClaims(namespace).Create(context.Background(), testPVC, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test PVC: %s", err)
	}
	_, err = mockClient.CoreV1().PersistentVolumes().Create(context.Background(), testPV, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test PV: %s", err)
	}

	// Call the GetPersistentVolumesInPod function
	pvs, err := api.GetPersistentVolumesInPod(context.Background(), testPod)

	// Validate the results
	assert.NoError(t, err, "GetPersistentVolumesInPod returned an error")
	assert.NotNil(t, pvs, "GetPersistentVolumesInPod returned nil")
	assert.Len(t, pvs, 1, "Expected 1 persistent volume")
	assert.Equal(t, pvName, pvs[0].Name, "Persistent volume name does not match")
}

func TestIsVolumeAttachmentToPod(t *testing.T) {
	mockClient := createClient()
	api := &Client{
		Client: mockClient,
	}

	namespace := "test-namespace"
	nodeName := "test-node"
	pvcName := "test-pvc"
	pvName := "test-pv"
	vaName := "test-va"

	// Create a test pod
	testPod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: namespace,
		},
		Spec: v1.PodSpec{
			NodeName: nodeName,
			Volumes: []v1.Volume{
				{
					Name: "test-volume",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvcName,
						},
					},
				},
			},
		},
	}

	// Create a test PVC
	testPVC := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: namespace,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			VolumeName: pvName,
		},
	}

	// Create a test VolumeAttachment
	testVA := &storagev1.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name: vaName,
		},
		Spec: storagev1.VolumeAttachmentSpec{
			NodeName: nodeName,
			Source: storagev1.VolumeAttachmentSource{
				PersistentVolumeName: &pvName,
			},
		},
	}

	// Add the resources to the fake client
	_, err := mockClient.CoreV1().Pods(namespace).Create(context.Background(), testPod, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test pod: %s", err)
	}

	_, err = mockClient.CoreV1().PersistentVolumeClaims(namespace).Create(context.Background(), testPVC, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test PVC: %s", err)
	}

	// Call the IsVolumeAttachmentToPod function
	result, err := api.IsVolumeAttachmentToPod(context.Background(), testVA, testPod)

	// Validate the results
	assert.NoError(t, err, "IsVolumeAttachmentToPod returned an error")
	assert.True(t, result, "Expected the volume attachment to be associated with the pod")

	// Test when pod NodeName doesn't match VolumeAttachment NodeName
	testPod.Spec.NodeName = "different-node"
	result, err = api.IsVolumeAttachmentToPod(context.Background(), testVA, testPod)
	assert.NoError(t, err, "IsVolumeAttachmentToPod returned an error")
	assert.False(t, result, "Expected the volume attachment not to be associated with the pod due to different node")

	// Test when PVC's VolumeName in pod does not match VolumeAttachment PersistentVolumeName
	testPod.Spec.NodeName = nodeName // Set it back to the original matching node
	differentPVC := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "different-pvc",
			Namespace: namespace,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			VolumeName: "different-pv",
		},
	}
	_, err = mockClient.CoreV1().PersistentVolumeClaims(namespace).Create(context.Background(), differentPVC, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create different PVC: %s", err)
	}

	testPod.Spec.Volumes[0].VolumeSource.PersistentVolumeClaim.ClaimName = "different-pvc"
	result, err = api.IsVolumeAttachmentToPod(context.Background(), testVA, testPod)
	assert.NoError(t, err, "IsVolumeAttachmentToPod returned an error")
	assert.False(t, result, "Expected the volume attachment not to be associated with the pod due to non-matching PVC volume name")
}

func TestGetPersistentVolumeClaimName(t *testing.T) {
	mockClient := createClient()
	api := &Client{
		Client: mockClient,
	}

	// Define the test PV name and PVC details
	pvName := "test-pv"
	pvcNamespace := "test-namespace"
	pvcName := "test-pvc"

	// Create a test PV with a ClaimRef
	testPV := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: pvName,
		},
		Spec: v1.PersistentVolumeSpec{
			ClaimRef: &v1.ObjectReference{
				Kind:      "PersistentVolumeClaim",
				Namespace: pvcNamespace,
				Name:      pvcName,
			},
		},
	}

	// Add the PV to the fake client
	_, err := mockClient.CoreV1().PersistentVolumes().Create(context.Background(), testPV, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test PV: %s", err)
	}

	// Call the GetPersistentVolumeClaimName function
	result, err := api.GetPersistentVolumeClaimName(context.Background(), pvName)

	// Validate the results
	assert.NoError(t, err, "GetPersistentVolumeClaimName returned an error")
	assert.Equal(t, pvcNamespace+"/"+pvcName, result, "Persistent volume claim name does not match")

	// Test with a PV without a ClaimRef
	emptyPV := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: "empty-pv",
		},
	}

	// Add the empty PV to the fake client
	_, err = mockClient.CoreV1().PersistentVolumes().Create(context.Background(), emptyPV, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create empty PV: %s", err)
	}

	// Call the GetPersistentVolumeClaimName function
	result, err = api.GetPersistentVolumeClaimName(context.Background(), "empty-pv")

	// Validate the results
	assert.NoError(t, err, "GetPersistentVolumeClaimName returned an error")
	assert.Equal(t, "", result, "Expected to return an empty string for a PV without a ClaimRef")
}

func TestGetPersistentVolume(t *testing.T) {
	mockClient := createClient()
	api := &Client{
		Client: mockClient,
	}

	pvName := "test-pv"
	testPV := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: pvName,
		},
	}

	// Add the PV to the fake client
	_, err := mockClient.CoreV1().PersistentVolumes().Create(context.Background(), testPV, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test PV: %s", err)
	}

	// Test successful retrieval of the PV
	pv, err := api.GetPersistentVolume(context.Background(), pvName)
	assert.NoError(t, err, "GetPersistentVolume returned an error")
	assert.NotNil(t, pv, "GetPersistentVolume returned nil")
	assert.Equal(t, pvName, pv.Name, "Persistent volume name does not match")

	// Test retrieval of a non-existent PV
	// pv, err = api.GetPersistentVolume(context.Background(), "non-existent-pv")
	// assert.Error(t, err, "GetPersistentVolume should have returned an error for a non-existent PV")
	// assert.Nil(t, pv, "GetPersistentVolume should have returned nil for a non-existent PV")

	// Test retrieval with a nil client
	api.Client = nil
	pv, err = api.GetPersistentVolume(context.Background(), pvName)
	assert.Error(t, err, "GetPersistentVolume should have returned an error for a nil client")
	assert.Nil(t, pv, "GetPersistentVolume should have returned nil for a nil client")
	assert.Equal(t, "No connection", err.Error(), "Error message does not match for a nil client")
}

func TestGetNode(t *testing.T) {
	mockClient := createClient()
	api := &Client{
		Client: mockClient,
	}

	nodeName := "test-node"
	testNode := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeName,
		},
	}

	// Add the node to the fake client
	_, err := mockClient.CoreV1().Nodes().Create(context.Background(), testNode, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test node: %s", err)
	}

	// Test successful retrieval of the node
	node, err := api.GetNode(context.Background(), nodeName)
	assert.NoError(t, err, "GetNode returned an error")
	assert.NotNil(t, node, "GetNode returned nil")
	assert.Equal(t, nodeName, node.Name, "Node name does not match")

	// // Test retrieval of a non-existent node
	// node, err = api.GetNode(context.Background(), "non-existent-node")
	// assert.Error(t, err, "GetNode should have returned an error for a non-existent node")
	// assert.Nil(t, node, "GetNode should have returned nil for a non-existent node")
}

func TestGetVolumeHandleFromVA(t *testing.T) {
	mockClient := createClient()
	api := &Client{
		Client: mockClient,
	}

	// Define test data
	vaName := "test-va"
	pvName := "test-pv"
	volumeHandle := "test-volume-handle"

	// Create a test PersistentVolume
	testPV := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: pvName,
		},
		Spec: v1.PersistentVolumeSpec{
			PersistentVolumeSource: v1.PersistentVolumeSource{
				CSI: &v1.CSIPersistentVolumeSource{
					VolumeHandle: volumeHandle,
				},
			},
		},
	}

	// Create a test VolumeAttachment
	testVA := &storagev1.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name: vaName,
		},
		Spec: storagev1.VolumeAttachmentSpec{
			Source: storagev1.VolumeAttachmentSource{
				PersistentVolumeName: &pvName,
			},
		},
	}

	// Add the PV to the fake client
	_, err := mockClient.CoreV1().PersistentVolumes().Create(context.Background(), testPV, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test PV: %s", err)
	}

	// Call the function under test
	handle, err := api.GetVolumeHandleFromVA(context.Background(), testVA)

	// Validate the results
	assert.NoError(t, err, "GetVolumeHandleFromVA returned an error")
	assert.Equal(t, volumeHandle, handle, "Volume handle does not match")

	// Test with a non-CSI volume
	nonCSIPV := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: "non-csi-pv",
		},
		Spec: v1.PersistentVolumeSpec{
			PersistentVolumeSource: v1.PersistentVolumeSource{
				// No CSI source here
			},
		},
	}

	// Add the non-CSI PV to the fake client
	_, err = mockClient.CoreV1().PersistentVolumes().Create(context.Background(), nonCSIPV, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create non-CSI PV: %s", err)
	}

	nonCSIVA := &storagev1.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "non-csi-va",
		},
		Spec: storagev1.VolumeAttachmentSpec{
			Source: storagev1.VolumeAttachmentSource{
				PersistentVolumeName: &nonCSIPV.Name,
			},
		},
	}

	// Call the function under test with the non-CSI volume
	handle, err = api.GetVolumeHandleFromVA(context.Background(), nonCSIVA)

	// Validate the results
	assert.Error(t, err, "GetVolumeHandleFromVA expected to return an error for non-CSI PV")
	assert.EqualError(t, err, "PV is not a CSI volume")
	assert.Equal(t, "", handle, "Expected volume handle to be empty for non-CSI PV")
}

func TestGetPVNameFromVA(t *testing.T) {
	api := &Client{}

	vaName := "test-va"
	pvName := "test-pv"

	// Test with a valid PersistentVolumeName
	testVA := &storagev1.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name: vaName,
		},
		Spec: storagev1.VolumeAttachmentSpec{
			Source: storagev1.VolumeAttachmentSource{
				PersistentVolumeName: &pvName,
			},
		},
	}

	result, err := api.GetPVNameFromVA(testVA)
	assert.NoError(t, err, "GetPVNameFromVA returned an error")
	assert.Equal(t, pvName, result, "PersistentVolumeName does not match")

	// Test with a missing PersistentVolumeName
	testVAWithoutPVName := &storagev1.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name: vaName,
		},
		Spec: storagev1.VolumeAttachmentSpec{
			Source: storagev1.VolumeAttachmentSource{
				PersistentVolumeName: nil, // Simulating missing PersistentVolumeName
			},
		},
	}

	result, err = api.GetPVNameFromVA(testVAWithoutPVName)
	expectedError := fmt.Sprintf("Could not find PersistentVolume from VolumeAttachment %s", vaName)
	assert.Error(t, err, "GetPVNameFromVA should have returned an error")
	assert.EqualError(t, err, expectedError)
	assert.Equal(t, "", result, "Expected PersistentVolumeName to be empty")

}

func TestGetNodeWithTimeout(t *testing.T) {
	mockClient := createClient()
	api := &Client{
		Client: mockClient,
	}

	// Create a test node
	nodeName := "test-node"
	testNode := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeName,
		},
	}

	// Add the node to the fake client
	_, err := mockClient.CoreV1().Nodes().Create(context.Background(), testNode, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test node: %s", err)
	}

	// Test successful retrieval of the node within the timeout
	t.Run("successful retrieval", func(t *testing.T) {
		duration := 2 * time.Second
		node, err := api.GetNodeWithTimeout(duration, nodeName)
		assert.NoError(t, err, "GetNodeWithTimeout returned an error")
		assert.NotNil(t, node, "GetNodeWithTimeout returned nil")
		assert.Equal(t, nodeName, node.Name, "Node name does not match")
	})

	// Test retrieval with an immediate timeout to simulate timeout scenario
	// t.Run("timeout scenario", func(t *testing.T) {
	// 	// We simulate a timeout by setting a very short timeout duration.
	// 	duration := 0 * time.Nanosecond
	// 	node, err := api.GetNodeWithTimeout(duration, nodeName)
	// 	assert.Error(t, err, "GetNodeWithTimeout should have returned an error for a timeout")
	// 	assert.Nil(t, node, "GetNodeWithTimeout should have returned nil for a timeout")
	// })
}

func TestTaintNode(t *testing.T) {
	mockClient := createClient()
	api := &Client{
		Client: mockClient,
	}

	nodeName := "test-node"
	taintKey := "test-key"
	effect := v1.TaintEffectNoSchedule

	testNode := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeName,
		},
		Spec: v1.NodeSpec{
			Taints: []v1.Taint{},
		},
	}

	// Add the node to the fake client
	_, err := mockClient.CoreV1().Nodes().Create(context.Background(), testNode, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test node: %s", err)
	}

	t.Run("add taint", func(t *testing.T) {
		err := api.TaintNode(context.Background(), nodeName, taintKey, effect, false)
		assert.NoError(t, err)

		node, err := api.GetNode(context.Background(), nodeName)
		assert.NoError(t, err)
		assert.Len(t, node.Spec.Taints, 1)
		assert.Equal(t, node.Spec.Taints[0].Key, taintKey)
		assert.Equal(t, node.Spec.Taints[0].Effect, effect)
	})

	t.Run("taint already exists", func(t *testing.T) {
		err := api.TaintNode(context.Background(), nodeName, taintKey, effect, false)
		assert.NoError(t, err)

		node, err := api.GetNode(context.Background(), nodeName)
		assert.NoError(t, err)
		assert.Len(t, node.Spec.Taints, 1)
		assert.Equal(t, node.Spec.Taints[0].Key, taintKey)
		assert.Equal(t, node.Spec.Taints[0].Effect, effect)
	})

	t.Run("remove taint", func(t *testing.T) {
		err := api.TaintNode(context.Background(), nodeName, taintKey, effect, true)
		assert.NoError(t, err)

		node, err := api.GetNode(context.Background(), nodeName)
		assert.NoError(t, err)
		assert.Len(t, node.Spec.Taints, 0)
	})

	t.Run("remove non-existing taint", func(t *testing.T) {
		err := api.TaintNode(context.Background(), nodeName, taintKey, effect, true)
		assert.NoError(t, err)

		node, err := api.GetNode(context.Background(), nodeName)
		assert.NoError(t, err)
		assert.Len(t, node.Spec.Taints, 0)
	})
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
