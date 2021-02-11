package k8sapi

// k8sapi package provides facilities for csi-drivers to call the kubernetes API from their containers.
// This is needed for some special use cases, like inspecting PVs or handling fail-over of pods from node failure.

import (
	"context"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sync"
	"time"
)

type K8sApiStruct struct {
	Client *kubernetes.Clientset
	Lock   sync.Mutex
}

var K8sClient K8sApiStruct

func (api *K8sApiStruct) GetClient() *kubernetes.Clientset {
	return api.Client
}

func (api *K8sApiStruct) GetContext(duration time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), duration)
}

func (api *K8sApiStruct) DeletePod(ctx context.Context, namespace, name string, force bool) error {
	deleteOptions := metav1.DeleteOptions{}
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

func (api *K8sApiStruct) GetPod(ctx context.Context, namespace, name string) (*v1.Pod, error) {
	getopt := metav1.GetOptions{}
	pod, err := api.Client.CoreV1().Pods(namespace).Get(context.Background(), name, getopt)
	if err != nil {
		log.Errorf("Unable to get pod %s/%s: %s", namespace, name, err)
	}
	return pod, err
}

// GetVolumeAttachments retrieves all the volume attachments
func (api *K8sApiStruct) GetVolumeAttachments(ctx context.Context) (*storagev1.VolumeAttachmentList, error) {
	volumeAttachments, err := api.Client.StorageV1().VolumeAttachments().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return volumeAttachments, nil
}

// DeleteVolumeAttachment deletes a volume attachment by name.
func (api *K8sApiStruct) DeleteVolumeAttachment(ctx context.Context, va string) error {
	deleteOptions := metav1.DeleteOptions{}
	log.Infof("Deleting volume attachment: %s", va)
	err := api.Client.StorageV1().VolumeAttachments().Delete(ctx, va, deleteOptions)
	if err != nil {
		log.Errorf("Couldn't delete VolumeAttachment %s: %s", va, err)
	}
	return err
}

// GetPersistentVolumeClaimsInNamespace returns all the pvcs in a namespace.
func (api *K8sApiStruct) GetPersistentVolumeClaimsInNamespace(ctx context.Context, namespace string) (*v1.PersistentVolumeClaimList, error) {
	persistentVolumes, err := api.Client.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return persistentVolumes, nil
}

// GetPersistentVolumeClaimsInPod returns all the pvcs in a pod.
func (api *K8sApiStruct) GetPersistentVolumeClaimsInPod(ctx context.Context, pod *v1.Pod) ([]*v1.PersistentVolumeClaim, error) {
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
func (api *K8sApiStruct) GetPersistentVolumesInPod(ctx context.Context, pod *v1.Pod) ([]*v1.PersistentVolume, error) {
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
func (api *K8sApiStruct) IsVolumeAttachmentToPod(ctx context.Context, va *storagev1.VolumeAttachment, pod *v1.Pod) (bool, error) {
	if pod.Spec.NodeName != va.Spec.NodeName || va.Spec.Source.PersistentVolumeName == nil {
		return false, nil
	}

	for _, vol := range pod.Spec.Volumes {
		if vol.VolumeSource.PersistentVolumeClaim != nil {
			pvc, err := api.GetPersistentVolumeClaim(ctx, pod.ObjectMeta.Namespace, vol.VolumeSource.PersistentVolumeClaim.ClaimName)
			if err != nil {
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
func (api *K8sApiStruct) GetPersistentVolumeClaimName(ctx context.Context, pvName string) (string, error) {
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
func (api *K8sApiStruct) GetPersistentVolume(ctx context.Context, pvName string) (*v1.PersistentVolume, error) {
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

func (api *K8sApiStruct) GetPersistentVolumeClaim(ctx context.Context, namespace, pvcName string) (*v1.PersistentVolumeClaim, error) {
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

func (api *K8sApiStruct) GetNode(ctx context.Context, nodeName string) (*v1.Node, error) {
	node, err := api.Client.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		log.Error("error retrieving node: " + nodeName + " : " + err.Error())
	}
	return node, err
}

func (api *K8sApiStruct) GetNodeWithTimeout(duration time.Duration, nodeName string) (*v1.Node, error) {
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()
	return api.GetNode(ctx, nodeName)
}

// connect establishes a connection with the k8s API server.
func (api *K8sApiStruct) Connect(kubeconfig *string) error {
	var err error
	var client *kubernetes.Clientset
	log.Info("attempting k8sapi connection")
	if kubeconfig != nil && *kubeconfig != "" {
		log.Infof("Using kubeconfig %s", *kubeconfig)
		config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			return err
		}
		client, err = kubernetes.NewForConfig(config)
	} else {
		log.Infof("Using InClusterConfig()")
		config, err := rest.InClusterConfig()
		if err != nil {
			return err
		}
		client, err = kubernetes.NewForConfig(config)
	}
	if err != nil {
		log.Error("unable to connect to k8sapi: " + err.Error())
		return err
	}
	api.Client = client
	log.Info("connected to k8sapi")
	return nil
}

func (api *K8sApiStruct) GetVolumeHandleFromVA(ctx context.Context, va *storagev1.VolumeAttachment) (string, error) {
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

func (api *K8sApiStruct) GetPVNameFromVA(va *storagev1.VolumeAttachment) (string, error) {
	if va.Spec.Source.PersistentVolumeName != nil {
		return *va.Spec.Source.PersistentVolumeName, nil
	}
	return "", fmt.Errorf("Could not find PersistentVolume from VolumeAttachment %s", va.ObjectMeta.Name)
}

func (api *K8sApiStruct) SetupPodWatch(ctx context.Context, namespace string, listOptions metav1.ListOptions) (watch.Interface, error) {
	watcher, err := api.Client.CoreV1().Pods(namespace).Watch(ctx, listOptions)
	return watcher, err
}

func (api *K8sApiStruct) SetupNodeWatch(ctx context.Context, listOptions metav1.ListOptions) (watch.Interface, error) {
	watcher, err := api.Client.CoreV1().Nodes().Watch(ctx, listOptions)
	return watcher, err
}
