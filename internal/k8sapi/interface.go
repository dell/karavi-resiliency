// Package k8sapi provides an interface for access Kubernetes objects.
package k8sapi

// k8sapi package provides facilities for csi-drivers to call the kubernetes API from their containers.
// This is needed for some special use cases, like inspecting PVs or handling fail-over of pods from node failure.

import (
	"context"
	"time"

	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

// K8sApi provides an API interface to kubernetes.
type K8sApi interface {
	// Connect connects to the Kubernetes system API
	Connect(kubeconfig *string) error

	// GetClient returns the kubernetes Clientset.
	GetClient() *kubernetes.Clientset

	// GetContext returns a context object for a certain timeout duration
	GetContext(duration time.Duration) (context.Context, context.CancelFunc)

	// DeletePod deletes a pod of the given namespace and name, an optionally uses force deletion.
	DeletePod(ctx context.Context, namespace, name string, force bool) error

	// GetPod retrieves a pod of the give namespace and name
	GetPod(ctx context.Context, namespace, name string) (*v1.Pod, error)

	// GetVolumeAttachments gets all the volume attachments in the K8S system
	GetVolumeAttachments(ctx context.Context) (*storagev1.VolumeAttachmentList, error)

	// DeleteVolumeAttachment deletes a volume attachment by name.
	DeleteVolumeAttachment(ctx context.Context, va string) error

	// GetPersistentVolumeClaimsInNamespace returns all the pvcs in a namespace.
	GetPersistentVolumeClaimsInNamespace(ctx context.Context, namespace string) (*v1.PersistentVolumeClaimList, error)

	// GetPersistentVolumeClaimsInPod returns all the pvcs in a pod.
	GetPersistentVolumeClaimsInPod(ctx context.Context, pod *v1.Pod) ([]*v1.PersistentVolumeClaim, error)

	// GetPersistentVolumesInPod returns all the pvs referenced from a pod.
	// Any unbound pvcs are not returned.
	GetPersistentVolumesInPod(ctx context.Context, pod *v1.Pod) ([]*v1.PersistentVolume, error)

	// IsVolumeAttachmentToPod returns true if va is attached to the specified pod.
	IsVolumeAttachmentToPod(ctx context.Context, va *storagev1.VolumeAttachment, pod *v1.Pod) (bool, error)

	// GetPersistentVolumeClaimName returns the PVC name referenced from PV named as input argument
	GetPersistentVolumeClaimName(ctx context.Context, pvName string) (string, error)

	// GetPersistentVolume retrieves a persistent volume given the pv name.
	GetPersistentVolume(ctx context.Context, pvName string) (*v1.PersistentVolume, error)

	// GetPersistentVolumeClaim returns the PVC of the given namespace/pvcName.
	GetPersistentVolumeClaim(ctx context.Context, namespace, pvcName string) (*v1.PersistentVolumeClaim, error)

	// GetNode returns the node with the specified nodeName.
	GetNode(ctx context.Context, nodeName string) (*v1.Node, error)

	// GetNode returns the node with the specified nodeName but using a timeout duration rather than a context.
	GetNodeWithTimeout(duration time.Duration, nodeName string) (*v1.Node, error)

	// GetVolumeHandleFromVA returns the volume handle (storage system ID) from the volume attachment.
	GetVolumeHandleFromVA(ctx context.Context, va *storagev1.VolumeAttachment) (string, error)

	// GetPVNameFromVA returns the PVCName from a specified volume attachment.
	GetPVNameFromVA(va *storagev1.VolumeAttachment) (string, error)

	// SetupPodWatch setups up a pod watch.
	SetupPodWatch(ctx context.Context, namespace string, listOptions metav1.ListOptions) (watch.Interface, error)

	// SetupNodeWatch setups up a node watch.
	SetupNodeWatch(ctx context.Context, listOptions metav1.ListOptions) (watch.Interface, error)
}
