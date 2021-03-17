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
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	labeledPodsToNodes  map[string]string
}

// Used for keeping of track of the last test that was
// run, so that we can clean up in case of failure
var lastTestDriverType string

// Keep track if the test should result in a podmon taint against the failed node(s)
var testExpectedPodmonTaint bool

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

// failureToScriptMap maps the failType in the Gerkin file to a script to invoke that failure
var failureToScriptMap = map[string]string{
	"interfacedown": "bounce.ip",
	"reboot":        "reboot.node",
}

const (
	// Timeout for the SSH client
	sshTimeout = 10 * time.Second
	// Directory where test scripts will be dropped
	remoteScriptDir = "/root/karavi-resiliency-tests"
	// An int value representing number of seconds to periodically check status
	checkTickerInterval = 10
)

// Workaround for non-inclusive word scan
var primary = []byte{'m', 'a', 's', 't', 'e', 'r'}
var primaryLabelKey = fmt.Sprintf("node-role.kubernetes.io/%s", string(primary))

// These are for tracking to which nodes the tests upload scripts.
// With multiple scenarios, we want to do this only once.
var nodesWithScripts map[string]bool
var nodesWithScriptsInitOnce sync.Once

// Parameters for use with the background poller
var k8sPollInterval = 2 * time.Second
var pollTick *time.Ticker

// isWorkerNode is a filter function for searching for nodes that look to be worker nodes
var isWorkerNode = func(node corev1.Node) bool {
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

// isPrimaryNode is a filter function for searching for nodes that look to be primary nodes
var isPrimaryNode = func(node corev1.Node) bool {
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
		message := fmt.Sprintf("kubernetes connection error: %s", err)
		log.Info(message)
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
	allRunning, err := i.allPodsInTestNamespacesAreRunning()
	if err != nil {
		return err
	}

	if allRunning {
		log.Info("All test pods are in the 'Running' state")
		return nil
	}

	log.Infof("Test pods are not all running. Waiting up to %d seconds.", wait)
	timeoutDuration := time.Duration(wait) * time.Second
	timeout := time.NewTimer(timeoutDuration)
	ticker := time.NewTicker(checkTickerInterval * time.Second)
	done := make(chan bool)
	start := time.Now()

	go func() {
		for {
			select {
			case <-timeout.C:
				log.Info("Timed out, but doing a last check to see if all test pods are running")
				// Check each of the test namespaces for running pods (final check)
				allRunning, err = i.allPodsInTestNamespacesAreRunning()
				done <- true
			case <-ticker.C:
				log.Infof("Checking if all test pods are running (time left %v)", timeoutDuration-time.Since(start))
				// Check each of the test namespaces for running pods (final check)
				allRunning, err = i.allPodsInTestNamespacesAreRunning()
				if allRunning {
					done <- true
				}
			}
		}
	}()

	<-done
	timeout.Stop()
	ticker.Stop()

	if err != nil {
		return err
	}
	log.Infof("Completed pod running check after %v (allRunning=%v)", time.Since(start), allRunning)

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

	log.Infof("Test with %2.2f failed workers and %2.2f failed primary nodes", workersToFail, primaryToFail)

	failedWorkers, err := i.failWorkerNodes(workersToFail, failure, wait)
	if err != nil {
		return err
	}

	failedPrimary, err := i.failPrimaryNodes(primaryToFail, failure, wait)
	if err != nil {
		return err
	}

	testExpectedPodmonTaint = i.nodeHasLabeledPods(failedWorkers) || i.nodeHasLabeledPods(failedPrimary)

	log.Infof("Requested nodes to fail. Waiting up to %d seconds to see if they show up as failed.", wait)
	timeoutDuration := time.Duration(wait) * time.Second
	timeout := time.NewTimer(timeoutDuration)
	ticker := time.NewTicker(checkTickerInterval * time.Second)
	done := make(chan bool)
	start := time.Now()

	log.Infof("Wait done, checking for failed nodes...")
	requestedWorkersAndFailed := func(node corev1.Node) bool {
		found := false
		for _, worker := range failedWorkers {
			if node.Name == worker && i.isNodeFailed(node, testExpectedPodmonTaint) {
				found = true
				break
			}
		}

		return found
	}

	requestedPrimaryAndFailed := func(node corev1.Node) bool {
		found := false
		for _, primaryNode := range failedPrimary {
			if node.Name == primaryNode && i.isNodeFailed(node, testExpectedPodmonTaint) {
				found = true
				break
			}
		}
		return found
	}

	foundFailedWorkers, err := i.searchForNodes(requestedWorkersAndFailed)
	foundFailedPrimary, err := i.searchForNodes(requestedPrimaryAndFailed)

	go func() {
		for {
			select {
			case <-timeout.C:
				log.Info("Timed out, but doing last check if requested nodes show up as failed")
				foundFailedWorkers, err = i.searchForNodes(requestedWorkersAndFailed)
				foundFailedPrimary, err = i.searchForNodes(requestedPrimaryAndFailed)
				done <- true
			case <-ticker.C:
				log.Infof("Checking if requested nodes show up as failed (time left %v)", timeoutDuration-time.Since(start))
				foundFailedWorkers, err = i.searchForNodes(requestedWorkersAndFailed)
				foundFailedPrimary, err = i.searchForNodes(requestedPrimaryAndFailed)
				if len(foundFailedPrimary) == len(failedPrimary) && len(foundFailedWorkers) == len(failedWorkers) {
					done <- true
				}
			}
		}
	}()

	<-done
	timeout.Stop()
	ticker.Stop()

	log.Infof("Completed checks for failed nodes after %v", time.Since(start))

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

func (i *integration) deployPods(podsPerNode, numVols, numDevs, driverType, storageClass string, wait int) error {
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
		"--storage-class", storageClass,
	}

	command := exec.Command(script, args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	log.Infof("Attempting to deploy with command: %v", command)
	err = command.Start()
	if err != nil {
		return err
	}

	err = command.Wait()
	if err != nil {
		return err
	}

	i.setDriverType(driverType)
	i.podCount = podCount
	i.devCount = devCount
	i.volCount = volCount

	log.Infof("Waiting up to %d seconds for pods to deploy", wait)
	runningCount := 0
	timeoutDuration := time.Duration(wait) * time.Second
	timeout := time.NewTimer(timeoutDuration)
	ticker := time.NewTicker(checkTickerInterval * time.Second)
	done := make(chan bool)
	start := time.Now()

	go func() {
		for {
			select {
			case <-timeout.C:
				log.Info("Timed out, but doing last check if test pods are running")
				runningCount = i.getNumberOfRunningTestPods()
				done <- true
			case <-ticker.C:
				log.Infof("Check if test pods are running (time left %v)", timeoutDuration-time.Since(start))
				runningCount = i.getNumberOfRunningTestPods()
				if runningCount == i.podCount {
					done <- true
				}
			}
		}
	}()

	<-done
	timeout.Stop()
	ticker.Stop()

	log.Infof("Test pods running check finished after %v", time.Since(start))
	err = AssertExpectedAndActual(assert.Equal, i.podCount, runningCount,
		fmt.Sprintf("Expected %d test pods to be running after %d seconds", i.podCount, wait))
	if err != nil {
		return err
	}

	// For the test pods number of namespaces = podCount
	for n := 1; n <= podCount; n++ {
		ns := fmt.Sprintf("%s%d", i.testNamespacePrefix, n)
		nsErr := i.thereIsThisNamespaceInTheCluster(ns)
		if nsErr != nil {
			return nsErr
		}
	}

	if err = i.populateLabeledPodsToNodes(); err != nil {
		return err
	}

	return nil
}

func (i *integration) theTaintsForTheFailedNodesAreRemovedWithinSeconds(wait int) error {
	if err := i.waitOnNodesToBeReady(wait); err != nil {
		return err
	}
	return i.waitOnTaintRemoval(wait)
}

func (i *integration) theseCSIDriverAreConfiguredOnTheSystem(driverName string) error {
	driverObj, err := i.k8s.GetClient().StorageV1().CSIDrivers().Get(context.Background(), driverName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	log.Infof("Driver %s exists on the cluster", driverObj.Name)
	return AssertExpectedAndActual(assert.Equal, driverName, driverObj.Name,
		fmt.Sprintf("No CSIDriver named %s found in cluster", driverName))
}

func (i *integration) thereIsThisNamespaceInTheCluster(namespace string) error {
	foundNamespace := false
	namespaces, err := i.k8s.GetClient().CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
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
	pods, err := i.k8s.GetClient().CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
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
	nRunningControllerPodmons := 0
	nRunningNodePodmons := 0
	for _, pod := range pods.Items {
		if pod.Status.Phase == "Running" {
			if strings.HasPrefix(pod.Name, lookForController) {
				if i.podmonContainerRunning(pod) {
					nRunningControllerPodmons++
				}
				nRunningControllers++
			} else if strings.HasPrefix(pod.Name, lookForNode) {
				if i.podmonContainerRunning(pod) {
					nRunningNodePodmons++
				}
				nRunningNode++
			}
		}
	}

	// Success condition is:
	//  - At least one controller running
	//  - All worker nodes have a running node driver pod
	//  - There is a controller podmon container running
	//  - There are node podmon containers running
	controllersRunning := nRunningControllers != 0
	allNodesRunning := nRunningNode == nWorkerNodes

	// First, check if we have the expected pods running
	err = AssertExpectedAndActual(assert.Equal, true, controllersRunning && allNodesRunning,
		fmt.Sprintf("Expected %s driver controller and node pods to be running in %s namespace. controllersRunning = %v, allNodesRunning = %v",
			prefix, namespace, controllersRunning, allNodesRunning))
	if err != nil {
		return err
	}

	// Second, check if we have running podmon containers that we expect
	controllerPodmonsGood := (nRunningControllerPodmons > 0) && nRunningControllerPodmons == nRunningControllers
	nodePodmonsGood := (nRunningNodePodmons > 0) && nRunningNodePodmons == nRunningNode
	return AssertExpectedAndActual(assert.Equal, true, controllerPodmonsGood && nodePodmonsGood,
		fmt.Sprintf("Expected podmon container to be running in %s controller and node pods. Number of controller podmon is %d. Number of node podmon is %d",
			prefix, nRunningControllerPodmons, nRunningNodePodmons))
}

func (i *integration) finallyCleanupEverything() error {
	uninstallScript := "uns.sh"
	prefix := "pmtv"
	if lastTestDriverType == "unity" {
		prefix = "pmtu"
	}

	if lastTestDriverType == "" {
		// Nothing to clean up
		return nil
	}

	log.Infof("Attempting to clean up everything for driverType '%s'", lastTestDriverType)

	deployScriptPath := filepath.Join("..", "..", "test", "podmontest", uninstallScript)
	script := "bash"

	args := []string{
		deployScriptPath,
		"--prefix", prefix,
		"--all", lastTestDriverType,
	}
	command := exec.Command(script, args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	log.Infof("Going to invoke uninstall script %v", command)
	err := command.Start()
	if err != nil {
		return err
	}

	err = command.Wait()
	if err != nil {
		return err
	}

	// Cleaned up, so zero the podCount
	i.podCount = 0
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
	nodes, err := i.searchForNodes(func(node corev1.Node) bool {
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
					log.Infof("Node %s already has scripts.", addr.Address)
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

func (i *integration) theseStorageClassesExistInTheCluster(storageClassList string) error {
	list, err := i.k8s.GetClient().StorageV1().StorageClasses().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		message := fmt.Sprintf("listing StorageClasses error: %s", err)
		return fmt.Errorf(message)
	}

	// storageClassList is a comma delimited list of storageClasses to check for in the cluster
	for _, expected := range strings.Split(storageClassList, ",") {
		expected = strings.TrimSpace(expected)
		foundIt := false
		for _, sc := range list.Items {
			if expected == sc.Name {
				foundIt = true
				break
			}
		}
		err = AssertExpectedAndActual(assert.Equal, true, foundIt,
			fmt.Sprintf("Expected '%s' StorageClass in the cluster, but was not found", expected))
		if err != nil {
			return err
		}
	}

	return nil
}

/* -- Helper functions -- */

func (i *integration) dumpNodeInfo() error {
	list, err := i.k8s.GetClient().CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		message := fmt.Sprintf("listing nodes error: %s", err)
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
		log.Infof("Host: %s IP:%s Ready: %v taint: %s ", node.Name, ipAddr, isReady, node.Spec.Taints)
		info := node.Status.NodeInfo
		log.Infof("\tOS: %s/%s/%s, k8s_version: %s", info.OSImage, info.KernelVersion, info.Architecture, info.KubeletVersion)
	}

	return nil
}

func (i *integration) checkIfAllNodesReady() (bool, error) {
	list, err := i.k8s.GetClient().CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		message := fmt.Sprintf("listing nodes error: %s", err)
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
	pods, err := i.k8s.GetClient().CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
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

// checkIfNodesHaveTaints takes a string parameter representing a comma delimited
// string of taint keys to check against the nodes in the cluster. It returns true
// iff *any* one of the taints is found on the nodes.
func (i *integration) checkIfNodesHaveTaints(check string) (bool, error) {
	list, err := i.k8s.GetClient().CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		message := fmt.Sprintf("listing nodes error: %s", err)
		return false, fmt.Errorf(message)
	}

	checkTaints := strings.Split(check, ",")
	for _, node := range list.Items {
		for _, taint := range node.Spec.Taints {
			for _, checkTaint := range checkTaints {
				if strings.Contains(taint.Key, checkTaint) {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

// selectFromRange takes the 'rangeValue' and returns a random value within that range (inclusive of min and max of the range).
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

	if min < -1 {
		return -1, fmt.Errorf("invalid range. Minimum is less than 1 (%d)", min)
	}

	if max < 1 {
		return -1, fmt.Errorf("invalid range. Maximum is less than 1 (%d)", max)
	}

	if min > max {
		return -1, fmt.Errorf("invalid range. Minimum value specified is greater than max (%d > %d)", min, max)
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

func (i *integration) failWorkerNodes(count float64, failureType string, wait int) ([]string, error) {
	return i.failNodes(isWorkerNode, count, failureType, wait)
}

func (i *integration) failPrimaryNodes(count float64, failureType string, wait int) ([]string, error) {
	return i.failNodes(isPrimaryNode, count, failureType, wait)
}

// failNodes applies the node filter and count to determine which nodes should
// be failed with the 'failureType'. The count is a float value based on the
// test spec that allows for a number or a ratio (e.g., "one-third"). When the
// count is a ratio, the total number of nodes to fail would be based on that
// ratio applied against the number of filtered nodes.
//
// Once this number is determined, a random list of the filtered nodes will be
// failed based on the 'failureType'.
func (i *integration) failNodes(filter func(node corev1.Node) bool, count float64, failureType string, wait int) ([]string, error) {
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
	} else { // count >= 1.0, so use the value
		numberToFail = int(count)
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
			if err = i.induceFailureOn(name, ip, failureType, wait); err != nil {
				return failedNodes, err
			}
			log.Infof("Failing %s %s", name, ip)
			failedNodes = append(failedNodes, name)
			failed++
		}
	}

	return failedNodes, nil
}

// searchForNodes returns an array of nodes from the k8s system that match the 'filter'
func (i *integration) searchForNodes(filter func(node corev1.Node) bool) ([]corev1.Node, error) {
	filteredList := make([]corev1.Node, 0)

	nodes, err := i.k8s.GetClient().CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
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

// copyOverTestScripts will SCP the scripts for inducing failures to the remote system 'address'
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
		Timeout:    sshTimeout,
	}

	log.Infof("Attempting to scp scripts from %s to %s:%s", i.scriptsDir, address, remoteScriptDir)

	mkDirCmd := fmt.Sprintf("date; rm -rf %s; mkdir %s", remoteScriptDir, remoteScriptDir)
	if mkDirErr := client.Run(mkDirCmd); mkDirErr == nil {
		for _, out := range client.GetOutput() {
			log.Infof("%s", out)
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

	// After copying the files, add execute permissions and list the directory
	lsDirCmd := fmt.Sprintf("chmod +x %s/* ; ls -ltr %s", remoteScriptDir, remoteScriptDir)
	if lsErr := client.Run(lsDirCmd); lsErr == nil {
		for _, out := range client.GetOutput() {
			log.Infof("%s", out)
		}
	} else {
		return lsErr
	}

	log.Infof("Scripts successfully copied to %s:%s", address, remoteScriptDir)
	return nil
}

func (i *integration) allPodsInTestNamespacesAreRunning() (bool, error) {
	allRunning := true
	for podIdx := 1; podIdx <= i.podCount; podIdx++ {
		namespace := fmt.Sprintf("%s%d", i.testNamespacePrefix, podIdx)
		running, err := i.checkIfAllPodsRunning(namespace)
		if err != nil {
			return false, err
		}
		if !running {
			log.Infof("Pods in %s namespace are not all running", namespace)
			allRunning = false
			// Don't break, we want to check all the namespaces so that we display which ones aren't running
		}
	}
	return allRunning, nil
}

// induceFailureOn will initiate a failure of the 'failureType' against the host at 'ip'.
// The 'wait' will be passed as parameter to the invocation script. If it is applicable,
// that 'wait' value indicates how long the failure should be active before it should
// go back into a non-failure state.
func (i *integration) induceFailureOn(name string, ip, failureType string, wait int) error {
	info := ssh.AccessInfo{
		Hostname: ip,
		Port:     "22",
		Username: os.Getenv("NODE_USER"),
		Password: os.Getenv("PASSWORD"),
	}

	wrapper := ssh.NewWrapper(&info)

	client := ssh.CommandExecution{
		AccessInfo: &info,
		SSHWrapper: wrapper,
		Timeout:    sshTimeout,
	}

	log.Infof("Attempting to induce the %s failure on %s/%s and waiting %d seconds", failureType, name, ip, wait)
	scriptToUse, ok := failureToScriptMap[failureType]
	if !ok {
		return fmt.Errorf("no mapping for failureType %s", failureType)
	}

	// Invoke script allows us to programmatically invoke the failure script and not fail the SSH session
	invokerScript := fmt.Sprintf("%s/invoke.sh", remoteScriptDir)
	failureScript := fmt.Sprintf("%s/%s", remoteScriptDir, scriptToUse)
	invokeFailCmd := fmt.Sprintf("%s %s --seconds %d", invokerScript, failureScript, wait)
	log.Infof("Command to invoke: %s", invokeFailCmd)
	if invokeErr := client.SendRequest(invokeFailCmd); invokeErr == nil {
		for _, out := range client.GetOutput() {
			log.Infof("%s", out)
		}
	} else {
		return invokeErr
	}

	return nil
}

func (i *integration) podmonContainerRunning(pod corev1.Pod) bool {
	for index, container := range pod.Spec.Containers {
		if container.Name == "podmon" {
			podmonIsReady := pod.Status.ContainerStatuses[index].Ready
			log.Infof("podmon %s on %s/%s is Ready=%v", container.Image, pod.Name, pod.Spec.NodeName, podmonIsReady)
			log.Infof("podmon %s on %s/%s args: %s", container.Image, pod.Name, pod.Spec.NodeName, strings.Join(container.Args, " "))
			if podmonIsReady {
				return true
			}
		}
	}
	return false
}

func nodeHasCondition(node corev1.Node, conditionType corev1.NodeConditionType) bool {
	for _, condition := range node.Status.Conditions {
		if conditionType == condition.Type {
			if condition.Status == "True" {
				return true
			}
		}
	}
	return false
}

func (i *integration) k8sPoll() {
	list, getNodesErr := i.k8s.GetClient().CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if getNodesErr == nil {
		for _, node := range list.Items {
			nodeReady := nodeHasCondition(node, "Ready")
			taintKeys := make([]string, 0)
			for _, taint := range node.Spec.Taints {
				taintKeys = append(taintKeys, taint.Key)
			}
			log.Infof("k8sPoll: Node: %s Ready:%v Taints: %s", node.Name, nodeReady, strings.Join(taintKeys, ","))
		}
	} else {
		log.Infof("k8sPoll: listing nodes error: %s", getNodesErr)
	}

	if i.driverType != "" {
		pods, getPodsErr := i.listPodsByLabel(fmt.Sprintf("podmon.dellemc.com/driver=csi-%s", i.driverType))
		if getPodsErr == nil {
			for _, pod := range pods.Items {
				log.Infof("k8sPoll: %s %s/%s %s", pod.Spec.NodeName, pod.Namespace, pod.Name, pod.Status.Phase)
			}
		} else {
			log.Infof("k8sPoll: get pods failed: %s", getPodsErr)
		}
	}
}

func (i *integration) getNumberOfRunningTestPods() int {
	if pods, getPodsErr := i.listPodsByLabel(fmt.Sprintf("podmon.dellemc.com/driver=csi-%s", lastTestDriverType)); getPodsErr == nil {
		nRunning := 0
		for _, pod := range pods.Items {
			if pod.Status.Phase == "Running" {
				nRunning++
			}
		}
		return nRunning
	}
	return 0
}

func (i *integration) listPodsByLabel(label string) (*corev1.PodList, error) {
	return i.k8s.GetClient().CoreV1().Pods("").List(context.Background(), metav1.ListOptions{LabelSelector: label})
}

func (i *integration) startK8sPoller() {
	for {
		select {
		case <-pollTick.C:
			i.k8sPoll()
		}
	}
}

func (i *integration) setDriverType(driver string) {
	i.driverType = driver
	lastTestDriverType = driver
}

// populateLabeledPodsToNodes fills in the integration.labeledPodsToNodes
// with the mapping of the labeled pods to node names. The key in the
// labeledPodsToNodes map will be in this format: "<namespace>/<podname>".
func (i *integration) populateLabeledPodsToNodes() error {
	i.labeledPodsToNodes = make(map[string]string)
	pods, getPodsErr := i.listPodsByLabel(fmt.Sprintf("podmon.dellemc.com/driver=csi-%s", lastTestDriverType))
	if getPodsErr != nil {
		return getPodsErr
	}

	for _, pod := range pods.Items {
		nsPodName := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)
		i.labeledPodsToNodes[nsPodName] = pod.Spec.NodeName
	}
	return nil
}

// nodeHasLabeledPods will search through the integration.labeledPodsToNodes map looking
// for any node names from 'checkNodeNames' that matches. If found, returns true.
func (i *integration) nodeHasLabeledPods(checkNodeNames []string) bool {
	for _, nodeName := range i.labeledPodsToNodes {
		for _, checkNodeName := range checkNodeNames {
			if nodeName == checkNodeName {
				return true
			}
		}
	}
	return false
}

func (i *integration) isNodeFailed(node corev1.Node, expectPodmonTaint bool) bool {
	isFailed := false
	if expectPodmonTaint {
		podmonTaint := fmt.Sprintf("%s.%s", lastTestDriverType, PodmonTaintKeySuffix)
		isFailed = nodeHasTaint(&node, podmonTaint, corev1.TaintEffectNoSchedule)
	} else {
		nodeIsNotReady := !nodeHasCondition(node, "Ready")
		isFailed = nodeIsNotReady
	}
	return isFailed
}

func (i *integration) waitOnNodesToBeReady(wait int) error {
	log.Infof("Checking if all the nodes are in 'Ready' state")
	var notReadyNodes []corev1.Node
	var err error

	thatAreNotReady := func(node corev1.Node) bool {
		return !nodeHasCondition(node, "Ready")
	}

	// Check now if all the nodes are ready
	allReady := false
	if notReadyNodes, err = i.searchForNodes(thatAreNotReady); err == nil {
		allReady = len(notReadyNodes) == 0
	}
	if allReady {
		return nil
	}

	// If we get here then, the nodes are not all ready, so check at an interval with a maximum 'wait' timeout
	timeoutDuration := time.Duration(wait) * time.Second
	timeout := time.NewTimer(timeoutDuration)
	ticker := time.NewTicker(checkTickerInterval * time.Second)
	done := make(chan bool)
	start := time.Now()

	go func() {
		for {
			select {
			case <-timeout.C:
				log.Infof("Timed out, but checking if all nodes are ready")
				if notReadyNodes, err = i.searchForNodes(thatAreNotReady); err == nil {
					allReady = len(notReadyNodes) == 0
				}
				done <- true
			case <-ticker.C:
				log.Infof("Checking if all nodes are ready (time left %v)", timeoutDuration-time.Since(start))
				if notReadyNodes, err = i.searchForNodes(thatAreNotReady); err == nil {
					if allReady = len(notReadyNodes) == 0; allReady {
						done <- true
					}
				}
			}
		}
	}()

	<-done
	timeout.Stop()
	ticker.Stop()

	log.Infof("Done checking if all nodes are ready after %v", time.Since(start))

	if err != nil {
		return err
	}

	return AssertExpectedAndActual(assert.Equal, true, allReady,
		fmt.Sprintf("Expected all nodes to be 'Ready' in %d seconds", wait))
}

func (i *integration) waitOnTaintRemoval(wait int) error {
	log.Infof("Checking if nodes have taints")
	// Should minimally expect to the the Kubernetes unreachable taint on the failed node
	taintKeys := "node.kubernetes.io/unreachable"
	if testExpectedPodmonTaint {
		// If the test failed some node(s) that had labeled pods in it, then we
		// expect the podmon taint to be cleaned up as well.
		taintKeys = taintKeys + "," + fmt.Sprintf("%s.%s", lastTestDriverType, PodmonTaintKeySuffix)
	}
	log.Infof("These taints should not be on the nodes: %s", taintKeys)
	hasTaints, err := i.checkIfNodesHaveTaints(taintKeys)
	if err != nil {
		return err
	}

	if hasTaints {
		log.Infof("Taints are still on nodes. Waiting up to %d seconds until the taint is removed.", wait)
	} else {
		log.Infof("Taints were not found on the nodes.")
		return nil
	}

	timeoutDuration := time.Duration(wait) * time.Second
	timeout := time.NewTimer(timeoutDuration)
	ticker := time.NewTicker(checkTickerInterval * time.Second)
	done := make(chan bool)
	start := time.Now()

	go func() {
		for {
			select {
			case <-timeout.C:
				log.Infof("Timed out, but checking again if nodes have podmon taint")
				hasTaints, err = i.checkIfNodesHaveTaints(taintKeys)
				done <- true
			case <-ticker.C:
				log.Infof("Checking if podmon taints have been removed (time left %v)", timeoutDuration-time.Since(start))
				hasTaints, err = i.checkIfNodesHaveTaints(taintKeys)
				if err == nil && !hasTaints {
					done <- true
				}
			}
		}
	}()

	<-done
	timeout.Stop()
	ticker.Stop()

	log.Infof("Done checking if taint removed after %v", time.Since(start))

	return AssertExpectedAndActual(assert.Equal, false, hasTaints,
		fmt.Sprintf("Expected %s taint(s) to be removed after %d seconds, but still exist", taintKeys, wait))
}

func IntegrationTestScenarioInit(context *godog.ScenarioContext) {
	i := &integration{}
	pollK8sEnabled := false
	if pollK8sStr := os.Getenv("POLL_K8S"); strings.ToLower(pollK8sStr) == "true" {
		pollK8sEnabled = true
	}
	context.BeforeScenario(func(sc *godog.Scenario) {
		if pollK8sEnabled {
			pollTick = time.NewTicker(k8sPollInterval)
			go i.startK8sPoller()
		}
	})
	context.AfterScenario(func(sc *godog.Scenario, err error) {
		if pollK8sEnabled {
			pollTick.Stop()
		}
	})
	context.Step(`^a kubernetes "([^"]*)"$`, i.givenKubernetes)
	context.Step(`^validate that all pods are running within (\d+) seconds$`, i.allPodsAreRunningWithinSeconds)
	context.Step(`^I fail "([^"]*)" worker nodes and "([^"]*)" primary nodes with "([^"]*)" failure for (\d+) seconds$`, i.failWorkerAndPrimaryNodes)
	context.Step(`^"([^"]*)" pods per node with "([^"]*)" volumes and "([^"]*)" devices using "([^"]*)" and "([^"]*)" in (\d+)$`, i.deployPods)
	context.Step(`^the taints for the failed nodes are removed within (\d+) seconds$`, i.theTaintsForTheFailedNodesAreRemovedWithinSeconds)
	context.Step(`^these CSI driver "([^"]*)" are configured on the system$`, i.theseCSIDriverAreConfiguredOnTheSystem)
	context.Step(`^there is a "([^"]*)" in the cluster$`, i.thereIsThisNamespaceInTheCluster)
	context.Step(`^there are driver pods in "([^"]*)" with this "([^"]*)" prefix$`, i.thereAreDriverPodsWithThisPrefix)
	context.Step(`^finally cleanup everything$`, i.finallyCleanupEverything)
	context.Step(`^cluster is clean of test pods$`, i.finallyCleanupEverything)
	context.Step(`^test environmental variables are set$`, i.expectedEnvVariablesAreSet)
	context.Step(`^can logon to nodes and drop test scripts$`, i.canLogonToNodesAndDropTestScripts)
	context.Step(`^these storageClasses "([^"]*)" exist in the cluster$`, i.theseStorageClassesExistInTheCluster)
	context.Step(`^wait (\d+) to see there are no taints$`, i.theTaintsForTheFailedNodesAreRemovedWithinSeconds)
}
