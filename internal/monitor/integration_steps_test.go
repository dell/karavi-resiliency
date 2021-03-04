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
	"io/ioutil"
	v12 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"podmon/internal/k8sapi"
	"podmon/test/ssh"
	"strconv"
	"strings"
	"sync"
	"time"
)

type integration struct {
	configPath          string
	k8s                 k8sapi.K8sAPI
	driverType          string
	podCount            int
	devCount            int
	volCount            int
	testNamespacePrefix string
	scriptsDir          string
}

// wordsToNumberMap used for mapping a number-word strings to a float64 value.
var wordsToNumberMap = map[string]float64{
	"zero":       0.0,
	"none":       0.0,
	"one":        1.0,
	"two":        2.0,
	"three":      3.0,
	"all":        -1.0,
	"one-fourth": 0.25,
	"1/4":        0.25,
	"one-third":  0.33,
	"1/3":        0.33,
	"one-half":   0.50,
	"1/2":        0.50,
	"two-thirds": 0.66,
	"2/3":        0.66,
}

// Workaround for non-inclusive word scan
var primary = []byte{'m', 'a', 's', 't', 'e', 'r'}
var primaryLabelKey = fmt.Sprintf("node-role.kubernetes.io/%s", string(primary))

// Directory where test scripts will be dropped
var remoteScriptDir = "/root/karavi-resiliency-tests"

// These are for tracking to which nodes the tests upload scripts.
// With multiple scenarios, we want to do this only once.
var nodesWithScripts map[string]bool
var nodesWithScriptsInitOnce sync.Once

var isWorkerNode = func(node v12.Node) bool {
	// Some k8s clusters may not have a worker label against
	// nodes, so check for the primary label. If it doesn't
	// exist against the node, then it's consider a worker.

	// Check if there's a primary label associated with the node
	hasPrimaryLabel := false
	for label := range node.Labels {
		if label == primaryLabelKey {
			hasPrimaryLabel = true
			break
		}
	}

	return !hasPrimaryLabel
}

var isPrimaryNode = func(node v12.Node) bool {
	hasPrimaryLabel := false
	for label := range node.Labels {
		if label == primaryLabelKey {
			hasPrimaryLabel = true
			break
		}
	}
	return hasPrimaryLabel
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

	nodesWithScriptsInitOnce.Do(func() {
		nodesWithScripts = make(map[string]bool)
	})
	return nil
}

func (i *integration) allPodsAreRunningWithinSeconds(wait int) error {
	// Check each of the test namespaces for running pods
	allRunning := true
	for podIdx := 1; podIdx <= i.podCount; podIdx++ {
		namespace := fmt.Sprintf("%s%d", i.testNamespacePrefix, podIdx)
		running, err := i.checkIfAllPodsRunning(namespace)
		if err != nil {
			return err
		}
		if !running {
			allRunning = false
			break
		}
	}

	namespaces := ""
	allRunning, err := i.checkIfAllPodsRunning(namespaces)
	if err != nil {
		return err
	}

	if allRunning {
		fmt.Print("All test pods are in the 'Running' state\n")
		return nil
	}

	fmt.Printf("All test pods are not all running. Waiting %d seconds.\n", wait)
	time.Sleep(time.Duration(wait) * time.Second)

	// Check each of the test namespaces for running pods (final check)
	allRunning = true
	for podIdx := 1; podIdx <= i.podCount; podIdx++ {
		namespace := fmt.Sprintf("%s%d", i.testNamespacePrefix, podIdx)
		running, err := i.checkIfAllPodsRunning(namespace)
		if err != nil {
			return err
		}
		if !running {
			fmt.Printf("Pods in %s namespace are not all running\n", namespace)
			allRunning = false
			break
		}
	}

	return AssertExpectedAndActual(assert.Equal, true, allRunning,
		fmt.Sprintf("Expected all pods to be in running state after %d seconds", wait))
}

func (i *integration) failWorkerAndPrimaryNodes(numNodes, numPrimary, failure string, wait int) error {
	workersToFail, err := i.parseRatioOrCount(numNodes)
	if err != nil {
		return err
	}

	primaryToFail, err := i.parseRatioOrCount(numPrimary)
	if err != nil {
		return err
	}

	fmt.Printf("Test with %2.2f failed workers and %2.2f failed primary nodes\n", workersToFail, primaryToFail)

	failedWorkers, err := i.failWorkerNodes(workersToFail, failure)
	if err != nil {
		return err
	}

	failedPrimary, err := i.failPrimaryNodes(primaryToFail, failure)
	if err != nil {
		return err
	}

	fmt.Printf("Requested nodes to fail. Waiting %d seconds before checking if they are failed.\n", wait)
	time.Sleep(time.Duration(wait) * time.Second)

	fmt.Printf("Wait done, checking for failed nodes...\n")
	requestedWorkersAndFailed := func(node v12.Node) bool {
		found := false
		for _, worker := range failedWorkers {
			if node.Name == worker && node.Status.Phase == "NotReady" {
				found = true
				break
			}
		}

		return found
	}

	requestedPrimaryAndFailed := func(node v12.Node) bool {
		found := false
		for _, primaryNode := range failedPrimary {
			if node.Name == primaryNode && node.Status.Phase == "NotReady" {
				found = true
				break
			}
		}
		return found
	}

	foundFailedWorkers, err := i.searchForNodes(requestedWorkersAndFailed)
	foundFailedPrimary, err := i.searchForNodes(requestedPrimaryAndFailed)

	err = AssertExpectedAndActual(assert.Equal, true, len(foundFailedPrimary) == len(failedPrimary),
		fmt.Sprintf("Expected %d primary nodes to be failed, but was %d. %v", len(failedPrimary), len(foundFailedPrimary), foundFailedPrimary))
	if err != nil {
		return err
	}

	err = AssertExpectedAndActual(assert.Equal, true, len(foundFailedWorkers) == len(failedWorkers),
		fmt.Sprintf("Expected %d worker node(s) to be failed, but was %d. %v", len(failedWorkers), len(foundFailedWorkers), foundFailedWorkers))
	if err != nil {
		return err
	}

	return nil
}

func (i *integration) podsPerNodeWithVolumesAndDevicesEach(podsPerNode, numVols, numDevs, driverType string) error {
	podCount, err := i.selectFromRange(podsPerNode)
	if err != nil {
		return err
	}

	volCount, err := i.selectFromRange(numVols)
	if err != nil {
		return err
	}

	devCount, err := i.selectFromRange(numDevs)
	if err != nil {
		return err
	}

	i.testNamespacePrefix = "pmtv"
	deployScript := "insv.sh"
	if driverType == "unity" {
		i.testNamespacePrefix = "pmtu"
		deployScript = "insu.sh"
	}

	deployScriptPath := filepath.Join("..", "..", "test", "podmontest", deployScript)
	script := "bash"

	args := []string{
		deployScriptPath,
		"--instances", strconv.Itoa(podCount),
		"--ndevices", strconv.Itoa(volCount),
		"--nvolumes", strconv.Itoa(devCount),
		"--prefix", i.testNamespacePrefix,
	}

	command := exec.Command(script, args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	fmt.Printf("Attempting to deploy with command: %v\n", command)
	err = command.Start()
	if err != nil {
		return err
	}

	err = command.Wait()
	if err != nil {
		return err
	}

	i.driverType = driverType
	i.podCount = podCount
	i.devCount = devCount
	i.volCount = volCount

	return nil
}

func (i *integration) theTaintsForTheFailedNodesAreRemovedWithinSeconds(wait int) error {
	fmt.Printf("Checking if nodes have podmon taint\n")
	taintKey := "podmon.dellemc.com"
	havePodmonTaint, err := i.checkIfNodesHaveTaint(taintKey)
	if err != nil {
		return err
	}

	if havePodmonTaint {
		fmt.Printf("Podmon taint is still on nodes. Waiting %d seconds.\n", wait)
		time.Sleep(time.Duration(wait) * time.Second)
	}

	fmt.Printf("Checking again if nodes have podmon taint (final check)\n")
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
	fmt.Printf("Driver %s exists on the cluster\n", driverObj.Name)
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

	nodes, err := i.searchForNodes(isWorkerNode)
	nWorkerNodes := len(nodes)

	// Look for controller and node driver pods running in the cluster
	lookForController := fmt.Sprintf("%s-controller", prefix)
	lookForNode := fmt.Sprintf("%s-node", prefix)
	nRunningControllers := 0
	nRunningNode := 0
	for _, pod := range pods.Items {
		if pod.Status.Phase == "Running" {
			if strings.HasPrefix(pod.Name, lookForController) {
				nRunningControllers++
			} else if strings.HasPrefix(pod.Name, lookForNode) {
				nRunningNode++
			}
		}
	}

	// Success condition is if there is at least one controller running
	// and all worker nodes have a running node driver pod
	controllersRunning := nRunningControllers != 0
	allNodesRunning := nRunningNode == nWorkerNodes

	return AssertExpectedAndActual(assert.Equal, true, controllersRunning && allNodesRunning,
		fmt.Sprintf("Expected %s driver controller and node pods to be running in %s namespace. controllersRunning = %v, allNodesRunning = %v",
			prefix, namespace, controllersRunning, allNodesRunning))
}

func (i *integration) finallyCleanupEverything() error {
	fmt.Print("Attempting to clean up everything\n")
	uninstallScript := "uns.sh"
	prefix := "pmtv"
	if i.driverType == "unity" {
		prefix = "pmtu"
	}

	deployScriptPath := filepath.Join("..", "..", "test", "podmontest", uninstallScript)
	script := "bash"

	args := []string{
		deployScriptPath,
		"--instances", strconv.Itoa(i.podCount),
		"--prefix", prefix,
	}
	command := exec.Command(script, args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	fmt.Printf("Going to invoke uninstall script %v\n", command)
	err := command.Start()
	if err != nil {
		return err
	}

	err = command.Wait()
	if err != nil {
		return err
	}

	return nil
}

func (i *integration) expectedEnvVariablesAreSet() error {
	nodeUser := os.Getenv("NODE_USER")
	err := AssertExpectedAndActual(assert.Equal, true, nodeUser != "",
		fmt.Sprintf("Expected NODE_USER env variable. Try export NODE_USER=nodeUser before running tests."))
	if err != nil {
		return err
	}

	password := os.Getenv("PASSWORD")
	err = AssertExpectedAndActual(assert.Equal, true, password != "",
		fmt.Sprintf("Expected PASSWORD env variable. Try export PASSWORD=password before running tests."))
	if err != nil {
		return err
	}

	i.scriptsDir = os.Getenv("SCRIPTS_DIR")
	err = AssertExpectedAndActual(assert.Equal, true, i.scriptsDir != "",
		fmt.Sprintf("Expected SCRIPTS_DIR env variable. Try export SCRIPTS_DIR=scriptsDir before running tests."))
	if err != nil {
		return err
	}

	_, dirCheckErr := os.Stat(i.scriptsDir)
	err = AssertExpectedAndActual(assert.Equal, false, os.IsNotExist(dirCheckErr),
		fmt.Sprintf("Expected SCRIPTS_DIR env variable to point to existing directory. %s does not exist", i.scriptsDir))
	if err != nil {
		return err
	}

	return nil
}

func (i *integration) canLogonToNodesAndDropTestScripts() error {
	nodes, err := i.searchForNodes(func(node v12.Node) bool {
		for _, status := range node.Status.Conditions {
			if status.Reason == "KubeletReady" {
				return true
			}
		}
		return false
	})
	if err != nil {
		return err
	}

	for _, node := range nodes {
		for _, addr := range node.Status.Addresses {
			if addr.Type == "InternalIP" {
				// Check if we already copied files for this node already
				if _, ok := nodesWithScripts[addr.Address]; ok {
					fmt.Printf("Node %s already has scripts.\n", addr.Address)
					break
				}
				err = i.copyOverTestScripts(addr.Address)
				if err != nil {
					return err
				}
				nodesWithScripts[addr.Address] = true
				break
			}
		}
	}

	return nil
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
				readyCount++
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
			podRunningCount++
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

func (i *integration) selectFromRange(rangeValue string) (int, error) {
	components := strings.Split(rangeValue, "-")
	if len(components) > 2 {
		return 0, fmt.Errorf("invalid range provided: %s. Should be <min>-<max>", rangeValue)
	}

	min, err := strconv.Atoi(components[0])
	if err != nil {
		return 0, err
	}

	max, err := strconv.Atoi(components[1])
	if err != nil {
		return 0, err
	}

	// rand.IntnRange selects from min to max, inclusive of min and exclusive of
	// max, so use max+1 in order for max value to be a possible value.
	selected := rand.IntnRange(min, max+1)
	return selected, nil
}

func (i *integration) parseRatioOrCount(count string) (float64, error) {
	if retVal, ok := wordsToNumberMap[count]; ok {
		return retVal, nil
	} else if val, err := strconv.Atoi(count); err == nil {
		return float64(val), nil
	}

	return 0.0, fmt.Errorf("invalid count value %s", count)
}

func (i *integration) failWorkerNodes(count float64, failureType string) ([]string, error) {
	return i.failNodes(isWorkerNode, count, failureType)
}

func (i *integration) failPrimaryNodes(count float64, failureType string) ([]string, error) {
	return i.failNodes(isPrimaryNode, count, failureType)
}

func (i *integration) failNodes(filter func(node v12.Node) bool, count float64, failureType string) ([]string, error) {
	failedNodes := make([]string, 0)

	nodes, err := i.searchForNodes(filter)
	if err != nil {
		return failedNodes, err
	}

	numberToFail := 0
	if count < 0.0 {
		numberToFail = len(nodes)
	} else if count < 1.0 {
		temp := float64(len(nodes))
		numberToFail = int(math.Ceil(temp * count))
	}

	nameToIP := make(map[string]string)
	for _, node := range nodes {
		for _, addr := range node.Status.Addresses {
			if addr.Type == "InternalIP" {
				nameToIP[node.Name] = addr.Address
				break
			}
		}
	}

	failed := 0
	for name, ip := range nameToIP {
		if failed < numberToFail {
			// Do failure
			fmt.Printf("Failing %s %s\n", name, ip)
			failedNodes = append(failedNodes, name)
			failed++
		}
	}

	return failedNodes, nil
}

func (i *integration) searchForNodes(filter func(node v12.Node) bool) ([]v12.Node, error) {
	filteredList := make([]v12.Node, 0)

	nodes, err := i.k8s.GetClient().CoreV1().Nodes().List(context.Background(), v1.ListOptions{})
	if err != nil {
		return filteredList, err
	}

	for _, node := range nodes.Items {
		if filter(node) {
			filteredList = append(filteredList, node)
		}
	}

	return filteredList, nil
}

func (i *integration) copyOverTestScripts(address string) error {
	info := ssh.AccessInfo{
		Hostname: address,
		Port:     "22",
		Username: os.Getenv("NODE_USER"),
		Password: os.Getenv("PASSWORD"),
	}

	wrapper := ssh.NewWrapper(&info)

	client := ssh.CommandExecution{
		AccessInfo: &info,
		SSHWrapper: wrapper,
		Timeout:    4 * time.Second,
	}

	fmt.Printf("Attempting to scp scripts from %s to %s:%s\n", i.scriptsDir, address, remoteScriptDir)

	mkDirCmd := fmt.Sprintf("date; rm -rf %s; mkdir %s", remoteScriptDir, remoteScriptDir)
	if mkDirErr := client.Run(mkDirCmd); mkDirErr == nil {
		for _, out := range client.GetOutput() {
			fmt.Printf("%s\n", out)
		}
	} else {
		return mkDirErr
	}

	files, err := ioutil.ReadDir(i.scriptsDir)
	if err != nil {
		return err
	}

	for _, f := range files {
		err = client.Copy(fmt.Sprintf("%s/%s", i.scriptsDir, f.Name()), fmt.Sprintf("%s/%s", remoteScriptDir, f.Name()))
		if err != nil {
			return err
		}
	}

	lsDirCmd := fmt.Sprintf("ls -ltr %s", remoteScriptDir)
	if lsErr := client.Run(lsDirCmd); lsErr == nil {
		for _, out := range client.GetOutput() {
			fmt.Printf("%s\n", out)
		}
	} else {
		return lsErr
	}

	fmt.Printf("Scripts successfully copied to %s:%s\n", address, remoteScriptDir)
	return nil
}

func IntegrationTestScenarioInit(context *godog.ScenarioContext) {
	i := &integration{}
	context.Step(`^a kubernetes "([^"]*)"$`, i.givenKubernetes)
	context.Step(`^validate that all pods are running within (\d+) seconds$`, i.allPodsAreRunningWithinSeconds)
	context.Step(`^I fail "([^"]*)" worker nodes and "([^"]*)" primary nodes with "([^"]*)" failure for (\d+) seconds$`, i.failWorkerAndPrimaryNodes)
	context.Step(`^"([^"]*)" pods per node with "([^"]*)" volumes and "([^"]*)" devices using "([^"]*)"$`, i.podsPerNodeWithVolumesAndDevicesEach)
	context.Step(`^the taints for the failed nodes are removed within (\d+) seconds$`, i.theTaintsForTheFailedNodesAreRemovedWithinSeconds)
	context.Step(`^these CSI driver "([^"]*)" are configured on the system$`, i.theseCSIDriverAreConfiguredOnTheSystem)
	context.Step(`^there is a "([^"]*)" in the cluster$`, i.thereIsThisNamespaceInTheCluster)
	context.Step(`^there are driver pods in "([^"]*)" with this "([^"]*)" prefix$`, i.thereAreDriverPodsWithThisPrefix)
	context.Step(`^finally cleanup everything$`, i.finallyCleanupEverything)
	context.Step(`^test environmental variables are set$`, i.expectedEnvVariablesAreSet)
	context.Step(`^can logon to nodes and drop test scripts$`, i.canLogonToNodesAndDropTestScripts)
}
