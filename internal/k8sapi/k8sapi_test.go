package k8sapi

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
)

func TestDeletePod(t *testing.T) {

	clientset := fake.NewSimpleClientset()
	api := &Client{
		Client: clientset,
	}

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

	podUID := types.UID("test-uid")

	force := true

	// Call the DeletePod function
	err = api.DeletePod(context.Background(), namespace, name, podUID, force)

	assert.NoError(t, err, "DeletePod returned an error")
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
