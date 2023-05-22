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

package k8sapi

import (
	"context"
	"errors"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

// K8sMock is a mock structure used for testing
type K8sMock struct {
	KeyToPod               map[string]*v1.Pod
	KeyToPVC               map[string]*v1.PersistentVolumeClaim
	NameToPV               map[string]*v1.PersistentVolume
	NameToVolumeAttachment map[string]*storagev1.VolumeAttachment
	NameToNode             map[string]*v1.Node
	WantFailCount          int
	FailCount              int
	InducedErrors          struct {
		Connect                              bool
		DeletePod                            bool
		GetPod                               bool
		GetVolumeAttachments                 bool
		DeleteVolumeAttachment               bool
		GetPersistentVolumeClaimsInNamespace bool
		GetPersistentVolumeClaimsInPod       bool
		GetPersistentVolumesInPod            bool
		IsVolumeAttachmentToPod              bool
		GetPersistentVolumeClaimName         bool
		GetPersistentVolume                  bool
		GetPersistentVolumeClaim             bool
		GetNode                              bool
		GetNodeWithTimeout                   bool
		GetNodeNoAnnotation                  bool
		GetNodeBadCSINode                    bool
		GetVolumeHandleFromVA                bool
		GetPVNameFromVA                      bool
		Watch                                bool
		TaintNode                            bool
		CreateEvent                          bool
	}
	Watcher *watch.RaceFreeFakeWatcher
}

// Initialize initial the mock structure
func (mock *K8sMock) Initialize() {
	mock.Watcher = watch.NewRaceFreeFake()
}

// AddPod creates unique functions for managing mocked database.
func (mock *K8sMock) AddPod(pod *v1.Pod) {
	key := mock.getKey(pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
	if mock.KeyToPod == nil {
		mock.KeyToPod = make(map[string]*v1.Pod)
	}
	mock.KeyToPod[key] = pod
}

// AddPVC adds mock PVCs for testing
func (mock *K8sMock) AddPVC(pvc *v1.PersistentVolumeClaim) {
	if mock.KeyToPVC == nil {
		mock.KeyToPVC = make(map[string]*v1.PersistentVolumeClaim)
	}
	key := mock.getKey(pvc.ObjectMeta.Namespace, pvc.ObjectMeta.Name)
	mock.KeyToPVC[key] = pvc
}

// AddPV adds mock PVs for testing
func (mock *K8sMock) AddPV(pv *v1.PersistentVolume) {
	if mock.NameToPV == nil {
		mock.NameToPV = make(map[string]*v1.PersistentVolume)
	}
	mock.NameToPV[pv.ObjectMeta.Name] = pv
}

// AddVA adds mock VAs for testing
func (mock *K8sMock) AddVA(va *storagev1.VolumeAttachment) {
	if mock.NameToVolumeAttachment == nil {
		mock.NameToVolumeAttachment = make(map[string]*storagev1.VolumeAttachment)
	}
	mock.NameToVolumeAttachment[va.ObjectMeta.Name] = va
}

// AddNode adds mock Nodes for testing
func (mock *K8sMock) AddNode(node *v1.Node) {
	if mock.NameToNode == nil {
		mock.NameToNode = make(map[string]*v1.Node)
	}
	mock.NameToNode[node.ObjectMeta.Name] = node
}

// Connect connects to the Kubernetes system API
func (mock *K8sMock) Connect(kubeconfig *string) error {
	if mock.InducedErrors.Connect {
		return errors.New("induced Connect error")
	}
	return nil
}

// GetClient returns the kubernetes Clientset.
func (mock *K8sMock) GetClient() *kubernetes.Clientset {
	var clientset *kubernetes.Clientset
	return clientset
}

// GetContext returns a context object for a certain timeout duration
func (mock *K8sMock) GetContext(duration time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), duration)
}

// DeletePod deletes a pod of the given namespace and name, an optionally uses force deletion.
func (mock *K8sMock) DeletePod(ctx context.Context, namespace, name string, podUID types.UID, force bool) error {
	if mock.InducedErrors.DeletePod {
		return errors.New("induced DeletePod error")
	}
	key := mock.getKey(namespace, name)
	delete(mock.KeyToPod, key)
	return nil
}

// GetPod retrieves a pod of the give namespace and name
func (mock *K8sMock) GetPod(ctx context.Context, namespace, name string) (*v1.Pod, error) {
	var pod *v1.Pod
	if mock.InducedErrors.GetPod {
		return pod, errors.New("induced GetPod error")
	}
	key := mock.getKey(namespace, name)
	var ok bool
	pod, ok = mock.KeyToPod[key]
	if !ok {
		return pod, fmt.Errorf("could not find pod %s", key)
	}
	return pod, nil
}

// GetCachedVolumeAttachment will try to load the volumeattachment select by the persistent volume name and node name.
// If found it is returned from the cache. If not found, the cache is reloaded and the result returned from the reloaded data.
func (mock *K8sMock) GetCachedVolumeAttachment(ctx context.Context, pvName, nodeName string) (*storagev1.VolumeAttachment, error) {
	valist := &storagev1.VolumeAttachmentList{}
	if mock.InducedErrors.GetVolumeAttachments {
		return nil, errors.New("induced GetVolumeAttachments error")
	}
	valist.Items = make([]storagev1.VolumeAttachment, 0)
	for _, item := range mock.NameToVolumeAttachment {
		if *item.Spec.Source.PersistentVolumeName == pvName && item.Spec.NodeName == nodeName {
			return item, nil
		}
	}
	return nil, nil
}

// GetVolumeAttachments gets all the volume attachments in the K8S system
func (mock *K8sMock) GetVolumeAttachments(ctx context.Context) (*storagev1.VolumeAttachmentList, error) {
	valist := &storagev1.VolumeAttachmentList{}
	if mock.InducedErrors.GetVolumeAttachments {
		return valist, errors.New("induced GetVolumeAttachments error")
	}
	valist.Items = make([]storagev1.VolumeAttachment, 0)
	for _, item := range mock.NameToVolumeAttachment {
		valist.Items = append(valist.Items, *item)
	}
	return valist, nil
}

// DeleteVolumeAttachment deletes a volume attachment by name.
func (mock *K8sMock) DeleteVolumeAttachment(ctx context.Context, va string) error {
	if mock.InducedErrors.DeleteVolumeAttachment {
		return errors.New("induced DeleteVolumeAttachment error")
	}
	delete(mock.NameToVolumeAttachment, va)
	return nil
}

// GetPersistentVolumeClaimsInNamespace returns all the pvcs in a namespace.
func (mock *K8sMock) GetPersistentVolumeClaimsInNamespace(ctx context.Context, namespace string) (*v1.PersistentVolumeClaimList, error) {
	pvclist := &v1.PersistentVolumeClaimList{}
	if mock.InducedErrors.GetPersistentVolumeClaimsInNamespace {
		return pvclist, errors.New("induced GetPersistentVolumeClaimsInNamespace error")
	}
	pvclist.Items = make([]v1.PersistentVolumeClaim, 0)
	for _, item := range mock.KeyToPVC {
		pvclist.Items = append(pvclist.Items, *item)
	}
	return pvclist, nil
}

// GetPersistentVolumeClaimsInPod returns all the pvcs in a pod.
func (mock *K8sMock) GetPersistentVolumeClaimsInPod(ctx context.Context, pod *v1.Pod) ([]*v1.PersistentVolumeClaim, error) {
	pvclist := make([]*v1.PersistentVolumeClaim, 0)
	if mock.InducedErrors.GetPersistentVolumeClaimsInPod {
		return pvclist, errors.New("induced GetPersistentVolumeClaimsInPod error")
	}
	for _, vol := range pod.Spec.Volumes {
		if vol.VolumeSource.PersistentVolumeClaim != nil {
			pvc, err := mock.GetPersistentVolumeClaim(ctx, pod.ObjectMeta.Namespace, vol.VolumeSource.PersistentVolumeClaim.ClaimName)
			if err != nil {
				return pvclist, fmt.Errorf("Could not retrieve PVC: %s/%s", pod.ObjectMeta.Namespace, vol.VolumeSource.PersistentVolumeClaim.ClaimName)
			}
			pvclist = append(pvclist, pvc)
		}
	}
	return pvclist, nil
}

// GetPersistentVolumesInPod returns all the pvs referenced from a pod.
// Any unbound pvcs are not returned.
func (mock *K8sMock) GetPersistentVolumesInPod(ctx context.Context, pod *v1.Pod) ([]*v1.PersistentVolume, error) {
	pvs := make([]*v1.PersistentVolume, 0)
	if mock.InducedErrors.GetPersistentVolumesInPod {
		return pvs, errors.New("induced GetPersistentVolumesInPod error")
	}
	pvcs, err := mock.GetPersistentVolumeClaimsInPod(ctx, pod)
	if err != nil {
		return pvs, err
	}
	// Fetch the pv for each pvc
	for _, pvc := range pvcs {
		if pvc.Status.Phase != "Bound" || pvc.Spec.VolumeName == "" {
			log.Infof("pvc %s/%s not bound", pvc.ObjectMeta.Namespace, pvc.ObjectMeta.Name)
			continue
		}
		pv, err := mock.GetPersistentVolume(ctx, pvc.Spec.VolumeName)
		if err != nil {
			return pvs, err
		}
		pvs = append(pvs, pv)
	}
	return pvs, nil
}

// IsVolumeAttachmentToPod returns true if va is attached to the specified pod.
func (mock *K8sMock) IsVolumeAttachmentToPod(ctx context.Context, va *storagev1.VolumeAttachment, pod *v1.Pod) (bool, error) {
	if mock.InducedErrors.IsVolumeAttachmentToPod {
		return false, errors.New("induced IsVolumeAttachmentToPod error")
	}
	if pod.Spec.NodeName != va.Spec.NodeName || va.Spec.Source.PersistentVolumeName == nil {
		return false, nil
	}
	for _, vol := range pod.Spec.Volumes {
		if vol.VolumeSource.PersistentVolumeClaim != nil {
			log.Debugf("namespace %s claimname %s", pod.ObjectMeta.Namespace, vol.VolumeSource.PersistentVolumeClaim.ClaimName)
			pvc, err := mock.GetPersistentVolumeClaim(ctx, pod.ObjectMeta.Namespace, vol.VolumeSource.PersistentVolumeClaim.ClaimName)
			if err != nil || pvc == nil {
				return false, fmt.Errorf("Could not retrieve PVC: %s/%s", pod.ObjectMeta.Namespace, vol.VolumeSource.PersistentVolumeClaim.ClaimName)
			}
			log.Debugf("va.pv %s pvc.pv %s", *va.Spec.Source.PersistentVolumeName, pvc.Spec.VolumeName)
			if pvc != nil && va.Spec.Source.PersistentVolumeName != nil && *va.Spec.Source.PersistentVolumeName == pvc.Spec.VolumeName {
				return true, nil
			}
		}
	}
	return false, nil
}

// GetPersistentVolumeClaimName returns the PVC name referenced from PV named as input argument
func (mock *K8sMock) GetPersistentVolumeClaimName(ctx context.Context, pvName string) (string, error) {
	var pvcname string
	if mock.InducedErrors.GetPersistentVolumeClaimName {
		return pvcname, errors.New("induced GetPersistentVolumeClaimName error")
	}
	pv, err := mock.GetPersistentVolume(ctx, pvName)
	if err != nil {
		return "", err
	}
	if pv.Spec.ClaimRef != nil {
		log.Printf("ClaimRef %#v", pv.Spec.ClaimRef)
		if pv.Spec.ClaimRef.Kind == "PersistentVolumeClaim" {
			pvcname = pv.Spec.ClaimRef.Namespace + "/" + pv.Spec.ClaimRef.Name
		}
	}
	log.Printf("pvcname %s", pvcname)

	return pvcname, nil
}

// GetPersistentVolume retrieves a persistent volume given the pv name.
func (mock *K8sMock) GetPersistentVolume(ctx context.Context, pvName string) (*v1.PersistentVolume, error) {
	var pv *v1.PersistentVolume
	if mock.InducedErrors.GetPersistentVolume {
		return pv, errors.New("induced GetPersistentVolume error")
	}
	pv = mock.NameToPV[pvName]
	return pv, nil
}

// GetPersistentVolumeClaim returns the PVC of the given namespace/pvcName.
func (mock *K8sMock) GetPersistentVolumeClaim(ctx context.Context, namespace, pvcName string) (*v1.PersistentVolumeClaim, error) {
	var pvc *v1.PersistentVolumeClaim
	if mock.InducedErrors.GetPersistentVolumeClaim {
		return pvc, errors.New("induced GetPersistentVolumeClaim error")
	}
	key := mock.getKey(namespace, pvcName)
	pvc = mock.KeyToPVC[key]
	return pvc, nil
}

// GetNode returns the node with the specified nodeName.
func (mock *K8sMock) GetNode(ctx context.Context, nodeName string) (*v1.Node, error) {
	var node *v1.Node
	if mock.InducedErrors.GetNode {
		return node, errors.New("induced GetNode error")
	}
	if mock.NameToNode[nodeName] != nil {
		if mock.InducedErrors.GetNodeNoAnnotation { // no node annotation at all
			mock.NameToNode[nodeName].ObjectMeta.Annotations["csi.volume.kubernetes.io/nodeid"] = ""
		}
		if mock.InducedErrors.GetNodeBadCSINode { // bad json
			mock.NameToNode[nodeName].ObjectMeta.Annotations["csi.volume.kubernetes.io/nodeid"] = "[["
		}
		return mock.NameToNode[nodeName], nil
	}
	node = &v1.Node{}
	node.ObjectMeta.Name = nodeName
	node.ObjectMeta.Annotations = make(map[string]string)
	node.ObjectMeta.Annotations["csi.volume.kubernetes.io/nodeid"] = "{\"csi-vxflexos.dellemc.com\":\"46C8B5F8-74A3-4B2B-B158-EF845654D38C\"}"
	node.ObjectMeta.UID = "46C8B5F8-74A3-4B2B-B158-EF845654D38C"
	return node, nil
}

// GetNodeWithTimeout returns the node with the specified nodeName but using a timeout duration rather than a context.
func (mock *K8sMock) GetNodeWithTimeout(duration time.Duration, nodeName string) (*v1.Node, error) {
	if mock.InducedErrors.GetNodeWithTimeout {
		mock.FailCount++
		if mock.FailCount <= mock.WantFailCount || mock.WantFailCount <= 0 {
			return nil, errors.New("induced GetNodeWithTimeout error")
		}
		// Reset, so that calls will now succeed
		mock.InducedErrors.GetNodeWithTimeout = false
	}
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()
	return mock.GetNode(ctx, nodeName)
}

// GetVolumeHandleFromVA returns the volume handle (storage system ID) from the volume attachment.
func (mock *K8sMock) GetVolumeHandleFromVA(ctx context.Context, va *storagev1.VolumeAttachment) (string, error) {
	if mock.InducedErrors.GetVolumeHandleFromVA {
		return "", errors.New("induced GetVolumeHandleFromVA error")
	}
	pvname, err := mock.GetPVNameFromVA(va)
	if err != nil {
		return "", err
	}
	pv, err := mock.GetPersistentVolume(ctx, pvname)
	if err != nil {
		return "", err
	}
	pvsrc := pv.Spec.PersistentVolumeSource
	if pvsrc.CSI != nil {
		return pvsrc.CSI.VolumeHandle, nil
	}
	return "", fmt.Errorf("PV is not a CSI volume")
}

// GetPVNameFromVA returns the PVCName from a specified volume attachment.
func (mock *K8sMock) GetPVNameFromVA(va *storagev1.VolumeAttachment) (string, error) {
	if mock.InducedErrors.GetPVNameFromVA {
		return "", errors.New("induced GetPVNameFromVA error")
	}
	if va.Spec.Source.PersistentVolumeName != nil {
		return *va.Spec.Source.PersistentVolumeName, nil
	}
	return "", fmt.Errorf("Could not find PersistentVolume from VolumeAttachment %s", va.ObjectMeta.Name)
}

func (mock *K8sMock) getKey(namespace, name string) string {
	return namespace + "/" + name
}

// SetupPodWatch returns a mock watcher
func (mock *K8sMock) SetupPodWatch(ctx context.Context, namespace string, listOptions metav1.ListOptions) (watch.Interface, error) {
	if mock.InducedErrors.Watch {
		return nil, errors.New("included Watch error")
	}
	return mock.Watcher, nil
}

// SetupNodeWatch returns a mock watcher
func (mock *K8sMock) SetupNodeWatch(ctx context.Context, listOptions metav1.ListOptions) (watch.Interface, error) {
	if mock.InducedErrors.Watch {
		return nil, errors.New("included Watch error")
	}
	return mock.Watcher, nil
}

// TaintNode mocks tainting a node
func (mock *K8sMock) TaintNode(ctx context.Context, nodeName, taintKey string, effect v1.TaintEffect, remove bool) error {
	if mock.InducedErrors.TaintNode {
		return errors.New("induced taint node error")
	}
	if node, err := mock.GetNode(ctx, nodeName); err == nil {
		currentTaints := node.Spec.Taints
		updatedTaints := make([]v1.Taint, 0)
		for _, taint := range currentTaints {
			if taint.Key != taintKey {
				updatedTaints = append(updatedTaints, taint)
			}
		}
		if !remove {
			updatedTaints = append(updatedTaints, v1.Taint{
				Key:    taintKey,
				Value:  "",
				Effect: effect,
			})
		}
		node.Spec.Taints = updatedTaints
		mock.AddNode(node)
	} else {
		return err
	}
	return nil
}

// CreateEvent creates an event for the specified object.
func (mock *K8sMock) CreateEvent(sourceComponent string, object runtime.Object, eventType, reason, messageFmt string, args ...interface{}) error {
	if mock.InducedErrors.CreateEvent {
		return errors.New("induced CreateEvent error")
	}
	return nil
}
