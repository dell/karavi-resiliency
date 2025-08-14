// Copyright © 2021-2023 Dell Inc. or its subsidiaries. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package monitor

import (
	"context"
	"errors"
	"fmt"
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

	"github.com/cucumber/godog"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
)

type integration struct {
	configPath            string
	k8s                   k8sapi.K8sAPI
	driverType            string
	podCount              int
	devCount              int
	volCount              int
	testNamespacePrefix   map[string]bool
	scriptsDir            string
	labeledPodsToNodes    map[string]string
	nodesToTaints         map[string]string
	isOpenshift           bool
	bastionNode           string
	customTaints          string
	preferredLabeledNodes []string
}

// Used for keeping of track of the last test that was
// run, so that we can clean up in case of failure
var lastTestDriverType string

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
	"kubeletdown":   "bounce.kubelet",
	"driverpod":     "failpods.sh",
}

const (
	// SSH timeout value
	sshTimeoutValue = 120
	// Timeout for the SSH client
	sshTimeoutDuration = sshTimeoutValue * time.Second
	// Directory where test scripts will be dropped
	remoteScriptDir = "/root/karavi-resiliency-tests"
	// Directory on Openshift nodes where the scripts will be dropped
	openShiftRemoteScriptDir = "/usr/tmp/karavi-resiliency-tests"
	// An int value representing number of seconds to periodically check status
	checkTickerInterval = 10
	stopFilename        = "stop_test"
	OpenshiftBastion    = "OPENSHIFT_BASTION"
	UnprotectedPodsNS   = "unlabeled"
	PowerflexNS         = "pmtv"
	UnityNS             = "pmtu"
	PowerScaleNS        = "pmti"
	PowerStoreNS        = "pmtps"
	PowerMaxNS          = "pmtpm"
	VM                  = "vm"
)

// Used for stopping the test from continuing
var stopTestRequested bool

// Workaround for non-inclusive word scan
var (
	primary         = []byte{'m', 'a', 's', 't', 'e', 'r'}
	primaryLabelKey = fmt.Sprintf("node-role.kubernetes.io/%s", string(primary))
	controlPlane    = "node-role.kubernetes.io/control-plane"
)

// These are for tracking to which nodes the tests upload scripts.
// With multiple scenarios, we want to do this only once.
var (
	nodesWithScripts         map[string]bool
	nodesWithScriptsInitOnce sync.Once
)

// Parameters for use with the background poller
var (
	k8sPollInterval = 2 * time.Second
	pollTick        *time.Ticker
)

// sshOptions used in SSH cli commands to K8s nodes
var sshOptions = fmt.Sprintf("-o 'ConnectTimeout=%d' -o 'UserKnownHostsFile /dev/null' -o 'StrictHostKeyChecking no'", sshTimeoutValue)

// isWorkerNode is a filter function for searching for nodes that look to be worker nodes
var isWorkerNode = func(node corev1.Node) bool {
	// Some k8s clusters may not have a worker label against
	// nodes, so check for the primary label. If it doesn't
	// exist against the node, then it's consider a worker.

	// Check if there's a primary label associated with the node
	return !isPrimaryNode(node)
}

// isPrimaryNode is a filter function for searching for nodes that look to be primary nodes
var isPrimaryNode = func(node corev1.Node) bool {
	hasPrimaryLabel := false
	for label := range node.Labels {
		if label == primaryLabelKey || label == controlPlane {
			hasPrimaryLabel = true
			break
		}
	}
	return hasPrimaryLabel
}

func (i *integration) givenKubernetes(configPath string) error {
	// Check if there was a request to stop the integration test. All tests would
	// need to go through this step of getting the Kubernetes configuration, so
	// it would be appropriate to do the check here to prevent further tests.
	if stopTestRequested {
		return godog.ErrUndefined
	}

	// Look for a "stop" file. If found, we signal that the tests should stop.
	if fileInfo, stopFileErr := os.Stat(stopFilename); stopFileErr == nil {
		log.Infof("Found stop test file %s", fileInfo.Name())
		stopTestRequested = true
		// Clean up the stop file, so that the test can be rerun.
		os.Remove(fileInfo.Name())
		return godog.ErrUndefined
	}

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
		return fmt.Errorf("%s", message)
	}

	i.isOpenshift, err = i.detectOpenshift()
	if err != nil {
		return err
	}

	if i.isOpenshift {
		// Expecting env var pointing to the Bastion node hostname/IP
		i.bastionNode = os.Getenv(OpenshiftBastion)
	}

	err = i.dumpNodeInfo()
	if err != nil {
		return err
	}

	i.nodesToTaints = make(map[string]string)

	nodesWithScriptsInitOnce.Do(func() {
		nodesWithScripts = make(map[string]bool)
	})

	i.testNamespacePrefix = make(map[string]bool)

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
	return i.internalFailWorkerAndPrimaryNodes(numNodes, numPrimary, failure, "", wait)
}

func (i *integration) failLabeledNodes(preferred, failure string, wait int) error {
	failedWorkers, err := i.failNodes(func(node corev1.Node) bool {
		return node.Labels["preferred"] == preferred
	}, -1, failure, wait)
	if err != nil {
		return err
	}

	err = i.verifyExpectedNodesFailed(failedWorkers, wait)
	if err != nil {
		return fmt.Errorf("[failLabeledNodes] failed to verify expected nodes failed: %v", err)
	}

	return nil
}

func (i *integration) verifyExpectedNodesFailed(failedWorkers []string, wait int) error {
	// Allow a little extra for node failure to be detected than just the node downtime.
	// This proved necessary for the really short failure times (45 sec.) to be reliable.
	wait = wait + wait
	log.Infof("Requested nodes to fail. Waiting up to %d seconds to see if they show up as failed.", wait)
	timeoutDuration := time.Duration(wait) * time.Second
	timeout := time.NewTimer(timeoutDuration)
	ticker := time.NewTicker(checkTickerInterval * time.Second)
	done := make(chan bool)
	start := time.Now()

	log.Infof("Waiting for failed nodes...")

	requestedWorkersAndFailed := func(node corev1.Node) bool {
		found := false
		for _, worker := range failedWorkers {
			if node.Name == worker && i.isNodeFailed(node, "") {
				found = true
				break
			}
		}
		return found
	}

	foundFailedWorkers, err := i.searchForNodes(requestedWorkersAndFailed)
	go func() {
		defer close(done)
		for {
			select {
			case <-timeout.C:
				log.Info("Timed out, but doing last check if requested nodes show up as failed")
				foundFailedWorkers, err = i.searchForNodes(requestedWorkersAndFailed)
				return
			case <-ticker.C:
				log.Infof("Checking if requested nodes show up as failed (time left %v)", timeoutDuration-time.Since(start))
				foundFailedWorkers, err = i.searchForNodes(requestedWorkersAndFailed)
				if len(foundFailedWorkers) == len(failedWorkers) {
					return
				}
			}
		}
	}()

	<-done
	timeout.Stop()
	ticker.Stop()

	log.Infof("Completed checks for failed nodes after %v", time.Since(start))
	err = AssertExpectedAndActual(assert.Equal, true, len(foundFailedWorkers) == len(failedWorkers),
		fmt.Sprintf("Expected %d worker node(s) to be failed, but was %d. %v", len(failedWorkers), len(foundFailedWorkers), foundFailedWorkers))
	if err != nil {
		return err
	}

	return nil
}

func (i *integration) failAndExpectingTaints(numNodes, numPrimary, failure string, wait int, expectedTaints string) error {
	return i.internalFailWorkerAndPrimaryNodes(numNodes, numPrimary, failure, expectedTaints, wait)
}

// internalFailWorkerAndPrimaryNodes will do the work of failing the number of primary and worker nodes in the cluster.
// If expectedTaints is non-empty, then these specific taints will be checked against the node as an indication of
// node failure.
func (i *integration) internalFailWorkerAndPrimaryNodes(numNodes, numPrimary, failure, expectedTaints string, wait int) error {
	if expectedTaints != "" {
		i.customTaints = expectedTaints
	}

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

	// Allow a little extra for node failure to be detected than just the node downtime.
	// This proved necessary for the really short failure times (45 sec.) to be reliable.
	wait = wait + wait
	log.Infof("Requested nodes to fail. Waiting up to %d seconds to see if they show up as failed.", wait)
	timeoutDuration := time.Duration(wait) * time.Second
	timeout := time.NewTimer(timeoutDuration)
	ticker := time.NewTicker(checkTickerInterval * time.Second)
	done := make(chan bool)
	start := time.Now()

	log.Infof("Waiting for failed nodes...")

	requestedWorkersAndFailed := func(node corev1.Node) bool {
		found := false
		for _, worker := range failedWorkers {
			if node.Name == worker && i.isNodeFailed(node, expectedTaints) {
				found = true
				break
			}
		}
		return found
	}

	requestedPrimaryAndFailed := func(node corev1.Node) bool {
		found := false
		for _, primaryNode := range failedPrimary {
			if node.Name == primaryNode && i.isNodeFailed(node, expectedTaints) {
				found = true
				break
			}
		}
		return found
	}

	foundFailedWorkers, err := i.searchForNodes(requestedWorkersAndFailed)
	foundFailedPrimary, err := i.searchForNodes(requestedPrimaryAndFailed)

	go func() {
		defer close(done)
		for {
			select {
			case <-timeout.C:
				log.Info("Timed out, but doing last check if requested nodes show up as failed")
				foundFailedWorkers, err = i.searchForNodes(requestedWorkersAndFailed)
				foundFailedPrimary, err = i.searchForNodes(requestedPrimaryAndFailed)
				return
			case <-ticker.C:
				log.Infof("Checking if requested nodes show up as failed (time left %v)", timeoutDuration-time.Since(start))
				foundFailedWorkers, err = i.searchForNodes(requestedWorkersAndFailed)
				foundFailedPrimary, err = i.searchForNodes(requestedPrimaryAndFailed)
				if len(foundFailedPrimary) == len(failedPrimary) && len(foundFailedWorkers) == len(failedWorkers) {
					return
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

func (i *integration) deployPods(protected bool, podsPerNode, numVols, numDevs, driverType, storageClass string, wait int, preferred string) error {
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

	// Select the deployment script to use based on the driver type.
	var deployScript string
	cleanUpWait := 1 * time.Second
	switch driverType {
	case "vxflexos":
		deployScript = "insv.sh"
	case "unity":
		deployScript = "insu.sh"
		cleanUpWait = 60 * time.Second
	case "isilon":
		deployScript = "insi.sh"
		cleanUpWait = 60 * time.Second
	case "powerstore":
		deployScript = "insps.sh"
		cleanUpWait = 60 * time.Second
	case "powermax":
		deployScript = "inspm.sh"
		cleanUpWait = 60 * time.Second
	}

	// Set test namespace prefix is based on the driver type.
	// If doing an unprotected pod, use a special prefix.
	var prefix string
	if protected {
		switch driverType {
		case "vxflexos":
			i.testNamespacePrefix[PowerflexNS] = true
			prefix = PowerflexNS
		case "unity":
			i.testNamespacePrefix[UnityNS] = true
			prefix = UnityNS
		case "isilon":
			i.testNamespacePrefix[PowerScaleNS] = true
			prefix = PowerScaleNS
		case "powerstore":
			i.testNamespacePrefix[PowerStoreNS] = true
			prefix = PowerStoreNS
		case "powermax":
			i.testNamespacePrefix[PowerMaxNS] = true
			prefix = PowerMaxNS
		}
	} else {
		i.testNamespacePrefix[UnprotectedPodsNS] = true
		prefix = UnprotectedPodsNS
	}

	deployScriptPath := filepath.Join("..", "..", "test", "podmontest", deployScript)
	script := "bash"

	args := []string{
		deployScriptPath,
		"--instances", strconv.Itoa(podCount),
		"--nvolumes", strconv.Itoa(volCount),
		"--ndevices", strconv.Itoa(devCount),
		"--prefix", prefix,
		"--storage-class", storageClass,
	}

	if preferred != "" {
		args = append(args, "--podPreferred", preferred)
	}

	if !protected {
		args = append(args, "--label", "none")
	}

	command := exec.Command(script, args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	// For consecutive run provide Unity array some cleanup times
	time.Sleep(cleanUpWait)
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
		ns := fmt.Sprintf("%s%d", prefix, n)
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

func (i *integration) deployVMs(protected bool, vmsPerNode, numVols, numDevs, driverType, storageClass string, wait int) error {
	podCount, err := i.selectFromRange(vmsPerNode)
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

	// Select the deployment script to use based on the driver type.
	var deployScript string
	cleanUpWait := 1 * time.Second
	switch driverType {
	case "vxflexos":
		deployScript = "insv.sh"
	case "isilon":
		deployScript = "insi.sh"
		cleanUpWait = 60 * time.Second
	case "powerstore":
		deployScript = "insps.sh"
		cleanUpWait = 60 * time.Second
	case "powermax":
		deployScript = "inspm.sh"
		cleanUpWait = 60 * time.Second
	}

	// Set test namespace prefix is based on the driver type.
	// If doing an unprotected vm, use a special prefix.
	var prefix string
	if protected {
		switch driverType {
		case "vxflexos":
			i.testNamespacePrefix[PowerflexNS] = true
			prefix = PowerflexNS
		case "isilon":
			i.testNamespacePrefix[PowerScaleNS] = true
			prefix = PowerScaleNS
		case "powerstore":
			i.testNamespacePrefix[PowerStoreNS] = true
			prefix = PowerStoreNS
		case "powermax":
			i.testNamespacePrefix[PowerMaxNS] = true
			prefix = PowerMaxNS
		}
	} else {
		i.testNamespacePrefix[UnprotectedPodsNS] = true
		prefix = UnprotectedPodsNS
	}

	deployScriptPath := filepath.Join("..", "..", "test", "podmontest", deployScript)
	script := "bash"

	args := []string{
		deployScriptPath,
		"--instances", strconv.Itoa(podCount),
		"--nvolumes", strconv.Itoa(volCount),
		"--ndevices", strconv.Itoa(devCount),
		"--prefix", prefix,
		"--storage-class", storageClass,
		"--workload-type", VM,
	}

	if !protected {
		args = append(args, "--label", "none")
	}

	command := exec.Command(script, args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	// For consecutive run provide Unity array some cleanup times
	time.Sleep(cleanUpWait)
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

	// For the test vms number of namespaces = vmsCount
	for n := 1; n <= podCount; n++ {
		ns := fmt.Sprintf("%s%d", prefix, n)
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

func (i *integration) deployProtectedPods(podsPerNode, numVols, numDevs, driverType, storageClass string, wait int) error {
	return i.deployPods(true, podsPerNode, numVols, numDevs, driverType, storageClass, wait, "")
}

func (i *integration) deployProtectedVMs(vmsPerNode, numVols, numDevs, driverType, storageClass string, wait int) error {
	return i.deployVMs(true, vmsPerNode, numVols, numDevs, driverType, storageClass, wait)
}

func (i *integration) deployUnprotectedPods(podsPerNode, numVols, numDevs, driverType, storageClass string, wait int) error {
	return i.deployPods(false, podsPerNode, numVols, numDevs, driverType, storageClass, wait, "")
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
	var err error
	var foundNamespace bool
	if foundNamespace, err = i.getNamespace(namespace); err != nil {
		return err
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

func (i *integration) removePreferredLabels() error {
	log.Println("Removing preferred labels from nodes")

	// Clean up nodes with the label
	labelKey := "preferred"
	nodes, err := i.k8s.GetClient().CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{
		LabelSelector: labelKey,
	})
	if err != nil {
		return err
	}

	for _, node := range nodes.Items {
		if _, exists := node.Labels[labelKey]; exists {
			delete(node.Labels, labelKey)
			_, err := i.k8s.GetClient().CoreV1().Nodes().Update(context.TODO(), &node, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (i *integration) finallyCleanupEverything() error {
	uninstallScript := "uns.sh"

	defer i.removePreferredLabels()

	if lastTestDriverType == "" {
		// Nothing to clean up
		return nil
	}

	log.Infof("Attempting to clean up everything for driverType '%s'", lastTestDriverType)

	scriptPath := filepath.Join("..", "..", "test", "podmontest", uninstallScript)
	script := "bash"

	for prefix := range i.testNamespacePrefix {
		args := []string{
			scriptPath,
			"--prefix", prefix,
			"--instances", strconv.Itoa(i.podCount),
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

	// If using Openshift, check for Openshift specific env vars
	if i.isOpenshift {
		err = AssertExpectedAndActual(assert.Equal, true, i.bastionNode != "",
			fmt.Sprintf("Expected %s env variable when using an Openshift cluster.\n"+
				"Try export %s=<name/IP of Bastion node> before running tests.", OpenshiftBastion, OpenshiftBastion))
		if err != nil {
			return err
		}
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

	if i.isOpenshift {
		err = i.copyOverTestScriptsToNode(os.Getenv(OpenshiftBastion))
		if err != nil {
			return err
		}
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
		return fmt.Errorf("%s", message)
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

// waitForPodsToSwitchNodes periodically checks labeled pods to see if the node they are
// currently scheduled on is different from the node they were initially scheduled on.
// If the pod(s) has not migrated after 'waitTimeSec' seconds, a non-nil error is returned.
func (i *integration) waitForPodsToSwitchNodes(waitTimeSec int) error {
	timeout, ticker, stop := newTimerWithTicker(waitTimeSec)
	defer stop()

	log.Infof("waiting for %d seconds for pods to switch nodes", waitTimeSec)
	for {
		select {
		case <-timeout.C:
			log.Errorf("timed out after %d seconds while waiting for pods to switch nodes", waitTimeSec)
			return errors.New("timed out waiting for pods to switch nodes")
		case <-ticker.C:
			err := i.labeledPodsChangedNodes()
			if err == nil {
				log.Info("pods successfully changed nodes")
				return nil
			}
			log.Warn("pods have not yet change nodes")
		}
	}
}

// labeledPodsChangedNodes examines the current assignment of labeled pods to nodes and compares it
// with what was populated upon initial deployment in i.labeledPodsToNodes. Expectation is that the
// nodes will have changed (assuming that the failure condition was detected and handled).
func (i *integration) labeledPodsChangedNodes() error {
	return i.arePodsProperlyChanged(func(_ string) bool {
		// Since this step does not care what node it is on and assumes all nodes are valid, just return true.
		// Previous step should have already verified that all nodes are valid and pods are ready.
		return true
	})
}

/* -- Helper functions -- */

func (i *integration) arePodsProperlyChanged(isOnValidNode func(nodeName string) bool) error {
	currentPodToNodeMap := make(map[string]string)
	pods, getPodsErr := i.listPodsByLabel(fmt.Sprintf("podmon.dellemc.com/driver=csi-%s", i.driverType))
	if getPodsErr == nil {
		for _, pod := range pods.Items {
			podName := pod.Name
			if strings.HasPrefix(podName, "virt-launcher") && len(podName) > 6 {
				podName = podName[:len(podName)-6] // Trim suffix
			}
			nsPodName := fmt.Sprintf("%s/%s", pod.Namespace, podName)
			currentPodToNodeMap[nsPodName] = pod.Spec.NodeName
		}
	} else {
		return getPodsErr
	}

	// Search through the labeled pod map and verify node change
	for iPodName, initialNode := range i.labeledPodsToNodes {
		currentNode, ok := currentPodToNodeMap[iPodName]
		if !ok {
			return fmt.Errorf("expected %s pod to be assigned to a node, but no association was found", iPodName)
		}

		if !isOnValidNode(currentNode) {
			return AssertExpectedAndActual(assert.Equal, true, currentNode != initialNode,
				fmt.Sprintf("Expected %s pod to be migrated to a healthy node. Currently '%s', initially '%s'",
					iPodName, currentNode, initialNode))
		}

		if currentNode == initialNode {
			return AssertExpectedAndActual(assert.Equal, false, currentNode == initialNode,
				fmt.Sprintf("Expected %s pod to be migrated to a healthy node. Currently '%s', initially '%s'",
					iPodName, currentNode, initialNode))
		}
	}

	return nil
}

func (i *integration) dumpNodeInfo() error {
	list, err := i.k8s.GetClient().CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		message := fmt.Sprintf("listing nodes error: %s", err)
		return fmt.Errorf("%s", message)
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
		return false, fmt.Errorf("%s", message)
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

// checkIfNodesHaveTaints will iterate through the list of nodes in the cluster
// validating if each node has the expected taints based on the failure.
// That is, if the node failed and it had pods on it, it should expect to see
// the Kubernetes unreachable and the podmon taint. If the node didn't have
// any pods running on it, then it should expect only the unreachable taint.
func (i *integration) checkIfNodesHaveTaints() (bool, error) {
	list, err := i.k8s.GetClient().CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		message := fmt.Sprintf("listing nodes error: %s", err)
		return false, fmt.Errorf("%s", message)
	}

	for _, node := range list.Items {
		for _, taint := range node.Spec.Taints {
			checkTaints := strings.Split(i.getExpectedTaints(node.Name), ",")
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

	minimum, err := strconv.Atoi(components[0])
	if err != nil {
		return 0, err
	}

	maximum, err := strconv.Atoi(components[1])
	if err != nil {
		return 0, err
	}

	if minimum < -1 {
		return -1, fmt.Errorf("invalid range. Minimum is less than 1 (%d)", minimum)
	}

	if maximum < 0 {
		return -1, fmt.Errorf("invalid range. Maximum is less than 0 (%d)", maximum)
	}

	if minimum > maximum {
		return -1, fmt.Errorf("invalid range. Minimum value specified is greater than max (%d > %d)", minimum, maximum)
	}

	// rand.IntnRange selects from min to max, inclusive of min and exclusive of
	// max, so use max+1 in order for max value to be a possible value.
	selected := rand.IntnRange(minimum, maximum+1)
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

	// Create a mapping of the node name to IP address
	nameToIP := make(map[string]string)
	for _, node := range nodes {
		for _, addr := range node.Status.Addresses {
			if addr.Type == "InternalIP" {
				nameToIP[node.Name] = addr.Address
				break
			}
		}
	}

	// Create a list of candidates. Prepend with the nodes that have labeled pods on them.
	// This way, we will always have a chance to test fail over scenario.
	candidates := make([]string, 0)
	tracker := make(map[string]bool) // Used to prevent duplicates in 'candidate' list.
	for _, name := range i.labeledPodsToNodes {
		if !tracker[name] {
			tracker[name] = true
			candidates = append(candidates, name)
		}
	}

	// Check if we have enough candidates to match the requested number to fail.
	if len(candidates) < numberToFail {
		// Need to add more to list of candidates, so add the remaining node names to the candidate list.
		for name := range nameToIP {
			if !tracker[name] {
				tracker[name] = true
				candidates = append(candidates, name)
			}
		}
	}

	if len(i.preferredLabeledNodes) > 0 {
		// Add the preferred labeled nodes to the candidate list for the failure
		candidates = make([]string, 0)
		tracker = make(map[string]bool)
		for _, name := range i.preferredLabeledNodes {
			if !tracker[name] {
				tracker[name] = true
				candidates = append(candidates, name)
			}
		}
	}
	log.Infof("All the candidate nodes to fail are %v", candidates)

	failed := 0
	for _, name := range candidates {
		// For CSI driver pod run the script from test host not worker node
		if failureType == "driverpod" {
			var cmd *exec.Cmd
			cmd = exec.Command( // #nosec G204
				"/bin/sh", fmt.Sprintf("%s/failpods.sh", i.scriptsDir),
				"--ns", i.driverType,
				"--timeoutseconds", fmt.Sprintf("%d", wait),
			)
			out, err := cmd.CombinedOutput()
			log.Infof("Driver node pod test executed %s", out)
			if err != nil {
				log.Infof("Failing err %v %s", err, out)
				return failedNodes, err
			}
			return failedNodes, nil
		}
		if failed < numberToFail {
			ip := nameToIP[name]
			if err = i.induceFailureOn(name, ip, failureType, wait); err != nil {
				return failedNodes, err
			}
			log.Infof("Failing %s %s", name, ip)
			failedNodes = append(failedNodes, name)
			failed++
		}
	}

	for _, name := range failedNodes {
		i.nodesToTaints[name] = i.getExpectedTaints(name)
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
	if i.isOpenshift {
		return i.copyOverTestScriptsToOpenshift(address)
	}
	return i.copyOverTestScriptsToNode(address)
}

// copyOverTestScriptsNode copies over the scripts to the node at 'address'.
// This will use internal SSH library to do the set up and copying to the
// specified node.
func (i *integration) copyOverTestScriptsToNode(address string) error {
	ctx := context.Background()
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
		Timeout:    sshTimeoutDuration,
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

	files, err := os.ReadDir(i.scriptsDir)
	if err != nil {
		return err
	}

	for _, f := range files {
		err = client.Copy(ctx, fmt.Sprintf("%s/%s", i.scriptsDir, f.Name()), fmt.Sprintf("%s/%s", remoteScriptDir, f.Name()))
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

// copyOverTestScriptsToOpenshift will copy over script files onto the Openshift node at 'address'.
// This depends on the Bastion node having the script files copied over already. Then for each
// script file, we will use the 'scp' command from the Bastion node to copy over the files from
// there to the Openshift node.
func (i *integration) copyOverTestScriptsToOpenshift(address string) error {
	// SSH and SCP are all done through the Bastion node
	info := ssh.AccessInfo{
		Hostname: i.bastionNode,
		Port:     "22",
		Username: os.Getenv("NODE_USER"),
		Password: os.Getenv("PASSWORD"),
	}

	wrapper := ssh.NewWrapper(&info)

	client := ssh.CommandExecution{
		AccessInfo: &info,
		SSHWrapper: wrapper,
		Timeout:    sshTimeoutDuration,
	}

	log.Infof("Attempting to scp scripts from Bastion node %s to %s:%s", i.bastionNode, address, openShiftRemoteScriptDir)
	mkDirCmd := fmt.Sprintf("ssh %s core@%s 'date; rm -rf %s; mkdir -p %s'", sshOptions, address, openShiftRemoteScriptDir, openShiftRemoteScriptDir)
	log.Info(mkDirCmd)
	if mkDirErr := client.Run(mkDirCmd); mkDirErr == nil {
		for _, out := range client.GetOutput() {
			log.Infof("%s", out)
		}
	} else {
		return mkDirErr
	}

	// Use SCP to copy the script files on the Bastion node into the /tmp dir of the Openshift node.
	copyFileCmd := fmt.Sprintf("scp -r %s %s core@%s:%s", sshOptions, remoteScriptDir, address, "/usr/tmp")
	log.Info(copyFileCmd)
	err := client.Run(copyFileCmd)
	if err != nil {
		return err
	}

	// After copying the files, add execute permissions and list the directory
	lsDirCmd := fmt.Sprintf("ssh %s core@%s 'sudo chmod +x %s/* ; sudo ls -ltr %s'", sshOptions, address, openShiftRemoteScriptDir, openShiftRemoteScriptDir)
	log.Info(lsDirCmd)
	if lsErr := client.Run(lsDirCmd); lsErr == nil {
		for _, out := range client.GetOutput() {
			log.Infof("%s", out)
		}
	} else {
		return lsErr
	}

	log.Infof("Scripts successfully copied to %s:%s", address, openShiftRemoteScriptDir)
	return nil
}

func (i *integration) allPodsInTestNamespacesAreRunning() (bool, error) {
	allRunning := true
	for podIdx := 1; podIdx <= i.podCount; podIdx++ {
		for prefix := range i.testNamespacePrefix {
			namespace := fmt.Sprintf("%s%d", prefix, podIdx)
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
	}
	return allRunning, nil
}

func (i *integration) initialDiskWriteAndVerifyAllVMs() error {
	log.Infof("Waiting 60 seconds to ensure VMs are fully running before initial disk write...")
	time.Sleep(60 * time.Second)
	for vmIdx := 1; vmIdx <= i.podCount; vmIdx++ {
		for prefix := range i.testNamespacePrefix {
			log.Infof("Verifying disk on VM %d in namespace %s", vmIdx, prefix)

			ns := fmt.Sprintf("%s%d", prefix, vmIdx)
			vmName := "vm-0"
			err := i.writeAndVerifyDiskOnVM(vmName, ns)
			if err != nil {
				return fmt.Errorf("Disk Verification Failed: %v", err)
			}
		}
	}
	return nil
}

func (i *integration) postFailoverVerifyAllVMs() error {
	log.Infof("Waiting 60 seconds to ensure VMs are fully running after failover...")
	time.Sleep(60 * time.Second)
	for vmIdx := 1; vmIdx <= i.podCount; vmIdx++ {
		for prefix := range i.testNamespacePrefix {
			ns := fmt.Sprintf("%s%d", prefix, vmIdx)
			vmName := "vm-0"
			err := i.verifyDiskContentOnVM(vmName, ns)
			if err != nil {
				return fmt.Errorf("Data Verification Failed: %v", err)
			}
		}
	}
	return nil
}

const (
	expectedData = "Test awesome shareable disks"
)

func (i *integration) writeAndVerifyDiskOnVM(vmName, namespace string) error {
	writeCmd := fmt.Sprintf(
		"sshpass -p 'fedora' virtctl ssh %s --namespace=%s --username=fedora "+
			"--local-ssh=true --local-ssh-opts='-o StrictHostKeyChecking=no' --local-ssh-opts='-o UserKnownHostsFile=/dev/null' "+
			"--command \"printf '%s' | sudo dd of=/dev/vdc bs=1 count=150 conv=notrunc\"",
		vmName, namespace, expectedData)
	log.Printf("Running: %s", writeCmd)
	writeOut, err := exec.Command("bash", "-c", writeCmd).CombinedOutput()
	if err != nil {
		log.Printf("Write failed on %s: %s", vmName, string(writeOut))
		return err
	}
	log.Printf("Write output for %s: %s", vmName, string(writeOut))

	// Read and verify
	return i.verifyDiskContentOnVM(vmName, namespace)
}

func (i *integration) verifyDiskContentOnVM(vmName, namespace string) error {
	readCmd := fmt.Sprintf(
		"sshpass -p 'fedora' virtctl ssh %s --namespace=%s --username=fedora "+
			"--local-ssh=true --local-ssh-opts='-o StrictHostKeyChecking=no'  --local-ssh-opts='-o UserKnownHostsFile=/dev/null' "+
			"--command \"sudo dd if=/dev/vdc bs=1 count=150\"",
		vmName, namespace)
	log.Printf("Running: %s", readCmd)
	readOut, err := exec.Command("bash", "-c", readCmd).CombinedOutput()
	if err != nil {
		log.Printf("Read failed on %s: %s", vmName, string(readOut))
		return err
	}

	log.Printf("Read output for %s: %s", vmName, string(readOut))

	if strings.Contains(string(readOut), expectedData) {
		log.Printf("Disk content verified for %s", vmName)
		return nil
	}
	log.Printf("Expected content not found in %s", vmName)
	return nil
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
	if i.isOpenshift {
		// On Openshift, failure scripts are invoked via the Bastion node
		info.Hostname = i.bastionNode
	}
	wrapper := ssh.NewWrapper(&info)

	client := ssh.CommandExecution{
		AccessInfo: &info,
		SSHWrapper: wrapper,
		Timeout:    sshTimeoutDuration,
	}

	// Split the failureType by ':' character. If specified, the first part is the key,
	// the second parts are some extra parameters.
	failureTypeSplit := strings.Split(failureType, ":")
	failureType = failureTypeSplit[0]
	hasOptions := len(failureTypeSplit) > 1

	log.Infof("Attempting to induce the %s failure on %s/%s for %d seconds", failureType, name, ip, wait)
	scriptToUse, ok := failureToScriptMap[failureType]
	if !ok {
		return fmt.Errorf("no mapping for failureType %s", failureType)
	}

	dirToUse := remoteScriptDir
	if i.isOpenshift {
		dirToUse = openShiftRemoteScriptDir
	}

	// Invoke script allows us to programmatically invoke the failure script and not fail the SSH session
	invokerScript := fmt.Sprintf("%s/invoke.sh", dirToUse)
	failureScript := fmt.Sprintf("%s/%s", dirToUse, scriptToUse)
	invokeFailCmd := fmt.Sprintf("%s %s --seconds %d", invokerScript, failureScript, wait)

	if (failureType == "interfacedown" || failureType == "reboot") && hasOptions {
		// If there are options specified for these tests, then use those as specific interface names
		interfaceEnvVarName := failureTypeSplit[1]
		specificInterfaces := os.Getenv(interfaceEnvVarName)
		if specificInterfaces == "" {
			return fmt.Errorf("test case %s failure type is expecting a %s environmental variable, but it does not exist", failureType, interfaceEnvVarName)
		}
		log.Infof("Specific interfaces '%s' will be affected", specificInterfaces)
		invokeFailCmd = fmt.Sprintf("%s %s --seconds %d --interfaces %s", invokerScript, failureScript, wait, specificInterfaces)
	}

	if failureType == "driverpod" {
		invokeFailCmd = fmt.Sprintf("%s %s --ns %s --timeoutseconds %d", invokerScript, failureScript, i.driverType, wait)
	}
	if i.isOpenshift {
		// For Openshift, failure script invocation is done from the Bastion node to the Openshift node
		invokeFailCmd = fmt.Sprintf("ssh %s core@%s sudo %s", sshOptions, ip, invokeFailCmd)
	}
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
			if expectedTaints, ok := i.nodesToTaints[node.Name]; ok {
				log.Infof("k8sPoll: ^^^^^^^^^^^ is a failed node. Expecting taints: %s", expectedTaints)
			}
		}
	} else {
		log.Infof("k8sPoll: listing nodes error: %s", getNodesErr)
	}

	if i.driverType != "" {
		pods, getPodsErr := i.listPodsByLabel(fmt.Sprintf("podmon.dellemc.com/driver=csi-%s", i.driverType))
		if getPodsErr == nil {
			for _, pod := range pods.Items {
				nodeSpec := pod.Spec.NodeName
				// Display the initial and the current nodes (if changed)
				nsPodName := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)
				if initialNode, ok := i.labeledPodsToNodes[nsPodName]; ok && initialNode != pod.Spec.NodeName {
					nodeSpec = fmt.Sprintf("%s --> %s", initialNode, pod.Spec.NodeName)
				}
				log.Infof("k8sPoll: %s [ PROTECTED   ] %s/%s %s", nodeSpec, pod.Namespace, pod.Name, pod.Status.Phase)
			}
		} else {
			log.Infof("k8sPoll: get pods failed: %s", getPodsErr)
		}
		// List unprotected pods
		pods, getPodsErr = i.listPodsByLabel("podmon.dellemc.com/driver=none")
		if getPodsErr == nil {
			for _, pod := range pods.Items {
				log.Infof("k8sPoll: %s [ UNPROTECTED ] %s/%s %s", pod.Spec.NodeName, pod.Namespace, pod.Name, pod.Status.Phase)
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
		podName := pod.Name
		if strings.HasPrefix(podName, "virt-launcher") && len(podName) > 6 {
			podName = podName[:len(podName)-6] // Trim last 6 characters
		}
		nsPodName := fmt.Sprintf("%s/%s", pod.Namespace, podName)

		i.labeledPodsToNodes[nsPodName] = pod.Spec.NodeName
	}
	return nil
}

func (i *integration) isNodeFailed(node corev1.Node, expectingTheseTaints string) bool {
	isFailed := false
	nodeIsNotReady := !nodeHasCondition(node, "Ready")
	if expectingTheseTaints != "" {
		// Only check if these specific taints are showing up as an indication of node failure
		taintCount := 0
		expected := strings.Split(expectingTheseTaints, ",")
		for _, taint := range expected {
			if nodeHasTaint(&node, taint, corev1.TaintEffectNoSchedule) {
				taintCount++
			}
		}
		// Fail only if all the expected taints show up
		isFailed = taintCount == len(expected)
	} else if i.nodeHadPodsRunning(node.Name) {
		podmonTaint := fmt.Sprintf("%s.%s", lastTestDriverType, PodmonTaintKeySuffix)
		hasTaint := nodeHasTaint(&node, podmonTaint, corev1.TaintEffectNoSchedule)
		isFailed = nodeIsNotReady && hasTaint
	} else {
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
	hasTaints, err := i.checkIfNodesHaveTaints()
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
				hasTaints, err = i.checkIfNodesHaveTaints()
				done <- true
			case <-ticker.C:
				log.Infof("Checking if podmon taints have been removed (time left %v)", timeoutDuration-time.Since(start))
				hasTaints, err = i.checkIfNodesHaveTaints()
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
		fmt.Sprintf("Expected taints to be removed after %d seconds, but still exist", wait))
}

// getExpectedTaints returns a comma delimited list of taints for the node with 'nodeName'
// supposing that it had been failed by the test. Note, this does not necessarily mean
// that the node was actually failed by the test.
func (i *integration) getExpectedTaints(nodeName string) string {
	// If the test requires a specific set of taints, return that
	if i.customTaints != "" {
		return i.customTaints
	}
	// Should minimally expect to the the Kubernetes unreachable taint on the failed node
	theseTaints := "node.kubernetes.io/unreachable,offline.vxflexos.storage.dell.com,offline.unity.storage.dell.com,offline.isilon.storage.dell.com,offline.powerstore.storage.dell.com,offline.powermax.storage.dell.com"
	if i.nodeHadPodsRunning(nodeName) {
		// If the test failed some node(s) that had labeled pods in it, then we
		// expect the podmon taint to be cleaned up as well.
		theseTaints = theseTaints + "," + fmt.Sprintf("%s.%s", lastTestDriverType, PodmonTaintKeySuffix)
	}
	return theseTaints
}

func (i *integration) nodeHadPodsRunning(nodeName string) bool {
	for _, failedNodeName := range i.labeledPodsToNodes {
		if failedNodeName == nodeName {
			return true
		}
	}
	return false
}

func (i *integration) getNamespace(namespace string) (bool, error) {
	namespaces, err := i.k8s.GetClient().CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return false, err
	}

	for _, ns := range namespaces.Items {
		if namespace == ns.Name {
			return true, nil
		}
	}

	return false, nil
}

func (i *integration) detectOpenshift() (bool, error) {
	var err error
	var hasOpenshiftNS bool
	if hasOpenshiftNS, err = i.getNamespace("openshift"); err != nil {
		return false, err
	}

	return hasOpenshiftNS, nil
}

func (i *integration) iFailDriverPodsTaints(numNodes, failure string, wait int, expectedTaints string) error {
	if expectedTaints != "" {
		i.customTaints = expectedTaints
	}

	i.scriptsDir = os.Getenv("SCRIPTS_DIR")

	workersToFail, err := i.parseRatioOrCount(numNodes)
	if err != nil {
		return err
	}

	log.Infof("Test with %2.2f failed workers nodes", workersToFail)

	failedWorkers, err := i.failWorkerNodes(workersToFail, failure, wait)
	if err != nil {
		return err
	}

	// Allow a little extra for node failure to be detected than just the node down time.
	// This proved necessary for the really short failure times (45 sec.) to be reliable.
	wait = wait + wait
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
			if node.Name == worker && i.isNodeFailed(node, expectedTaints) {
				found = true
				break
			}
		}

		return found
	}

	foundFailedWorkers, err := i.searchForNodes(requestedWorkersAndFailed)

	go func() {
		for {
			select {
			case <-timeout.C:
				log.Info("Timed out, but doing last check if requested nodes show up as failed")
				foundFailedWorkers, err = i.searchForNodes(requestedWorkersAndFailed)
				done <- true
			case <-ticker.C:
				log.Infof("Checking if requested nodes show up as failed (time left %v)", timeoutDuration-time.Since(start))
				foundFailedWorkers, err = i.searchForNodes(requestedWorkersAndFailed)
				if len(foundFailedWorkers) == len(failedWorkers) {
					done <- true
				}
			}
		}
	}()

	<-done
	timeout.Stop()
	ticker.Stop()

	log.Infof("Completed checks for failed nodes after %v", time.Since(start))

	err = AssertExpectedAndActual(assert.Equal, true, len(foundFailedWorkers) == len(failedWorkers),
		fmt.Sprintf("Expected %d worker node(s) to be failed, but was %d. %v", len(failedWorkers), len(foundFailedWorkers), foundFailedWorkers))
	if err != nil {
		return err
	}
	return nil
}

func (i *integration) verifyKubeVirtIPAMControllerPodExists() error {
	namespace := "openshift-cnv"
	podPrefix := "kubevirt-ipam-controller-manager-"

	podList, err := i.k8s.GetClient().CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list pods in namespace '%s': %v", namespace, err)
	}

	for _, pod := range podList.Items {
		if strings.HasPrefix(pod.Name, podPrefix) {
			log.Infof("Found pod '%s' in namespace '%s'", pod.Name, pod.Namespace)
			return nil
		}
	}

	return fmt.Errorf("No pod with prefix '%s' found in namespace '%s'. OpenShift Virtualization might not be installed", podPrefix, namespace)
}

func (i *integration) createNodeNameMap(numNodes string, filter func(node corev1.Node) bool) (map[string]string, int, error) {
	count, err := i.parseRatioOrCount(numNodes)
	if err != nil {
		return nil, 0, err
	}

	nodes, err := i.searchForNodes(filter)
	if err != nil {
		return nil, 0, err
	}

	numberToLabel := 0
	if count < 0.0 {
		numberToLabel = len(nodes)
	} else if count < 1.0 {
		temp := float64(len(nodes))
		numberToLabel = int(math.Ceil(temp * count))
	} else { // count >= 1.0, so use the value
		numberToLabel = int(count)
	}

	// Create a mapping of the node name to IP address
	nameToIP := make(map[string]string)
	for _, node := range nodes {
		for _, addr := range node.Status.Addresses {
			if addr.Type == "InternalIP" {
				nameToIP[node.Name] = addr.Address
				break
			}
		}
	}

	return nameToIP, numberToLabel, nil
}

func (i *integration) labelNodeAsPreferredSite(numNodes, preferred string) error {
	nameToIP, numberToLabel, err := i.createNodeNameMap(numNodes, isWorkerNode)
	if err != nil {
		return err
	}

	// Create a list of candidates. Prepend with the nodes that have labeled pods on them.
	// This way, we will always have a chance to test fail over scenario.
	candidates := make([]string, 0)
	tracker := make(map[string]bool) // Used to prevent duplicates in 'candidate' list.
	for _, name := range i.labeledPodsToNodes {
		if !tracker[name] {
			if len(candidates) >= numberToLabel {
				break
			}
			tracker[name] = true
			candidates = append(candidates, name)
		}
	}

	// Check if we have enough candidates to match the requested number to fail.
	if len(candidates) < numberToLabel {
		// Need to add more to list of candidates, so add the remaining node names to the candidate list.
		for name := range nameToIP {
			if !tracker[name] {
				if len(candidates) >= numberToLabel {
					break
				}
				tracker[name] = true
				candidates = append(candidates, name)
			}
		}
	}

	i.preferredLabeledNodes = []string{}
	labeled := 0
	for _, name := range candidates {
		if labeled < numberToLabel {
			log.Infof("Labeling node %s as %s", name, preferred)

			nodeObj, err := i.k8s.GetClient().CoreV1().Nodes().Get(context.TODO(), name, metav1.GetOptions{})
			if err != nil {
				return fmt.Errorf("Failed to get node '%s': %v", name, err)
			}

			// Add or update the label
			if nodeObj.ObjectMeta.Labels == nil {
				nodeObj.ObjectMeta.Labels = make(map[string]string)
			}
			nodeObj.ObjectMeta.Labels["preferred"] = preferred
			i.preferredLabeledNodes = append(i.preferredLabeledNodes, name)

			// Update the node
			_, err = i.k8s.GetClient().CoreV1().Nodes().Update(context.TODO(), nodeObj, metav1.UpdateOptions{})
			if err != nil {
				return fmt.Errorf("Failed to label node '%s': %v", name, err)
			}
			labeled++
		}
	}
	return nil
}

func (i *integration) deployProtectedPreferredPods(podsPerNode, numVols, numDevs, driverType, storageClass string, wait int, preferred string) error {
	return i.deployPods(true, podsPerNode, numVols, numDevs, driverType, storageClass, wait, preferred)
}

func (i *integration) allPodsOnNodesWithPreferredLabel(preferred string) error {
	for podIdx := 1; podIdx <= i.podCount; podIdx++ {
		for prefix := range i.testNamespacePrefix {
			namespace := fmt.Sprintf("%s%d", prefix, podIdx)
			pods, err := i.k8s.GetClient().CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
			if err != nil {
				return err
			}

			for _, pod := range pods.Items {
				nodeObj, err := i.k8s.GetClient().CoreV1().Nodes().Get(context.TODO(), pod.Spec.NodeName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				if nodeObj.ObjectMeta.Labels["preferred"] != preferred {
					return fmt.Errorf("Pod '%s' is not on preferred node '%s'", pod.Name, pod.Spec.NodeName)
				}
			}
		}
	}

	return nil
}

func (i *integration) verifyPodsOnNonPreferredNodes() error {
	for count := 1; count <= i.podCount; count++ {
		namespace := fmt.Sprintf("%s%d", PowerStoreNS, count)
		podList, err := i.k8s.GetClient().CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			log.Errorf("Failed to list pods in namespace '%s': %v", namespace, err)
			return err
		}
		log.Infof("Pods in namespace '%s': %v", namespace, podList.Items)

		for _, pod := range podList.Items {
			nodeName := pod.Spec.NodeName
			log.Infof("Checking pod %s on node %s", pod.Name, nodeName)
			for _, labeledNode := range i.preferredLabeledNodes {
				if nodeName == labeledNode {
					return fmt.Errorf("Pod '%s' is on preferred node '%s'", pod.Name, nodeName)
				}
			}
		}
	}
	return nil
}

func (i *integration) thereAreAtLeastWorkerNodesWhichAreReady(count int) error {
	list, err := i.k8s.GetClient().CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		message := fmt.Sprintf("listing nodes error: %s", err)
		return fmt.Errorf("%s", message)
	}

	workerNodesCount := 0
	for _, node := range list.Items {
		isControlPlane := false
		for label := range node.Labels {
			if label == "node-role.kubernetes.io/control-plane" {
				log.Infof("Node %s is a control plane node", node.Name)
				isControlPlane = true
				break
			}
		}

		if isControlPlane {
			continue
		}

		nodeReady := nodeHasCondition(node, "Ready")
		if nodeReady {
			workerNodesCount++
		}
	}

	if workerNodesCount < count {
		log.Warnln("Skipping this scenario. Expected at least", count, "but found", workerNodesCount)
		return godog.ErrSkip
	}

	return nil
}

func (i *integration) iFailNodesWithLabelWithFailureForSeconds(numNodes, label, failure string, wait int) error {
	filter := func(node corev1.Node) bool {
		if isWorkerNode(node) && node.ObjectMeta.Labels["preferred"] == label {
			return true
		}

		return false
	}

	nameToIP, numberToLabel, err := i.createNodeNameMap(numNodes, filter)
	if err != nil {
		return err
	}

	log.Printf("Labeling %d nodes with preferred=%s", numberToLabel, label)

	// Get application pods that were deployed by podmontest.
	nodeToFail := ""
	for podIdx := 1; podIdx <= i.podCount; podIdx++ {
		for prefix := range i.testNamespacePrefix {
			namespace := fmt.Sprintf("%s%d", prefix, podIdx)
			podList, err := i.k8s.GetClient().CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				return fmt.Errorf("failed to list pods in namespace '%s': %v", namespace, err)
			}

			if len(podList.Items) == 0 {
				return fmt.Errorf("no pods found in namespace '%s'", namespace)
			}

			for _, pod := range podList.Items {
				if _, ok := nameToIP[pod.Spec.NodeName]; ok {
					nodeToFail = pod.Spec.NodeName
					break
				}
			}
		}
	}

	log.Info("Node to fail: ", nodeToFail)

	failedWorkers, err := i.failNodes(func(node corev1.Node) bool {
		return node.Name == nodeToFail
	}, -1, failure, wait)
	if err != nil {
		return err
	}

	err = i.verifyExpectedNodesFailed(failedWorkers, wait)
	if err != nil {
		return fmt.Errorf("[iFailNodesWithLabelWithFailureForSeconds] failed to verify expected nodes failed: %v", err)
	}

	return nil
}

func (i *integration) labeledPodsAreOnANode(label string) error {
	labelKey := "preferred=" + label
	nodes, err := i.k8s.GetClient().CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{
		LabelSelector: labelKey,
	})
	if err != nil {
		return err
	}

	// Creates preferredNodeMap for quick search
	preferredNodeMap := make(map[string]bool)
	for _, node := range nodes.Items {
		preferredNodeMap[node.Name] = true
	}

	return i.arePodsProperlyChanged(func(nodeName string) bool {
		// Ensure that the node that the pod is running on is within the preferred nodes.
		_, ok := preferredNodeMap[nodeName]
		return ok
	})
}

// newTimerWithTicker takes a wait time in seconds and returns a timer, a ticker and a stop function.
// These can be used to periodically execute an action over a set period of time.
// Users should call the stop() function as a best practice.
func newTimerWithTicker(waitTimeSec int) (timeout *time.Timer, ticker *time.Ticker, stop func()) {
	timeoutDuration := time.Duration(waitTimeSec) * time.Second
	timeout = time.NewTimer(timeoutDuration)
	ticker = time.NewTicker(checkTickerInterval * time.Second)

	stop = func() {
		timeout.Stop()
		ticker.Stop()
	}

	return
}

func IntegrationTestScenarioInit(context *godog.ScenarioContext) {
	i := &integration{}
	pollK8sEnabled := false
	if pollK8sStr := os.Getenv("POLL_K8S"); strings.ToLower(pollK8sStr) == "true" {
		pollK8sEnabled = true
	}
	context.BeforeScenario(func(_ *godog.Scenario) {
		if pollK8sEnabled {
			pollTick = time.NewTicker(k8sPollInterval)
			go i.startK8sPoller()
		}
	})
	context.AfterScenario(func(_ *godog.Scenario, _ error) {
		if pollK8sEnabled {
			pollTick.Stop()
		}
	})
	context.Step(`^a kubernetes "([^"]*)"$`, i.givenKubernetes)
	context.Step(`^validate that all pods are running within (\d+) seconds$`, i.allPodsAreRunningWithinSeconds)
	context.Step(`^I fail "([^"]*)" worker nodes and "([^"]*)" primary nodes with "([^"]*)" failure for (\d+) seconds$`, i.failWorkerAndPrimaryNodes)
	context.Step(`^I fail "([^"]*)" worker nodes and "([^"]*)" primary nodes with "([^"]*)" failure for (\d+) and I expect these taints "([^"]*)"$`, i.failAndExpectingTaints)
	context.Step(`I fail labeled "([^"]*)" nodes with "([^"]*)" failure for (\d+) seconds`, i.failLabeledNodes)
	context.Step(`^"([^"]*)" pods per node with "([^"]*)" volumes and "([^"]*)" devices using "([^"]*)" and "([^"]*)" in (\d+)$`, i.deployProtectedPods)
	context.Step(`^"([^"]*)" vms per node with "([^"]*)" volumes and "([^"]*)" devices using "([^"]*)" and "([^"]*)" in (\d+)$`, i.deployProtectedVMs)
	context.Step(`^"([^"]*)" unprotected pods per node with "([^"]*)" volumes and "([^"]*)" devices using "([^"]*)" and "([^"]*)" in (\d+)$`, i.deployUnprotectedPods)
	context.Step(`^the taints for the failed nodes are removed within (\d+) seconds$`, i.theTaintsForTheFailedNodesAreRemovedWithinSeconds)
	context.Step(`^these CSI driver "([^"]*)" are configured on the system$`, i.theseCSIDriverAreConfiguredOnTheSystem)
	context.Step(`^there is a "([^"]*)" in the cluster$`, i.thereIsThisNamespaceInTheCluster)
	context.Step(`^there are driver pods in "([^"]*)" with this "([^"]*)" prefix$`, i.thereAreDriverPodsWithThisPrefix)
	context.Step(`^Check OpenShift Virtualization is installed in the cluster$`, i.verifyKubeVirtIPAMControllerPodExists)
	context.Step(`^finally cleanup everything$`, i.finallyCleanupEverything)
	context.Step(`^cluster is clean of test pods$`, i.finallyCleanupEverything)
	context.Step(`^cluster is clean of test vms$`, i.finallyCleanupEverything)
	context.Step(`^test environmental variables are set$`, i.expectedEnvVariablesAreSet)
	context.Step(`^can logon to nodes and drop test scripts$`, i.canLogonToNodesAndDropTestScripts)
	context.Step(`^these storageClasses "([^"]*)" exist in the cluster$`, i.theseStorageClassesExistInTheCluster)
	context.Step(`^wait (\d+) to see there are no taints$`, i.theTaintsForTheFailedNodesAreRemovedWithinSeconds)
	context.Step(`^labeled pods are on a different node$`, i.labeledPodsChangedNodes)
	context.Step(`^I fail "([^"]*)" worker driver pod with "([^"]*)" failure for (\d+) and I expect these taints "([^"]*)"$`, i.iFailDriverPodsTaints)
	context.Step(`^initial disk write and verify on all VMs succeeds$`, i.initialDiskWriteAndVerifyAllVMs)
	context.Step(`^post failover disk content verification on all VMs succeeds$`, i.postFailoverVerifyAllVMs)
	context.Step(`^label "([^"]*)" node as "([^"]*)" site$`, i.labelNodeAsPreferredSite)
	context.Step(`^"([^"]*)" pods per node with "([^"]*)" volumes and "([^"]*)" devices using "([^"]*)" and "([^"]*)" in (\d+) with "([^"]*)" affinity$`, i.deployProtectedPreferredPods)
	context.Step(`^pods are scheduled on the non preferred nodes$`, i.verifyPodsOnNonPreferredNodes)
	context.Step(`^all pods are running on "([^"]*)" node$`, i.allPodsOnNodesWithPreferredLabel)
	context.Step(`^there are at least (\d+) worker nodes which are ready$`, i.thereAreAtLeastWorkerNodesWhichAreReady)
	context.Step(`^I fail "([^"]*)" nodes with label "([^"]*)" with "([^"]*)" failure for (\d+) seconds$`, i.iFailNodesWithLabelWithFailureForSeconds)
	context.Step(`^labeled pods are on a "([^"]*)" node$`, i.labeledPodsAreOnANode)
	context.Step(`wait at least (\d+) seconds for pods to switch nodes$`, i.waitForPodsToSwitchNodes)
}
