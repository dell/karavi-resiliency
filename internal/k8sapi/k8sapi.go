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

// k8sapi package provides facilities for csi-drivers to call the kubernetes API from their containers.
// This is needed for some special use cases, like inspecting PVs or handling fail-over of pods from node failure.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	repv1 "github.com/dell/csm-replication/api/v1"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apiExtensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	apiTypes "k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Client holds a reference to a Kubernetes client
type Client struct {
	Client                    *kubernetes.Clientset
	ReplicationClient         client.Client
	Config                    *rest.Config
	Lock                      sync.Mutex
	eventRecorder             record.EventRecorder
	volumeAttachmentCache     map[string]*storagev1.VolumeAttachment
	volumeAttachmentNameToKey map[string]string
}

const (
	taintNoUpdateNeeded = "TaintNoUpdateNeeded"
	taintAlreadyExists  = "TaintAlreadyExists"
	taintDoesNotExist   = "TaintDoesNotExist"
	taintAdd            = "TaintAdd"
	taintRemove         = "TaintRemove"
	taintedWithPodmon   = "podmon"
)

// K8sClient references the k8sapi.Client
var K8sClient Client

// GetClient returns instance of Kubernetes.Clientset
func (api *Client) GetClient() *kubernetes.Clientset {
	return api.Client
}

// GetContext returns clientContext and cancel function based on the duration
func (api *Client) GetContext(duration time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), duration)
}

// DeletePod deletes a Pod referenced by a namespace and name
func (api *Client) DeletePod(ctx context.Context, namespace, name string, podUID types.UID, force bool) error {
	deleteOptions := metav1.DeleteOptions{}
	deleteOptions.Preconditions = &metav1.Preconditions{}
	deleteOptions.Preconditions.UID = &podUID
	if force {
		gracePeriodSec := int64(0)
		deleteOptions.GracePeriodSeconds = &gracePeriodSec
	}
	log.Infof("Deleting pod %s/%s force %t", namespace, name, force)
	err := api.Client.CoreV1().Pods(namespace).Delete(ctx, name, deleteOptions)
	if err != nil {
		log.Errorf("Unable to delete pod %s/%s: %s", namespace, name, err)
	}
	return err
}

// GetPod returns a Pod object referenced by the namespace and name
func (api *Client) GetPod(ctx context.Context, namespace, name string) (*v1.Pod, error) {
	getopt := metav1.GetOptions{}
	pod, err := api.Client.CoreV1().Pods(namespace).Get(ctx, name, getopt)
	if err != nil {
		log.Errorf("Unable to get pod %s/%s: %s", namespace, name, err)
	}
	return pod, err
}

var vacachehit, vacachemiss int

// GetCachedVolumeAttachment will try to load the volumeattachment select by the persistent volume name and node name.
// If found it is returned from the cache. If not found, the cache is reloaded and the result returned from the reloaded data.
func (api *Client) GetCachedVolumeAttachment(ctx context.Context, pvName, nodeName string) (*storagev1.VolumeAttachment, error) {
	api.Lock.Lock()
	defer api.Lock.Unlock()
	key := fmt.Sprintf("%s/%s", pvName, nodeName)
	log.Debugf("Looking for volume attachment %s", key)
	if api.volumeAttachmentCache != nil && api.volumeAttachmentCache[key] != nil {
		// Cache hit - return cached VA.
		vacachehit++
		log.Debugf("VA Cache Hit %d / Miss %d", vacachehit, vacachemiss)
		return api.volumeAttachmentCache[key], nil
	}
	// Cache miss. Read all the volume attachments.
	volumeAttachmentList, err := api.GetVolumeAttachments(ctx)
	if err != nil {
		return nil, err
	}
	// Rebuild the cache
	vacachemiss++
	log.Debugf("VA Cache Miss %d / %d", vacachemiss, vacachehit)
	api.volumeAttachmentCache = make(map[string]*storagev1.VolumeAttachment)
	api.volumeAttachmentNameToKey = make(map[string]string)
	log.Infof("Rebuilding VA cache, hits %d misses %d", vacachehit, vacachemiss)
	for _, va := range volumeAttachmentList.Items {
		vaCopy := va.DeepCopy() // To prevent gosec error: "G601 (CWE-118): Implicit memory aliasing in for loop"
		if va.Spec.Source.PersistentVolumeName != nil {
			vaKey := fmt.Sprintf("%s/%s", *va.Spec.Source.PersistentVolumeName, va.Spec.NodeName)
			api.volumeAttachmentCache[vaKey] = vaCopy
			api.volumeAttachmentNameToKey[vaCopy.ObjectMeta.Name] = vaKey
			log.Debugf("Adding VA Cache %s %s", vaCopy.ObjectMeta.Name, vaKey)
		}
	}
	return api.volumeAttachmentCache[key], nil
}

// GetVolumeAttachments retrieves all the volume attachments
func (api *Client) GetVolumeAttachments(ctx context.Context) (*storagev1.VolumeAttachmentList, error) {
	volumeAttachments, err := api.Client.StorageV1().VolumeAttachments().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return volumeAttachments, nil
}

// DeleteVolumeAttachment deletes a volume attachment by name.
func (api *Client) DeleteVolumeAttachment(ctx context.Context, vaname string) error {
	deleteOptions := metav1.DeleteOptions{}
	log.Infof("Deleting volume attachment: %s", vaname)
	err := api.Client.StorageV1().VolumeAttachments().Delete(ctx, vaname, deleteOptions)
	if err != nil {
		log.Errorf("Couldn't delete VolumeAttachment %s: %s", vaname, err)
	}
	api.Lock.Lock()
	defer api.Lock.Unlock()
	if api.volumeAttachmentNameToKey != nil {
		// Look for and delete the name to delete.
		vaKey := api.volumeAttachmentNameToKey[vaname]
		if vaKey != "" {
			log.Infof("Deleting VolumeAttachment from VA Cache %s %s", vaname, vaKey)
			delete(api.volumeAttachmentCache, vaKey)
			delete(api.volumeAttachmentNameToKey, vaname)
		}
	}
	return err
}

// GetPersistentVolumeClaimsInNamespace returns all the pvcs in a namespace.
func (api *Client) GetPersistentVolumeClaimsInNamespace(ctx context.Context, namespace string) (*v1.PersistentVolumeClaimList, error) {
	persistentVolumes, err := api.Client.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return persistentVolumes, nil
}

// GetPersistentVolumeClaimsInPod returns all the pvcs in a pod.
func (api *Client) GetPersistentVolumeClaimsInPod(ctx context.Context, pod *v1.Pod) ([]*v1.PersistentVolumeClaim, error) {
	pvcs := make([]*v1.PersistentVolumeClaim, 0)
	for _, vol := range pod.Spec.Volumes {
		if vol.VolumeSource.PersistentVolumeClaim != nil {
			pvc, err := api.GetPersistentVolumeClaim(ctx, pod.ObjectMeta.Namespace, vol.VolumeSource.PersistentVolumeClaim.ClaimName)
			if err != nil {
				return pvcs, fmt.Errorf("Could not retrieve PVC: %s/%s", pod.ObjectMeta.Namespace, vol.VolumeSource.PersistentVolumeClaim.ClaimName)
			}
			pvcs = append(pvcs, pvc)
		}
	}
	return pvcs, nil
}

// GetPersistentVolumesInPod returns all the pvs referenced from a pod.
// Any unbound pvcs are not returned.
func (api *Client) GetPersistentVolumesInPod(ctx context.Context, pod *v1.Pod) ([]*v1.PersistentVolume, error) {
	pvs := make([]*v1.PersistentVolume, 0)
	pvcs, err := api.GetPersistentVolumeClaimsInPod(ctx, pod)
	if err != nil {
		return pvs, err
	}
	// Fetch the pv for each pvc
	for _, pvc := range pvcs {
		if pvc.Status.Phase != "Bound" || pvc.Spec.VolumeName == "" {
			log.Infof("pvc %s/%s not bound", pvc.ObjectMeta.Namespace, pvc.ObjectMeta.Name)
			continue
		}
		pv, err := api.GetPersistentVolume(ctx, pvc.Spec.VolumeName)
		if err != nil {
			return pvs, err
		}
		pvs = append(pvs, pv)
	}
	return pvs, nil
}

// IsVolumeAttachmentToPod returns true if va is attached to the specified pod.
func (api *Client) IsVolumeAttachmentToPod(ctx context.Context, va *storagev1.VolumeAttachment, pod *v1.Pod) (bool, error) {
	if pod.Spec.NodeName != va.Spec.NodeName || *va.Spec.Source.PersistentVolumeName == "" {
		return false, nil
	}

	for _, vol := range pod.Spec.Volumes {
		if vol.VolumeSource.PersistentVolumeClaim != nil {
			pvc, err := api.GetPersistentVolumeClaim(ctx, pod.ObjectMeta.Namespace, vol.VolumeSource.PersistentVolumeClaim.ClaimName)
			if err != nil {
				return false, fmt.Errorf("Could not retrieve PVC: %s/%s", pod.ObjectMeta.Namespace, vol.VolumeSource.PersistentVolumeClaim.ClaimName)
			}
			volumeName := "nil"
			if pvc != nil {
				volumeName = pvc.Spec.VolumeName
			}
			log.Debugf("va.pv %s pvc.pv %s", *va.Spec.Source.PersistentVolumeName, volumeName)
			if pvc != nil && va.Spec.Source.PersistentVolumeName != nil && *va.Spec.Source.PersistentVolumeName == pvc.Spec.VolumeName {
				return true, nil
			}
		}
	}
	return false, nil
}

// GetPersistentVolumeClaimName returns the PVC name referenced from PV named as input argument
func (api *Client) GetPersistentVolumeClaimName(ctx context.Context, pvName string) (string, error) {
	pvcname := ""
	pv, err := api.GetPersistentVolume(ctx, pvName)
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

// GetPersistentVolume retrieves a persistent volume given the pv name. It returns a PersistentVolume struct.
func (api *Client) GetPersistentVolume(ctx context.Context, pvName string) (*v1.PersistentVolume, error) {
	var err error
	if api.Client == nil {
		return nil, errors.New("No connection")
	}
	getopt := metav1.GetOptions{}
	pv, err := api.Client.CoreV1().PersistentVolumes().Get(ctx, pvName, getopt)
	if err != nil {
		log.Error("error retrieving PersistentVolume: " + pvName + " : " + err.Error())
	}
	return pv, err
}

// GetPersistentVolumesWithLabels returns all the PVs matching one or more labels passed in the labelSelector.
// The format of the labelSelector a string of form "key1=value1,key2=value2"
func (api *Client) GetPersistentVolumesWithLabels(ctx context.Context, labelSelector string) (*v1.PersistentVolumeList, error) {
	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
	}
	pvs, err := api.Client.CoreV1().PersistentVolumes().List(ctx, listOptions)
	return pvs, err
}

func (api *Client) GetPersistentVolumeClaimsWithLabels(ctx context.Context, labelSelector string) (*v1.PersistentVolumeClaimList, error) {
	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
	}
	pvcs, err := api.Client.CoreV1().PersistentVolumeClaims(v1.NamespaceAll).List(ctx, listOptions)
	return pvcs, err
}

// GetPersistentVolumeClaim returns a PVC object given its namespace and name
func (api *Client) GetPersistentVolumeClaim(ctx context.Context, namespace, pvcName string) (*v1.PersistentVolumeClaim, error) {
	var err error
	if api.Client == nil {
		return nil, errors.New("No connection")
	}
	pvcinterface := api.Client.CoreV1().PersistentVolumeClaims(namespace)
	getopt := metav1.GetOptions{}
	pvc, err := pvcinterface.Get(ctx, pvcName, getopt)
	if err != nil {
		log.Errorf("error retrieving PVC: %s : %s", pvcName, err.Error())
	}
	return pvc, err
}

// GetNode returns a Node object given its name
func (api *Client) GetNode(ctx context.Context, nodeName string) (*v1.Node, error) {
	node, err := api.Client.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		log.Error("error retrieving node: " + nodeName + " : " + err.Error())
	}
	return node, err
}

// GetNodeWithTimeout returns a Node object given its name waiting for certain duration before timing out
func (api *Client) GetNodeWithTimeout(duration time.Duration, nodeName string) (*v1.Node, error) {
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()
	return api.GetNode(ctx, nodeName)
}

// Connect connect establishes a connection with the k8s API server.
func (api *Client) Connect(kubeconfig *string) error {
	var err error
	var client *kubernetes.Clientset
	log.Info("attempting k8sapi connection")
	var config *rest.Config
	if kubeconfig != nil && *kubeconfig != "" {
		log.Infof("Using kubeconfig %s", *kubeconfig)
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			return err
		}
		// Change parameters to address k8s throttling issues
		config.QPS = 100
		config.Burst = 100
		client, err = kubernetes.NewForConfig(config)
	} else {
		log.Infof("Using InClusterConfig()")
		config, err = rest.InClusterConfig()
		if err != nil {
			return err
		}
		client, err = kubernetes.NewForConfig(config)
	}
	if err != nil {
		log.Error("unable to connect to k8sapi: " + err.Error())
		return err
	}
	api.Config = config
	api.Client = client

	// Make a replication client
	api.ReplicationClient, err = getReplicationClient(api.Config)
	if err != nil {
		log.Errorf("unable to create ReplicationClient: %s", err.Error())
		return err
	}

	log.Info("connected to k8sapi")
	return nil
}

// Gets a replication client
func getReplicationClient(clientConfig *rest.Config) (client.Client, error) {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(apiExtensionsv1.AddToScheme(scheme))
	utilruntime.Must(repv1.AddToScheme(scheme))
	k8sClient, err := client.New(clientConfig, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}
	return k8sClient, nil
}

// GetVolumeHandleFromVA returns a the CSI.VolumeHandle string for a given VolumeAttachment
func (api *Client) GetVolumeHandleFromVA(ctx context.Context, va *storagev1.VolumeAttachment) (string, error) {
	pvname, err := api.GetPVNameFromVA(va)
	if err != nil {
		return "", err
	}
	pv, err := api.GetPersistentVolume(ctx, pvname)
	if err != nil {
		return "", err
	}
	pvsrc := pv.Spec.PersistentVolumeSource
	if pvsrc.CSI != nil {
		return pvsrc.CSI.VolumeHandle, nil
	}
	return "", fmt.Errorf("PV is not a CSI volume")
}

// GetPVNameFromVA returns the PV name given a VolumeAttachment object reference
func (api *Client) GetPVNameFromVA(va *storagev1.VolumeAttachment) (string, error) {
	if va.Spec.Source.PersistentVolumeName != nil {
		return *va.Spec.Source.PersistentVolumeName, nil
	}
	return "", fmt.Errorf("Could not find PersistentVolume from VolumeAttachment %s", va.ObjectMeta.Name)
}

// SetupPodWatch returns a watch.Interface given the namespace and list options
func (api *Client) SetupPodWatch(ctx context.Context, namespace string, listOptions metav1.ListOptions) (watch.Interface, error) {
	watcher, err := api.Client.CoreV1().Pods(namespace).Watch(ctx, listOptions)
	return watcher, err
}

// SetupNodeWatch returns a watch.Interface given the list options
func (api *Client) SetupNodeWatch(ctx context.Context, listOptions metav1.ListOptions) (watch.Interface, error) {
	watcher, err := api.Client.CoreV1().Nodes().Watch(ctx, listOptions)
	return watcher, err
}

// PatchNodeLabels will patch a node's labels. To add or update a label place it in replacedLabels. To delete a label place
// it in deletedLabels.
func (api *Client) PatchNodeLabels(ctx context.Context, nodeName string, replacedLabels map[string]string, deletedLabels []string) error {
	node, err := api.GetNode(ctx, nodeName)
	if err != nil {
		return err
	}
	oldData, err := json.Marshal(node)
	if err != nil {
		return err
	}

	labels := node.Labels
	// Remove any labels to be deleted
	var updated bool
	for i := 0; i < len(deletedLabels); i++ {
		if labels[deletedLabels[i]] != "" {
			delete(labels, deletedLabels[i])
			updated = true
		}
	}
	// Add/update any new labels
	for k, v := range replacedLabels {
		if labels[k] != v {
			labels[k] = v
			updated = true
		}
	}
	if !updated {
		return nil
	}

	newData, err := json.Marshal(node)
	if err != nil {
		return err
	}

	log.Infof("Patching node labels %s to %v", node.Name, labels)
	// Produce a patch update object
	patchBytes, err := strategicpatch.CreateTwoWayMergePatch(oldData, newData, node)
	if err != nil {
		return err
	}
	patchOptions := metav1.PatchOptions{FieldManager: taintedWithPodmon}

	_, err = api.Client.CoreV1().Nodes().Patch(ctx, node.Name, types.StrategicMergePatchType, patchBytes, patchOptions)

	return err
}

// PatchPodLabels will patch a pod's labels. To add or update a label place it in replacedLabels. To delete a label place
// it in deletedLabels.
func (api *Client) PatchPodLabels(ctx context.Context, podNamespace, podName string, replacedLabels map[string]string, deletedLabels []string) error {
	pod, err := api.GetPod(ctx, podNamespace, podName)
	if err != nil {
		return err
	}
	oldData, err := json.Marshal(pod)
	if err != nil {
		return err
	}

	labels := pod.Labels
	// Remove any labels to be deleted
	var updated bool
	for i := 0; i < len(deletedLabels); i++ {
		if labels[deletedLabels[i]] != "" {
			delete(labels, deletedLabels[i])
			updated = true
		}
	}
	// Add/update any new labels
	for k, v := range replacedLabels {
		if labels[k] != v {
			labels[k] = v
			updated = true
		}
	}
	if !updated {
		return nil
	}

	newData, err := json.Marshal(pod)
	if err != nil {
		return err
	}

	log.Infof("Patching pod %s/%s to %v", pod.Namespace, pod.Name, labels)
	// Produce a patch update object
	patchBytes, err := strategicpatch.CreateTwoWayMergePatch(oldData, newData, pod)
	if err != nil {
		return err
	}
	patchOptions := metav1.PatchOptions{FieldManager: taintedWithPodmon}

	_, err = api.Client.CoreV1().Pods(pod.Namespace).Patch(ctx, pod.Name, types.StrategicMergePatchType, patchBytes, patchOptions)

	return err
}

// TaintNode applies the specified 'taintKey' string and 'effect' to the node with 'nodeName'
// The 'remove' flag indicates if the taint should be removed from the node, if it exists.
func (api *Client) TaintNode(ctx context.Context, nodeName, taintKey string, effect v1.TaintEffect, remove bool) error {
	node, err := api.GetNode(ctx, nodeName)
	if err != nil {
		return err
	}

	// Capture what the node looks like now
	oldData, err := json.Marshal(node)
	if err != nil {
		return err
	}

	// Apply the taint request against the node and determine if it should be patched
	// Note: node.Spec.Taints will have an updated list if 'shouldPatch' == true
	operation, shouldPatch := updateTaint(node, taintKey, effect, remove)
	if shouldPatch {
		log.Infof("Attempting %s : %s against node %s", operation, taintKey, nodeName)

		// Should be patched, so get latest json data for node containing updated taints
		newData, err2 := json.Marshal(node)
		if err2 != nil {
			return err2
		}

		// Produce a patch update object
		patchBytes, err2 := strategicpatch.CreateTwoWayMergePatch(oldData, newData, node)
		if err2 != nil {
			return err2
		}

		// Indicate what's making the taint patch
		patchOptions := metav1.PatchOptions{FieldManager: taintedWithPodmon}

		// Request k8s to patch the node with the new taints applied
		_, err2 = api.Client.CoreV1().Nodes().Patch(ctx, nodeName, types.StrategicMergePatchType, patchBytes, patchOptions)
		return err2
	}

	log.Infof("%s : %s on node %s", operation, taintKey, nodeName)

	return err
}

// updateTaint adds or removes the specified taint key with the effect against the node
// Returns a string indicating the operation or message and a boolean value indicating
// if the taint should be Patched.
func updateTaint(node *v1.Node, taintKey string, effect v1.TaintEffect, remove bool) (string, bool) {
	// Init parameters
	theTaint := v1.Taint{
		Key:    taintKey,
		Value:  "",
		Effect: effect,
	}
	taintOperation := taintNoUpdateNeeded
	shouldPatchNode := false
	updatedTaints := make([]v1.Taint, 0)
	oldTaints := node.Spec.Taints

	if remove {
		// Request to remove the taint. If it doesn't exist, then return now
		if !taintExists(node, taintKey, effect) {
			return taintDoesNotExist, false
		}

		// Copy over taints, skipping the one that we want to remove
		for _, taint := range oldTaints {
			if !taint.MatchTaint(&theTaint) {
				updatedTaints = append(updatedTaints, taint)
			}
		}

		shouldPatchNode = true
		taintOperation = taintRemove
	} else {
		// Request to add taint. If it already exists, then return now
		if taintExists(node, taintKey, effect) {
			return taintAlreadyExists, false
		}

		timeNow := metav1.Now()
		theTaint.TimeAdded = &timeNow

		// Add the new taint to the list of existing ones
		newTaints := append([]v1.Taint{}, theTaint)
		updatedTaints = append(newTaints, oldTaints...)

		shouldPatchNode = true
		taintOperation = taintAdd
	}

	// Update the node object's taints
	if shouldPatchNode {
		node.Spec.Taints = updatedTaints
	}

	return taintOperation, shouldPatchNode
}

// taintExists checks if the node contains the taint 'key' with the specified 'effect'
func taintExists(node *v1.Node, key string, effect v1.TaintEffect) bool {
	for _, taint := range node.Spec.Taints {
		if taint.Key == key && taint.Effect == effect {
			return true
		}
	}
	return false
}

// CreateEvent creates an event on a runtime object.
// eventType is the type of this event (Normal, Warning)
// reason is why the action was taken. It is human-readable.
// messageFmt and args for a human readable description of the status of this operation
func (api *Client) CreateEvent(sourceComponent string, object runtime.Object, eventType, reason, messageFmt string, args ...interface{}) error {
	if api.eventRecorder == nil {
		broadcaster := record.NewBroadcaster()
		broadcaster.StartLogging(log.Infof)
		broadcaster.StartRecordingToSink(&corev1.EventSinkImpl{Interface: api.Client.CoreV1().Events(v1.NamespaceAll)})
		api.eventRecorder = broadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: fmt.Sprintf(sourceComponent)})
	}
	api.eventRecorder.Eventf(object, eventType, reason, messageFmt, args)
	return nil
}

func (api *Client) GetReplicationGroup(ctx context.Context, rgName string) (*repv1.DellCSIReplicationGroup, error) {
	found := &repv1.DellCSIReplicationGroup{}
	err := api.ReplicationClient.Get(ctx, apiTypes.NamespacedName{Name: rgName}, found)
	if err != nil {
		log.Errorf("Error finding replication group %s: %s", rgName, err.Error())
		return nil, err
	}
	if found == nil {
		err = fmt.Errorf("ReplicationGroup %s not found", rgName)
	}
	return found, err
}

func (api *Client) UpdateReplicationGroup(ctx context.Context, rg *repv1.DellCSIReplicationGroup) error {
	err := api.ReplicationClient.Update(ctx, rg)
	if err != nil {
		log.Errorf("Error updating replication group %s: %s", rg.Name, err.Error())
		return err
	}
	return nil
}
