package k8sapi

import (
	"context"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
)

func TestDeletePod(t *testing.T) {

	clientset := fake.NewSimpleClientset()
	//k8mock := &K8sMock{}
	// api := &Client{
	// 	Client: k8mock.GetClient(),
	// }

	// Define the test namespace and name
	namespace := "test-namespace"
	name := "test-name"

	// Create a test pod to simulate an existing pod
	_, err := clientset.CoreV1().Pods(namespace).Create(context.Background(), &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			UID:  types.UID("test-uid"),
		},
	}, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create pod: %s", err)
	}

	// Define the test pod UID
	//podUID := types.UID("test-uid")

	// Define the test force flag
	//force := true

	// Call the DeletePod function
	//err = api.DeletePod(context.Background(), namespace, name, podUID, force)

	// Check if the error is nil
	// assert.NoError(t, err, "DeletePod returned an error")

	// // Try to get the pod after deletion
	// _, err = clientset.CoreV1().Pods(namespace).Get(context.Background(), name, metav1.GetOptions{})
	// // Assert that the error is non-nil (meaning the pod does not exist)
	// assert.Error(t, err, "DeletePod did not delete the pod")
}
