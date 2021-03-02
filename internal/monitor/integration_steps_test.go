/*
 * Copyright (c) 2021. Dell Inc., or its subsidiaries. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 */

package monitor

import (
	"context"
	"fmt"
	"github.com/cucumber/godog"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"path/filepath"
	"podmon/internal/k8sapi"
	"strings"
	"time"
)

type integration struct {
	configPath string
	k8s        k8sapi.K8sAPI
}

func (i *integration) givenKubernetes(configPath string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	if configPath == "" {
		i.configPath = filepath.Join(homeDir, ".kube", "config")
	} else {
		i.configPath = configPath
	}

	i.k8s = &k8sapi.K8sClient
	err = i.k8s.Connect(&i.configPath)
	if err != nil {
		message := fmt.Sprintf("kubernetes connection error: %s\n", err)
		fmt.Print(message)
		return fmt.Errorf(message)
	}

	err = i.dumpNodeInfo()
	if err != nil {
		return err
	}
	return nil
}

func (i *integration) allPodsAreRunningWithinSeconds(wait int) error {
	namespaces := ""
	allRunning, err := i.checkIfAllPodsRunning(namespaces)
	if err != nil {
		return err
	}

	if allRunning {
		fmt.Printf("All pods in %s namespaces are in the 'Running' state", namespaces)
		return nil
	} else {
		time.Sleep(time.Duration(wait) * time.Second)
	}

	allRunning, err = i.checkIfAllPodsRunning(namespaces)
	if err != nil {
		return err
	}

	return AssertExpectedAndActual(assert.Equal, true, allRunning,
		fmt.Sprintf("Expected all pods to be in running state after %d seconds", wait))
}

func (i *integration) failWorkerAndMasterNodes(numNodes, numMasters, failure string, wait int) error {
	return godog.ErrPending
}

func (i *integration) podsPerNodeWithVolumesAndDevicesEach(podsPerNode, numVols, numDevs string) error {
	return godog.ErrPending
}

func (i *integration) theTaintsForTheFailedNodesAreRemovedWithinSeconds(wait int) error {
	taintKey := "podmon.dellemc.com"
	havePodmonTaint, err := i.checkIfNodesHaveTaint(taintKey)
	if err != nil {
		return err
	}

	if havePodmonTaint {
		time.Sleep(time.Duration(wait) * time.Second)
	}

	havePodmonTaint, err = i.checkIfNodesHaveTaint(taintKey)
	if err != nil {
		return err
	}

	return AssertExpectedAndActual(assert.Equal, false, havePodmonTaint,
		fmt.Sprintf("Expected %s taint to be removed after %d seconds, but still exist", taintKey, wait))
}

func (i *integration) theseCSIDriverAreConfiguredOnTheSystem(driverName string) error {
	driverObj, err := i.k8s.GetClient().StorageV1().CSIDrivers().Get(context.Background(), driverName, v1.GetOptions{})
	if err != nil {
		return err
	}
	fmt.Printf("Driver %s exists on the cluster", driverObj.Name)
	return AssertExpectedAndActual(assert.Equal, driverName, driverObj.Name,
		fmt.Sprintf("No CSIDriver named %s found in cluster", driverName))
}

func (i *integration) thereIsThisNamespaceInTheCluster(namespace string) error {
	foundNamespace := false
	namespaces, err := i.k8s.GetClient().CoreV1().Namespaces().List(context.Background(), v1.ListOptions{})
	if err != nil {
		return err
	}

	for _, ns := range namespaces.Items {
		if namespace == ns.Name {
			foundNamespace = true
			break
		}
	}

	return AssertExpectedAndActual(assert.Equal, true, foundNamespace,
		fmt.Sprintf("Namespace %s was expected, but does not exist", namespace))
}

func (i *integration) thereAreDriverPodsWithThisPrefix(namespace, prefix string) error {
	pods, err := i.k8s.GetClient().CoreV1().Pods(namespace).List(context.Background(), v1.ListOptions{})
	if err != nil {
		return err
	}

	nWorkerNodes := 0
	nodes, err := i.k8s.GetClient().CoreV1().Nodes().List(context.Background(), v1.ListOptions{})
	if err != nil {
		return err
	}
	for _, node := range nodes.Items {
		// Some k8s clusters may not have a worker label against
		// nodes, so check for the master label. If it doesn't
		// exist against the node, then it's consider a worker.

		// Check if there's a master label associated with the node
		hasMasterLabel := false
		for label := range node.Labels {
			if label == "node-role.kubernetes.io/master" {
				hasMasterLabel = true
				break
			}
		}

		// No master label found implies it's a worker node
		if !hasMasterLabel {
			nWorkerNodes += 1
		}
	}

	// Look for controller and node driver pods running in the cluster
	lookForController := fmt.Sprintf("%s-controller", prefix)
	lookForNode := fmt.Sprintf("%s-node", prefix)
	nRunningControllers := 0
	nRunningNode := 0
	for _, pod := range pods.Items {
		if pod.Status.Phase == "Running" {
			if strings.HasPrefix(pod.Name, lookForController) {
				nRunningControllers += 1
			} else if strings.HasPrefix(pod.Name, lookForNode) {
				nRunningNode += 1
			}
		}
	}

	// Success condition is if there is at least one controller running
	// and all worker nodes have a running node driver pod
	controllersRunning := nRunningControllers != 0
	allNodesRunning := nRunningNode == nWorkerNodes

	return AssertExpectedAndActual(assert.Equal, true, controllersRunning && allNodesRunning,
		fmt.Sprintf("Expected driver controller and node components to be running. controllersRunning = %v, allNodesRunning = %v",
			controllersRunning, allNodesRunning))
}

/* -- Helper functions -- */

func (i *integration) dumpNodeInfo() error {
	list, err := i.k8s.GetClient().CoreV1().Nodes().List(context.Background(), v1.ListOptions{})
	if err != nil {
		message := fmt.Sprintf("listing nodes error: %s\n", err)
		return fmt.Errorf(message)
	}

	for _, node := range list.Items {
		ipAddr := ""
		for _, addr := range node.Status.Addresses {
			if addr.Type == "InternalIP" {
				ipAddr = addr.Address
				break
			}
		}
		isReady := false
		for _, status := range node.Status.Conditions {
			if status.Reason == "KubeletReady" {
				isReady = true
				break
			}
		}
		fmt.Printf("Host: %s IP:%s Ready: %v taint: %s \n", node.Name, ipAddr, isReady, node.Spec.Taints)
		info := node.Status.NodeInfo
		fmt.Printf("\tOS: %s/%s/%s, k8s_version: %s\n", info.OSImage, info.KernelVersion, info.Architecture, info.KubeletVersion)
	}

	return nil
}

func (i *integration) checkIfAllNodesReady() (bool, error) {
	list, err := i.k8s.GetClient().CoreV1().Nodes().List(context.Background(), v1.ListOptions{})
	if err != nil {
		message := fmt.Sprintf("listing nodes error: %s\n", err)
		return false, fmt.Errorf(message)
	}

	readyCount := 0
	for _, node := range list.Items {
		for _, status := range node.Status.Conditions {
			if status.Reason == "KubeletReady" {
				readyCount += 0
				break
			}
		}
	}

	return readyCount == len(list.Items), nil
}

func (i *integration) checkIfAllPodsRunning(namespace string) (bool, error) {
	pods, err := i.k8s.GetClient().CoreV1().Pods(namespace).List(context.Background(), v1.ListOptions{})
	if err != nil {
		return false, err
	}

	podRunningCount := 0
	for _, pod := range pods.Items {
		if pod.Status.Phase == "Running" {
			podRunningCount += 1
		}
	}

	return len(pods.Items) == podRunningCount, nil
}

func (i *integration) checkIfNodesHaveTaint(check string) (bool, error) {
	list, err := i.k8s.GetClient().CoreV1().Nodes().List(context.Background(), v1.ListOptions{})
	if err != nil {
		message := fmt.Sprintf("listing nodes error: %s\n", err)
		return false, fmt.Errorf(message)
	}

	for _, node := range list.Items {
		for _, taint := range node.Spec.Taints {
			if strings.Contains(taint.Key, check) {
				return true, nil
			}
		}
	}

	return false, nil
}

func IntegrationTestScenarioInit(context *godog.ScenarioContext) {
	i := &integration{}
	context.Step(`^a kubernetes "([^"]*)"$`, i.givenKubernetes)
	context.Step(`^all pods are running within (\d+) seconds$`, i.allPodsAreRunningWithinSeconds)
	context.Step(`^I fail "([^"]*)" nodes and "([^"]*)" master nodes with "([^"]*)" failure for (\d+) seconds$`, i.failWorkerAndMasterNodes)
	context.Step(`^"([^"]*)" pods per node with "([^"]*)" volumes and "([^"]*)" devices each$`, i.podsPerNodeWithVolumesAndDevicesEach)
	context.Step(`^the taints for the failed nodes are removed within (\d+) seconds$`, i.theTaintsForTheFailedNodesAreRemovedWithinSeconds)
	context.Step(`^these CSI driver "([^"]*)" are configured on the system$`, i.theseCSIDriverAreConfiguredOnTheSystem)
	context.Step(`^there is a "([^"]*)" in the cluster$`, i.thereIsThisNamespaceInTheCluster)
	context.Step(`^there are driver pods in "([^"]*)" with this "([^"]*)" prefix$`, i.thereAreDriverPodsWithThisPrefix)
}
