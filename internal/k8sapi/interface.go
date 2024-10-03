/*
* Copyright (c) 2021-2023 Dell Inc., or its subsidiaries. All Rights Reserved.
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

// Package k8sapi provides an interface for access Kubernetes objects.
package k8sapi

// k8sapi package provides facilities for csi-drivers to call the kubernetes API from their containers.
// This is needed for some special use cases, like inspecting PVs or handling fail-over of pods from node failure.

import (
	"context"
	"time"

	repv1 "github.com/dell/csm-replication/api/v1"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

// K8sAPI provides an API interface to kubernetes.
type K8sAPI interface {
	// Connect connects to the Kubernetes system API
	Connect(kubeconfig *string) error

	// GetClient returns the kubernetes Clientset.
	GetClient() *kubernetes.Clientset

	// GetContext returns a context object for a certain timeout duration
	GetContext(duration time.Duration) (context.Context, context.CancelFunc)

	// DeletePod deletes a pod of the given namespace and name, an optionally uses force deletion.
	DeletePod(ctx context.Context, namespace, name string, podUID types.UID, force bool) error

	// GetPod retrieves a pod of the give namespace and name
	GetPod(ctx context.Context, namespace, name string) (*v1.Pod, error)

	// GetCachedVolumeAttachment will try to load the volumeattachment select by the persistent volume name and node name.
	// If found it is returned from the cache. If not found, the cache is reloaded and the result returned from the reloaded data.
	GetCachedVolumeAttachment(ctx context.Context, pvName, nodeName string) (*storagev1.VolumeAttachment, error)

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

	// GetPersistentVolumesWithLabels returns all the PVs matching one or more labels passed in the labelSelector.
	// The format of the labelSelector a string of form "key1=value1,key2=value2"
	GetPersistentVolumesWithLabels(ctx context.Context, labelSelector string) (*v1.PersistentVolumeList, error)

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

	// PatchNodeLabels will patch the node if the labels are altered.
	PatchNodeLabels(ctx context.Context, nodeName string, replacedLabels map[string]string, deletedLabels []string) error

	// PatchPodLabels will patch the pod if the labels are altered.
	PatchPodLabels(ctx context.Context, podNamespace, podName string, replacedLabels map[string]string, deletedLabels []string) error

	// TaintNode applies the specified 'taintKey' string and 'effect' to the node with 'nodeName'
	// The 'remove' flag indicates if the taint should be removed from the node, if it exists.
	TaintNode(ctx context.Context, nodeName, taintKey string, effect v1.TaintEffect, remove bool) error

	// CreateEvent creates an event on a runtime object.
	// sourceComponent is name of component producing event, e.g. "podmon"
	// eventType is the type of this event (Normal, Warning)
	// reason is why the action was taken. It is human-readable.
	// messageFmt and args for a human readable description of the status of this operation
	CreateEvent(sourceComponent string, object runtime.Object, eventType, reason, messageFmt string, args ...interface{}) error

	// GetReplicationGroup will retrieve the named replication group.
	GetReplicationGroup(ctx context.Context, rgName string) (*repv1.DellCSIReplicationGroup, error)

	// UpdatereplicationGroup will update an existing Replication Group.
	UpdateReplicationGroup(ctx context.Context, rg *repv1.DellCSIReplicationGroup) error

	// Patch the number of replicas in a StatefulSet. It returns the previous numbber of replicas and error.
	PatchStatefulSetReplicas(ctx context.Context, namespace, name string, replicas int32) (int32, error)
}

const (
	// EventTypeNormal will log a "Normal" event.
	EventTypeNormal = "Normal"
	// EventTypeWarning will log a "Warning" event.
	EventTypeWarning = "Warning"
)
