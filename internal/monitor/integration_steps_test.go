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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"podmon/internal/k8sapi"
	"podmon/test/ssh"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dell/csi-powerstore/v2/core"
	pstoreArray "github.com/dell/csi-powerstore/v2/pkg/array"
	pstoreController "github.com/dell/csi-powerstore/v2/pkg/controller"
	pstoreID "github.com/dell/csi-powerstore/v2/pkg/identifiers"
	"github.com/dell/gopowerstore"
	"github.com/cucumber/godog"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
)

type customResource struct {
	APIVersion string        `json:"apiVersion"`
	Items      []interface{} `json:"items"`
	Kind       string        `json:"kind"`
}

type integration struct {
	configPath          string
	k8s                 k8sapi.K8sAPI
	driverType          string
	podCount            int
	devCount            int
	volCount            int
	storageClass        *storagev1.StorageClass
	driverNamespaceName string
	driverSecretName    string
	testNamespacePrefix map[string]bool
	scriptsDir          string
	// map using pod names as keys to the node name on which they are scheduled
	labeledPodsToNodes     map[string]string
	nodesToTaints          map[string]string
	isOpenshift            bool
	bastionNode            string
	customTaints           string
	preferredLabeledNodes  []string
	shouldNotFailNode      string
	isLabelCleanupRequired bool
	metroVolInfo           map[string]volumeInformation
}

// Used for determining whether to disable or re-enable network connection
// between a Kubernetes worker node and a PowerStore metro server.
//
//	MetroConnectionRestore: re-enable the metro connection
//	MetroConnectionFail: disable the metro connection
type MetroConnection string

const (
	MetroConnectionRestore MetroConnection = "ALLOW"
	MetroConnectionFail    MetroConnection = "BLOCK"
)

type volumeInformation struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	ReplicationSessions []struct {
		ID    string `json:"id"`
		State string `json:"state"`
	} `json:"replication_sessions"`
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
	preferredLabelKey   = "topology.kubernetes.io/zone"
	// The name of the PowerStore secret as queried by Kubernetes
	powerstoreSecretName = "powerstore-config"
	// The name of the parent key under which the array config is
	// listed in the powerstore secret
	powerstoreSecretDataKeyName = "config"
	blockTrafficScriptName      = "block-traffic.sh"
	customResourceDR            = "/apis/dr.storage.dell.com/v1"
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

	i.isLabelCleanupRequired = true

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

func (i *integration) allPodsAreNotRunningWithinSeconds(wait int) error {
	// Check each of the test namespaces for running pods
	allRunning, err := i.allPodsInTestNamespacesAreRunning()
	if err != nil {
		return err
	}

	if allRunning {
		return fmt.Errorf("All test pods are in the 'Running' state")
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

	return AssertExpectedAndActual(assert.Equal, false, allRunning,
		fmt.Sprintf("Expected all pods to be not in running state after %d seconds", wait))
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
		return node.Labels[preferredLabelKey] == preferred
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

// failNonpreferredNodesWithFailureForSeconds fails non-preferred nodes with a specified failure for a given number of seconds.
func (i *integration) failNonpreferredNodesWithFailureForSeconds(preferred string, failure string, wait int) error {
	failedWorkers, err := i.failNodes(func(node corev1.Node) bool {
		// check for only worker nodes
		if isPrimaryNode(node) {
			return false
		}

		// Check if the node's label indicates it's not a preferred site
		val, ok := node.Labels[preferredLabelKey]
		if !ok || val != preferred {
			return true
		}
		return false
	}, -1, failure, wait)
	if err != nil {
		return err
	}

	err = i.verifyExpectedNodesFailed(failedWorkers, wait)
	if err != nil {
		return fmt.Errorf("[failNonpreferredNodesWithFailureForSeconds] failed to verify expected nodes failed: %v", err)
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
	if err != nil {
		return err
	}

	foundFailedPrimary, err := i.searchForNodes(requestedPrimaryAndFailed)
	if err != nil {
		return err
	}

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

	i.storageClass, err = i.k8s.GetClient().StorageV1().StorageClasses().Get(context.Background(), storageClass, metav1.GetOptions{})
	if err != nil {
		log.Errorf("failed to deploy pods. Encountered an error while querying for the StorageClass: %s", err.Error())
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
	if err != nil {
		return err
	}

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
	allNodesRunning := nRunningNode >= nWorkerNodes

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
	labelKey := preferredLabelKey
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

	if i.isLabelCleanupRequired {
		defer i.removePreferredLabels()
	}

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

func (i *integration) finallyCleanupEverythingButLabels() error {
	i.isLabelCleanupRequired = false
	return i.finallyCleanupEverything()
}

func (i *integration) expectedMetroEnvVariablesAreSet() error {
	pstoreNodeUser := os.Getenv("POWERSTORE_NODE_USER")
	err := AssertExpectedAndActual(assert.Equal, true, pstoreNodeUser != "",
		"Expected POWERSTORE_NODE_USER env variable. Try export POWERSTORE_NODE_USER=nodeUser before running tests.")
	if err != nil {
		return err
	}

	password := os.Getenv("POWERSTORE_NODE_PASSWORD")
	err = AssertExpectedAndActual(assert.Equal, true, password != "",
		"Expected POWERSTORE_NODE_PASSWORD env variable. Try export POWERSTORE_NODE_PASSWORD=password before running tests.")
	if err != nil {
		return err
	}
	return i.expectedEnvVariablesAreSet()
}

func (i *integration) expectedEnvVariablesAreSet() error {
	nodeUser := os.Getenv("NODE_USER")
	err := AssertExpectedAndActual(assert.Equal, true, nodeUser != "",
		"Expected NODE_USER env variable. Try export NODE_USER=nodeUser before running tests.")
	if err != nil {
		return err
	}

	password := os.Getenv("PASSWORD")
	err = AssertExpectedAndActual(assert.Equal, true, password != "",
		"Expected PASSWORD env variable. Try export PASSWORD=password before running tests.")
	if err != nil {
		return err
	}

	i.scriptsDir = os.Getenv("SCRIPTS_DIR")
	err = AssertExpectedAndActual(assert.Equal, true, i.scriptsDir != "",
		"Expected SCRIPTS_DIR env variable. Try export SCRIPTS_DIR=scriptsDir before running tests.")
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

// havePodsMigrated checks if pods have migrated to new nodes.
//
// This function takes no parameters and returns a boolean indicating if pods have migrated,
// a string indicating which pod has migrated, and an error if there was an issue checking pods.
func (i *integration) havePodsMigrated() (bool, string, error) {
	currentPodToNodeMap := make(map[string]string)
	pods, err := i.listPodsByLabel(fmt.Sprintf("podmon.dellemc.com/driver=csi-%s", i.driverType))
	if err != nil {
		return false, "", err
	}

	for _, pod := range pods.Items {
		nsPodName := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)
		currentPodToNodeMap[nsPodName] = pod.Spec.NodeName
	}

	// Search through the labeled pod map and verify node change
	for podName, orginalNode := range i.labeledPodsToNodes {
		currentNode, ok := currentPodToNodeMap[podName]
		if !ok {
			return false, "", fmt.Errorf("expected %s pod to be assigned to a node, but no association was found", podName)
		}

		if currentNode != orginalNode {
			return true, podName, nil
		}
	}

	return false, "", nil
}

// verifyPodsDoNotMigrate periodically checks the labeled pods to see if any have migrated
// and returns false if migration is detected. At the end of waitTimeSec seconds, a nil error
// is returned if the pods have not migrated.
func (i *integration) verifyPodsDoNotMigrate(waitTimeSec int) error {
	log.Info("validating pods have not and will not migrate")

	// update the list of pods and the node they are on
	err := i.populateLabeledPodsToNodes()
	if err != nil {
		return fmt.Errorf("encountered an error while validating pods will not migrate: %s", err.Error())
	}

	timeoutDuration := time.Duration(waitTimeSec) * time.Second
	timeout, ticker, stop := newTimerWithTicker(waitTimeSec)
	defer stop()

	start := time.Now()

	for {
		select {
		case <-timeout.C:
			// if the timeout is reached, the test passes
			log.Infof("success: pods did not migrate in %d seconds", waitTimeSec)
			return nil
		case <-ticker.C:
			log.Infof("validating pods have not migrated (time left %v)", timeoutDuration-time.Since(start))

			migrated, podName, err := i.havePodsMigrated()
			if err != nil {
				return fmt.Errorf("encountered an error while validating pods have not migrated: %s", err)
			}

			if migrated {
				return fmt.Errorf("pod %s has migrated when it should not have", podName)
			}
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

// cliToolIsInstalledOnThisMachine validates whether the provided cliToolName resolves
// to an installed executable in the PATH.
func (i *integration) cliToolIsInstalledOnThisMachine(cliToolName string) error {
	log.Infof("checking if the %q executable is installed and part of the $PATH", cliToolName)
	_, err := exec.LookPath(cliToolName)
	if err != nil {
		return fmt.Errorf("could not find cli tool %q on this machine: %s", cliToolName, err.Error())
	}

	return nil
}

// getNodesWithPreferredLabelValue returns another function getNodes()
// that selects the nodes that are labeled with the provided labelValue
func (i *integration) getNodesWithPreferredLabelValue(labelValue string) func() (*corev1.NodeList, error) {
	opts := getPreferredNodeOpts(true, labelValue)

	getNodes := func() (*corev1.NodeList, error) {
		return i.getNodes(context.Background(), opts)
	}
	return getNodes
}

// failPreferredMetroConnection utilizes iptables entries to simulate network failure between
// the preferred storage array in a metro configuration and select worker nodes with the
// preferred=`labelValue` label.
func (i *integration) failPreferredMetroConnection(labelValue string) error {
	getNodes := i.getNodesWithPreferredLabelValue(labelValue)
	return i.setPreferredMetroConnection(MetroConnectionFail, getNodes)
}

// failNonPreferredMetroConnection utilizes iptables entries to simulate network failure between
// the non preferred storage array in a metro configuration and select worker nodes with the
// preferred=`labelValue` label.
func (i *integration) failNonPreferredMetroConnection(labelValue string) error {
	getNodes := i.getNodesWithPreferredLabelValue(labelValue)
	return i.setNonPreferredMetroConnection(MetroConnectionFail, getNodes)
}

// restorePreferredMetroConnection removes iptables entries added by failPreferredMetroConnection
// for worker nodes with preferred=`labelValue` label, restoring the network connection between the
// worker node and the preferred storage array in a metro configuration.
func (i *integration) restorePreferredMetroConnection(labelValue string) error {
	getNodes := i.getNodesWithPreferredLabelValue(labelValue)
	return i.setPreferredMetroConnection(MetroConnectionRestore, getNodes)
}

// restoreNonPreferredMetroConnection removes iptables entries added by failNonPreferredMetroConnection
// for worker nodes with preferred=`labelValue` label, restoring the network connection between the
// worker node and the non preferred storage array in a metro configuration.
func (i *integration) restoreNonPreferredMetroConnection(labelValue string) error {
	getNodes := i.getNodesWithPreferredLabelValue(labelValue)
	return i.setNonPreferredMetroConnection(MetroConnectionRestore, getNodes)
}

// setNonPreferredMetroConnection uses the gopowerstore client to determine the non preferred array for a
// metro volume, and pstcli to get the iSCSI IPs for the storage array, then updates the iptable entries
// for nodes returned by getNodes to either drop or accept (determined by operation) incoming packets from
// the iSCSI IPs.
func (i *integration) setNonPreferredMetroConnection(operation MetroConnection, getNodes func() (*corev1.NodeList, error)) error {
	preferredArray, err := i.getPowerStoreArrayInfo(i.storageClass.Parameters[pstoreID.KeyArrayID])
	if err != nil || preferredArray == nil {
		return fmt.Errorf("unable to get PowerStore secret: %s", err.Error())
	}

	// Initializing the gopowerstore client
	clientOptions := gopowerstore.NewClientOptions()
	clientOptions.SetInsecure(true)
	clientOptions.SetDefaultTimeout(2 * time.Second)
	pstoreClient, err := gopowerstore.NewClientWithArgs(preferredArray.Endpoint, preferredArray.Username, preferredArray.Password, clientOptions)
	if err != nil {
		return fmt.Errorf("unable to create PowerStore client: %s", err.Error())
	}

	pstoreClient.SetCustomHTTPHeaders(http.Header{
		"Application-Type": {fmt.Sprintf("%s/%s", pstoreID.VerboseName, core.SemVer)},
	})
	pstoreClient.SetLogger(&pstoreID.CustomLogger{})

	// Get the list of remote systems for the preferred array
	remoteSystems, err := pstoreClient.GetAllRemoteSystems(context.Background())
	if err != nil {
		log.Infof("unable to get the remote systems: %s", err.Error())
	}

	var nonPreferredKeyArrayID string
	// Filter the remote systems to find the arrayID of the non preferred array using the remote system mentioned in the storage class
	remoteSystemID := i.storageClass.Parameters[pstoreController.ReplicationPrefix+"/"+pstoreController.KeyReplicationRemoteSystem]
	for _, remoteSystem := range remoteSystems {
		if remoteSystem.Name == remoteSystemID {
			nonPreferredKeyArrayID = remoteSystem.SerialNumber
			break
		}
	}

	nonPreferredArray, err := i.getPowerStoreArrayInfo(nonPreferredKeyArrayID)
	if err != nil || nonPreferredArray == nil {
		return fmt.Errorf("unable to get PowerStore secret: %s", err.Error())
	}
	log.Infof("Non Preferred Array: %v", nonPreferredArray)

	err = i.dropIncomingPackets(operation, nonPreferredArray, getNodes)
	if err != nil {
		return fmt.Errorf("unable to drop incoming packets: %s", err.Error())
	}
	return nil
}

func (i *integration) getNonPreferredArray(storageClass *storagev1.StorageClass) (*pstoreArray.PowerStoreArray, error) {
	preferredArray, err := i.getPowerStoreArrayInfo(storageClass.Parameters[pstoreID.KeyArrayID])
	if err != nil || preferredArray == nil {
		return nil, fmt.Errorf("unable to get PowerStore secret: %s", err.Error())
	}

	// Initializing the gopowerstore client
	clientOptions := gopowerstore.NewClientOptions()
	clientOptions.SetInsecure(true)
	clientOptions.SetDefaultTimeout(2 * time.Second)
	pstoreClient, err := gopowerstore.NewClientWithArgs(preferredArray.Endpoint, preferredArray.Username, preferredArray.Password, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("unable to create PowerStore client: %s", err.Error())
	}

	pstoreClient.SetCustomHTTPHeaders(http.Header{
		"Application-Type": {fmt.Sprintf("%s/%s", pstoreID.VerboseName, core.SemVer)},
	})
	pstoreClient.SetLogger(&pstoreID.CustomLogger{})

	// Get the list of remote systems for the preferred array
	remoteSystems, err := pstoreClient.GetAllRemoteSystems(context.Background())
	if err != nil {
		log.Infof("unable to get the remote systems: %s", err.Error())
	}

	var nonPreferredKeyArrayID string
	// Filter the remote systems to find the arrayID of the non preferred array using the remote system mentioned in the storage class
	remoteSystemID := storageClass.Parameters[pstoreController.ReplicationPrefix+"/"+pstoreController.KeyReplicationRemoteSystem]
	for _, remoteSystem := range remoteSystems {
		if remoteSystem.Name == remoteSystemID {
			nonPreferredKeyArrayID = remoteSystem.SerialNumber
			break
		}
	}

	return i.getPowerStoreArrayInfo(nonPreferredKeyArrayID)
}

// setPreferredMetroConnection uses the configured storage class to determine the preferred array for a
// metro volume, and pstcli to get the iSCSI IPs for the storage array, then updates the iptable entries
// for nodes returned by getNodes to either drop or accept (determined by operation) incoming packets from
// the iSCSI IPs.
func (i *integration) setPreferredMetroConnection(operation MetroConnection, getNodes func() (*corev1.NodeList, error)) error {
	preferredArray, err := i.getPowerStoreArrayInfo(i.storageClass.Parameters[pstoreID.KeyArrayID])
	if err != nil || preferredArray == nil {
		return fmt.Errorf("unable to get PowerStore secret: %s", err.Error())
	}

	err = i.dropIncomingPackets(operation, preferredArray, getNodes)
	if err != nil {
		return fmt.Errorf("unable to drop incoming packets: %s", err.Error())
	}

	return nil
}

func (i *integration) dropIncomingPackets(operation MetroConnection, array *pstoreArray.PowerStoreArray, getNodes func() (*corev1.NodeList, error)) error {
	// get the iSCSI IPs via pstcli so we know which IPs to fail
	iscsiIPs, err := getIscsiIPs(array.Endpoint, array.Username, array.Password)
	if err != nil {
		return fmt.Errorf("unable to get iSCSI IPs: %s", err.Error())
	}
	log.Infof("iSCSI IPs for the Array %s: %v", array.Endpoint, iscsiIPs)

	// create a list of nodes to fail using the preferred label
	nodesToFail, err := getNodes()
	if err != nil {
		return fmt.Errorf("unable to fail nodes due to a failure querying the nodes: %s", err.Error())
	}

	// log in to each node and fail the iSCSI IPs
	for _, node := range nodesToFail.Items {

		// Get the node's IP address by looping over "status.addresses"
		// field in the K8s node resource and filtering by the "internalIP" type.
		var nodeIP string
		for _, address := range node.Status.Addresses {
			if address.Type == corev1.NodeInternalIP {
				nodeIP = address.Address
				break
			}
		}

		// build a single command to drop all incoming packets from all the iSCSI IPs
		var dropPacketsCmd, op string
		switch operation {
		case MetroConnectionFail:
			log.Infof("Attempting to block incoming packets from %s on node %s", iscsiIPs, nodeIP)
			op = "-A" // add rule to DROP all packets
		case MetroConnectionRestore:
			log.Infof("Attempting to allow incoming packets from %s on node %s", iscsiIPs, nodeIP)
			op = "-D" // delete previously added rule
		}
		for _, iscsiIP := range iscsiIPs {
			dropPacketsCmd = dropPacketsCmd + fmt.Sprintf("iptables %s INPUT -j DROP -s %s -m comment --comment %q; ", op, iscsiIP, "resiliency testing; delete me")
		}

		client := i.getSSHClient(nodeIP)
		if _, err := i.SSHExec(client, nodeIP, dropPacketsCmd); err != nil {
			return fmt.Errorf("encountered an error while attempting to drop incoming iSCSI packets on preferred nodes: %s", err.Error())
		}
	}
	return nil
}

func (i *integration) setStorageClass(storageClassParam string) error {
	storageClass, err := i.k8s.GetClient().StorageV1().StorageClasses().Get(context.Background(), storageClassParam, metav1.GetOptions{})
	if err != nil {
		message := fmt.Sprintf("getting storage class %s, error: %s", storageClassParam, err)
		return fmt.Errorf("%s", message)
	}

	if storageClass == nil {
		message := fmt.Sprintf("storage class %s not found", storageClassParam)
		return fmt.Errorf("%s", message)
	}
	i.storageClass = storageClass
	return nil
}

func (i *integration) disruptConnectivityBetweenMetroArrays(storageClassParam string) error {
	err := i.setStorageClass(storageClassParam)
	if err != nil {
		return err
	}
	return i.manageConnectivityBetweenMetroArrays(MetroConnectionFail)
}

func (i *integration) restoreConnectivityBetweenMetroArrays(storageClassParam string) error {
	err := i.setStorageClass(storageClassParam)
	if err != nil {
		return err
	}
	return i.manageConnectivityBetweenMetroArrays(MetroConnectionRestore)
}

func (i *integration) manageConnectivityBetweenMetroArrays(operation MetroConnection) error {
	preferredArray, err := i.getPowerStoreArrayInfo(i.storageClass.Parameters[pstoreID.KeyArrayID])
	if err != nil {
		return fmt.Errorf("unable to get PowerStore secret: %w", err)
	}
	if preferredArray == nil {
		return fmt.Errorf("unable to get PowerStore secret: array info is nil")
	}

	// Initializing the gopowerstore client
	clientOptions := gopowerstore.NewClientOptions()
	clientOptions.SetInsecure(true)
	clientOptions.SetDefaultTimeout(30 * time.Second)
	pstoreClient, err := gopowerstore.NewClientWithArgs(preferredArray.Endpoint, preferredArray.Username, preferredArray.Password, clientOptions)
	if err != nil {
		return fmt.Errorf("unable to get PowerStore client: %s", err.Error())
	}

	pstoreClient.SetCustomHTTPHeaders(http.Header{
		"Application-Type": {fmt.Sprintf("%s/%s", pstoreID.VerboseName, core.SemVer)},
	})
	pstoreClient.SetLogger(&pstoreID.CustomLogger{})

	// Get the list of remote systems for the preferred array
	remoteSystems, err := pstoreClient.GetAllRemoteSystems(context.Background())
	if err != nil {
		log.Infof("unable to get the remote systems: %s", err.Error())
	}

	var nonPreferredKeyArrayID string
	// Filter the remote systems to find the arrayID of the non preferred array using the remote system mentioned in the storage class
	remoteSystemID := i.storageClass.Parameters[pstoreController.ReplicationPrefix+"/"+pstoreController.KeyReplicationRemoteSystem]
	for _, remoteSystem := range remoteSystems {
		if remoteSystem.Name == remoteSystemID {
			nonPreferredKeyArrayID = remoteSystem.SerialNumber
			break
		}
	}

	nonPreferredArray, err := i.getPowerStoreArrayInfo(nonPreferredKeyArrayID)
	if err != nil {
		return fmt.Errorf("unable to get PowerStore secret: %w", err)
	}
	if nonPreferredArray == nil {
		return fmt.Errorf("unable to get PowerStore secret: array info is nil")
	}
	log.Infof("Non Preferred Array: %v", nonPreferredArray)

	err = i.blockUnblockReplicationTraffic(operation, preferredArray, nonPreferredArray)
	if err != nil {
		return fmt.Errorf("unable to drop incoming packets: %s", err.Error())
	}

	return nil
}

func (i *integration) blockUnblockReplicationTraffic(
	operation MetroConnection,
	array *pstoreArray.PowerStoreArray,
	remoteArray *pstoreArray.PowerStoreArray,
) error {
	// Discover local & remote iSCSI IPs
	localIPs, err := getIscsiIPs(array.Endpoint, array.Username, array.Password)
	if err != nil {
		return fmt.Errorf("get local iSCSI IPs: %w", err)
	}
	if len(localIPs) == 0 {
		return fmt.Errorf("no local iSCSI IPs discovered for array %s", array.Endpoint)
	}
	log.Infof("Local array %s iSCSI IPs: %v", array.Endpoint, localIPs)

	remoteIPs, err := getIscsiIPs(remoteArray.Endpoint, remoteArray.Username, remoteArray.Password)
	if err != nil {
		return fmt.Errorf("get remote iSCSI IPs: %w", err)
	}
	if len(remoteIPs) == 0 {
		return fmt.Errorf("no remote iSCSI IPs discovered for array %s", remoteArray.Endpoint)
	}
	log.Infof("Remote array %s iSCSI IPs: %v", remoteArray.Endpoint, remoteIPs)

	scriptsDir := os.Getenv("SCRIPTS_DIR")
	scriptPath := fmt.Sprintf("%s/%s", scriptsDir, blockTrafficScriptName)

	// Auth for SSH performed by the script; prefer keys. If using passwords, set SSHPASS.
	sshUser := os.Getenv("POWERSTORE_NODE_USER")
	if sshUser == "" {
		return fmt.Errorf("POWERSTORE_NODE_USER env must be set for SSH to local powerstore nodes")
	}

	sshPassword := os.Getenv("POWERSTORE_NODE_PASSWORD")
	os.Setenv("SSHPASS", sshPassword)

	// Decide action
	action := "block"
	if operation == MetroConnectionRestore {
		action = "unblock"
	}

	// Build block-traffic.sh command

	args := []string{
		scriptPath, // the script path passed to bash
		"--local-ips", strings.Join(localIPs, " "),
		"--remote-ips", strings.Join(remoteIPs, " "),
		"--action", action,
		"--ssh-user", sshUser,
		"--parallel", "4",
	}

	// Execute locally
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	env := os.Environ()
	env = append(env, "SSHPASS="+sshPassword)
	cmd.Env = env

	log.Infof("Running block-traffic.sh: %s %s", scriptPath, strings.Join(args, " "))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("block-traffic.sh failed: %w", err)
	}

	return nil
}

func (i *integration) setDriverNamespaceName(driverNamespaceName string) error {
	if driverNamespaceName == "" {
		return errors.New("expected a driver namespace name but it was empty")
	}

	i.driverNamespaceName = driverNamespaceName

	return nil
}

func (i *integration) setDriverSecretName(driverSecretName string) error {
	if driverSecretName == "" {
		return errors.New("expected a driver secret name but it was empty")
	}

	i.driverSecretName = driverSecretName

	return nil
}

// labeledNodesWithPodsAreTainted periodically checks nodes for the given taints, `taints` provided
// the node has a pod scheduled to it with the "podmon.dellemc.com/driver" label.
// If taints are found within the given wait time, `waitTimeSeconds`, it returns
// a nil error. If the timeout is reached, it returns an error.
func (i *integration) labeledNodesWithPodsAreTainted(labelValue, taints string, waitTimeSeconds int) error {
	// get nodes that either have or do not have the labelValue based on
	// the truthiness of `withLabel`
	opts := getPreferredNodeOpts(true, labelValue)

	timeout, ticker, stop := newTimerWithTicker(waitTimeSeconds)
	defer stop()

	log.Info("waiting for nodes to have the expected taints")

	start := time.Now()
	for {
		select {
		case <-timeout.C:
			return errors.New("timed out waiting to confirm nodes had the expected taints")
		case <-ticker.C:
			nodes, err := i.k8s.GetClient().CoreV1().Nodes().List(context.Background(), opts)
			if err != nil {
				return fmt.Errorf("unable to list nodes: %s", err.Error())
			}

			nodesWithPodsTainted := func() bool {
				for _, node := range nodes.Items {
					// get pods on this node that are monitored by resiliency
					podsOnNode, err := i.k8s.GetClient().CoreV1().Pods("").List(context.Background(), metav1.ListOptions{
						// gets pods on this node
						FieldSelector: fmt.Sprintf("spec.nodeName=%s", node.Name),
						// ignores the label value and should return pods that possess this label key,
						// indicating it is a pod monitored by podmon
						LabelSelector: "podmon.dellemc.com/driver",
					})
					if err != nil {
						log.Errorf("error listing pods on node %q: %s", node.Name, err.Error())
						continue
					}

					// exit early and return false if a pod exists on the node and the node is not yet tainted
					if len(podsOnNode.Items) != 0 && !i.isNodeFailed(node, taints) {
						log.Warnf("node %q still waiting to be tainted; time remaining %v",
							node.Name, time.Duration(waitTimeSeconds)*time.Second-time.Since(start))
						return false
					}
				}
				return true
			}()

			if nodesWithPodsTainted {
				log.Info("all nodes with pods are tainted as expected")
				return nil
			}
		}
	}
}

/* -- Helper functions -- */

// getPowerStoreArrayInfo reads the "powerstore-config" secret from the i.driverNamespaceName and returns
// a pointer to the PowerStoreArray that contains the provided globalID. If the array with the provided
// globalID is not found, an error and a nil pointer are returned.
func (i *integration) getPowerStoreArrayInfo(globalID string) (secret *pstoreArray.PowerStoreArray, err error) {
	// get the secret info using the powerstore array info from the storageclass
	secretObj, err := i.k8s.GetClient().CoreV1().Secrets(i.driverNamespaceName).Get(context.Background(), powerstoreSecretName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to fail the preferred metro connection. error encountered while getting driver secret: %s", err.Error())
	}

	// ingest the secret from the secretObj and unmarshal into the appropriate struct
	viper.SetConfigType("yaml")
	if err := viper.ReadConfig(bytes.NewBuffer(secretObj.Data[powerstoreSecretDataKeyName])); err != nil {
		return nil, fmt.Errorf("unable to ingest the \"powerstore-config\" Kubernetes secret: %s", err.Error())
	}

	var Secrets struct {
		Arrays []pstoreArray.PowerStoreArray `yaml:"arrays"`
	}
	if err := viper.Unmarshal(&Secrets); err != nil {
		return nil, fmt.Errorf("unable to unmarshal Kubernetes secret into PowerStoreArrays struct: %s", err.Error())
	}

	// use the provided globalID to look up the desired array secret
	for _, array := range Secrets.Arrays {
		if array.GlobalID == globalID {
			secret = &array
			break
		}
	}

	if reflect.DeepEqual(secret, pstoreArray.PowerStoreArray{}) {
		return nil, fmt.Errorf("unable to find the PowerStore array info from the provided StorageClass %q, and Secret %q", i.storageClass.Name, i.driverSecretName)
	}

	return secret, nil
}

// getPreferredNodeOpts returns a set of options used to query the Kubernetes API
// for nodes. If matchLabel is true, the returned options will select nodes that
// have the provided labelValue. If matchLabel is false, the returned options
// will select nodes that both are not part of the control plane and do not have
// the provided labelValue.
func getPreferredNodeOpts(matchLabel bool, labelValue string) metav1.ListOptions {
	if matchLabel {
		// select nodes that are labeled with the provided labelValue
		return metav1.ListOptions{
			LabelSelector: strings.Join([]string{preferredLabelKey, labelValue}, "="),
		}
	}

	// select nodes that are both not part of the control plane
	// and do not have the provided labelValue
	return metav1.ListOptions{
		LabelSelector: preferredLabelKey + "!=" + labelValue + "," + controlPlane + "!=",
	}
}

// getSSHClient determines whether the test is running on vanilla Kubernetes
// or OpenShift and configures the ssh client accordingly. SSH commands run
// with this client will be sent to the targetIP IP address if run against
// Kubernetes or the previously configured i.bastionNode IP address if running
// on OpenShift. Environment variables NODE_USER and PASSWORD are used for
// the SSH user and password.
func (i *integration) getSSHClient(targetIP string) ssh.CommandExecution {
	username := os.Getenv("NODE_USER")
	password := os.Getenv("PASSWORD")
	hostname := targetIP
	if i.isOpenshift {
		hostname = i.bastionNode
	}

	info := ssh.AccessInfo{
		Hostname: hostname,
		Port:     "22",
		Username: username,
		Password: password,
	}

	wrapper := ssh.NewWrapper(&info)

	return ssh.CommandExecution{
		AccessInfo: &info,
		SSHWrapper: wrapper,
		Timeout:    sshTimeoutDuration,
	}
}

// SSHExec provides a convenient method for executing commands over ssh on cluster worker nodes
// making adjustments to the request, as necessary, to support connections to OpenShift nodes.
func (i *integration) SSHExec(client ssh.CommandExecution, IPAddr string, cmd string) (response []string, err error) {
	if i.isOpenshift {
		cmd = fmt.Sprintf("ssh %s core@%s '%s'", sshOptions, IPAddr, cmd)
	}

	err = client.Run(cmd)
	response = client.GetOutput()
	if err != nil {
		return response, fmt.Errorf("encountered an error executing ssh on IP %q: %s :%s", IPAddr, response, err.Error())
	}
	for _, line := range response {
		log.Info(line)
	}

	return response, nil
}

// getIscsiIPs uses pstcli to query the provided PowerStore array for any available iSCSI IPs
func getIscsiIPs(endpoint, username, password string) (iscsiIPs []string, err error) {
	// isolate the IP address for the API endpoint
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimSuffix(endpoint, "/")
	endpoint = strings.TrimSuffix(endpoint, "/api/rest")

	// build the pstcli command to query for iSCSI IPs
	iscsiIPQuery := "purposes contains Storage_Iscsi_Target"
	pstcli := exec.Command("pstcli", "-d", endpoint, "-u", username, "-p", password, "-ssl", "accept",
		"ip_pool_address", "show", "-select", "address", "-query", iscsiIPQuery, "-output", "json", "-raw")

	log.Infof("attempting to get PowerStore iSCSI IPs: %v", pstcli.Args)

	// execute the command and get the results
	iscsiPortsJSON, err := pstcli.CombinedOutput()
	if err != nil {
		return []string{}, fmt.Errorf("unable to get iSCSI IPs: %s: %s", err.Error(), string(iscsiPortsJSON))
	}

	// unmarshal the json response into something we can more easily parse
	iscsiPorts := []struct {
		Address string `json:"address"`
	}{}
	err = json.Unmarshal(iscsiPortsJSON, &iscsiPorts)
	if err != nil {
		return []string{}, fmt.Errorf("unable to unmarshal the response returned by pstcli when querying for iSCSI IPs: %s", err.Error())
	}

	// format the response as a string array
	for _, IP := range iscsiPorts {
		iscsiIPs = append(iscsiIPs, IP.Address)
	}

	return iscsiIPs, nil
}

// getNodes uses the k8s client from the integration struct to query the kuberentest API for
// a list of nodes matching any opts supplied.
func (i *integration) getNodes(ctx context.Context, opts metav1.ListOptions) (nodes *corev1.NodeList, err error) {
	return i.k8s.GetClient().CoreV1().Nodes().List(ctx, opts)
}

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

		// Check to see if the node was failed, if it wasn't then the pods would not have migrated.
		_, ok = i.nodesToTaints[initialNode]
		if !ok {
			log.Infof("node %s is not tainted so it was not a failed node", currentNode)
			continue
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
		for _, name := range i.preferredLabeledNodes {
			if !tracker[name] {
				tracker[name] = true
				candidates = append(candidates, name)
			}
		}
	}

	log.Infof("All the candidate nodes to fail are %v", candidates)

	if i.driverType == "" {
		return nil, fmt.Errorf("driver type not specified")
	}

	// Get deployment and see how many replicas for the controller there are.
	deployment, err := i.getDriverControllerDeployment(i.driverType, i.driverNamespaceName)
	if err != nil {
		return failedNodes, err
	}

	if int(*deployment.Spec.Replicas) == 1 {
		expectedControllerPodName := i.driverType + "-controller"

		// If there is only one replica, get node that the controller pod is running on and ensure that we don't fail it.
		driverPods, err := i.k8s.GetClient().CoreV1().Pods(i.driverType).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return failedNodes, err
		}

		for _, pod := range driverPods.Items {
			if strings.Contains(pod.Name, expectedControllerPodName) {
				log.Infof("Controller pod is running on node %s", pod.Spec.NodeName)
				i.shouldNotFailNode = pod.Spec.NodeName
				break
			}
		}
	}

	failed := 0
	for _, name := range candidates {
		// For CSI driver pod run the script from test host not worker node
		if failureType == "driverpod" {
			cmd := exec.Command( // #nosec G204
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

		if name == i.shouldNotFailNode {
			log.Infof("Detected only a single controller replica running on a worker node; skipping failing this node %s", name)
			continue
		}

		if failed < numberToFail {
			ip, ok := nameToIP[name]
			if !ok {
				log.Warnf("the IP address for node %q is unknown; skipping node failure", name)
				continue
			}
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
		"sshpass -p 'fedora' virtctl ssh vm/%s --namespace=%s --username=fedora "+
			"--local-ssh=true --local-ssh-opts='-o StrictHostKeyChecking=no' --local-ssh-opts='-o UserKnownHostsFile=/dev/null' "+
			"--command \"printf '%s' | sudo dd of=/dev/vdc bs=1 count=150 conv=notrunc\"",
		vmName, namespace, expectedData)

	var writeErr error
	var writeOut []byte
	retries := 3
	retryDelay := 30 * time.Second

	for attempt := 0; attempt <= retries; attempt++ {
		log.Printf("Running (attempt %d/%d): %s", attempt+1, retries, writeCmd)
		writeOut, writeErr = exec.Command("bash", "-c", writeCmd).CombinedOutput()
		if writeErr != nil {
			log.Printf("Write failed on %s (attempt %d/%d): %s", vmName, attempt+1, retries, string(writeOut))
			if attempt < retries {
				time.Sleep(retryDelay)
			}
		} else {
			log.Printf("Write output for %s (attempt %d/%d): %s", vmName, attempt+1, retries, string(writeOut))
			break
		}
	}

	if writeErr != nil {
		return writeErr
	}

	// Read and verify
	return i.verifyDiskContentOnVM(vmName, namespace)
}

func (i *integration) verifyDiskContentOnVM(vmName, namespace string) error {
	readCmd := fmt.Sprintf(
		"sshpass -p 'fedora' virtctl ssh vm/%s --namespace=%s --username=fedora "+
			"--local-ssh=true --local-ssh-opts='-o StrictHostKeyChecking=no'  --local-ssh-opts='-o UserKnownHostsFile=/dev/null' "+
			"--command \"sudo dd if=/dev/vdc bs=1 count=150\"",
		vmName, namespace)

	retries := 3
	retryDelay := 30 * time.Second
	var readOut []byte
	var readErr error

	for attempt := 0; attempt <= retries; attempt++ {
		log.Printf("Running (attempt %d/%d): %s", attempt+1, retries, readCmd)
		readOut, readErr = exec.Command("bash", "-c", readCmd).CombinedOutput()
		if readErr != nil {
			log.Printf("Read failed on %s (attempt %d/%d): %s", vmName, attempt+1, retries, string(readOut))
			if attempt < retries {
				time.Sleep(retryDelay)
			}
		} else {
			log.Printf("Read output for %s (attempt %d/%d): %s", vmName, attempt+1, retries, string(readOut))
			break
		}
	}

	if readErr != nil {
		return readErr
	}

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
		default:
			continue
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
			nodeObj.ObjectMeta.Labels[preferredLabelKey] = preferred
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
				if nodeObj.ObjectMeta.Labels[preferredLabelKey] != preferred {
					return fmt.Errorf("expected pod to be scheduled to a node with the preferred=%s label. Pod %q is on node %q",
						preferred, pod.Name, pod.Spec.NodeName)
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
				if nodeName == i.shouldNotFailNode {
					log.Warnf("Due to single replica and controller pod running on a preferred node, pod %s is still running on a preferred node", pod.Name)
					continue
				}

				if nodeName == labeledNode {
					return fmt.Errorf("Pod '%s' is on preferred node '%s'", pod.Name, nodeName)
				}
			}
		}
	}
	return nil
}

func (i *integration) checkArrayConnectivity(storageClassName string, checkFunc func(*pstoreArray.PowerStoreArray) error) error {
	storageClass, err := i.k8s.GetClient().StorageV1().StorageClasses().Get(context.Background(), storageClassName, metav1.GetOptions{})
	if err != nil {
		log.Errorf("Encountered an error while querying for the StorageClass: %s", err.Error())
		return err
	}

	preferredArray, err := i.getPowerStoreArrayInfo(storageClass.Parameters[pstoreID.KeyArrayID])
	if err != nil || preferredArray == nil {
		return fmt.Errorf("unable to get PowerStore details from secret: %s", err.Error())
	}
	if err := checkFunc(preferredArray); err != nil {
		return err
	}

	// Initializing the gopowerstore client
	clientOptions := gopowerstore.NewClientOptions()
	clientOptions.SetInsecure(true)
	clientOptions.SetDefaultTimeout(30 * time.Second)
	pstoreClient, err := gopowerstore.NewClientWithArgs(preferredArray.Endpoint, preferredArray.Username, preferredArray.Password, clientOptions)
	if err != nil {
		return fmt.Errorf("unable to create PowerStore client: %s", err.Error())
	}

	pstoreClient.SetCustomHTTPHeaders(http.Header{
		"Application-Type": {fmt.Sprintf("%s/%s", pstoreID.VerboseName, core.SemVer)},
	})
	pstoreClient.SetLogger(&pstoreID.CustomLogger{})

	// Get the list of remote systems for the preferred array
	remoteSystems, err := pstoreClient.GetAllRemoteSystems(context.Background())
	if err != nil {
		return fmt.Errorf("unable to get the remote systems: %s", err.Error())
	}

	var nonPreferredKeyArrayID string
	// Filter the remote systems to find the arrayID of the non preferred array using the remote system mentioned in the storage class
	remoteSystemID := storageClass.Parameters[pstoreController.ReplicationPrefix+"/"+pstoreController.KeyReplicationRemoteSystem]
	for _, remoteSystem := range remoteSystems {
		if remoteSystem.Name == remoteSystemID {
			nonPreferredKeyArrayID = remoteSystem.SerialNumber
			break
		}
	}

	nonPreferredArray, err := i.getPowerStoreArrayInfo(nonPreferredKeyArrayID)
	if err != nil || nonPreferredArray == nil {
		return fmt.Errorf("unable to get PowerStore details from secret: %s", err.Error())
	}
	log.Infof("Non Preferred Array: %v", nonPreferredArray)

	return checkFunc(nonPreferredArray)
}

func (i *integration) checkUniformConnectivity(array *pstoreArray.PowerStoreArray) error {
	if array.HostConnectivity == nil || array.HostConnectivity.Metro.ColocatedLocal.Size() == 0 || array.HostConnectivity.Metro.ColocatedRemote.Size() == 0 {
		return fmt.Errorf("Array %s not configured for metro connectivity", array.GlobalID)
	}
	return nil
}

func (i *integration) checkNonUniformConnectivity(array *pstoreArray.PowerStoreArray) error {
	if array.HostConnectivity == nil || array.HostConnectivity.Local.Size() == 0 {
		return fmt.Errorf("Array %s not configured for local connectivity", array.GlobalID)
	}
	return nil
}

func (i *integration) theArraysInStorageclassAreInUniformConfiguration(storageClassName string) error {
	return i.checkArrayConnectivity(storageClassName, i.checkUniformConnectivity)
}

func (i *integration) theArraysInStorageclassAreInNonUniformConfiguration(storageClassName string) error {
	return i.checkArrayConnectivity(storageClassName, i.checkNonUniformConnectivity)
}

func (i *integration) thereAreNodesLabelled(labelValue string) error {
	getNodes := i.getNodesWithPreferredLabelValue(labelValue)
	nodes, err := getNodes()
	if err != nil {
		return fmt.Errorf("No nodes found with label %s : %s", labelValue, err.Error())
	}

	if len(nodes.Items) == 0 {
		return fmt.Errorf("No nodes found with label %s", labelValue)
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
		if isWorkerNode(node) && node.ObjectMeta.Labels[preferredLabelKey] == label {
			return true
		}

		return false
	}

	nameToIP, numberToLabel, err := i.createNodeNameMap(numNodes, filter)
	if err != nil {
		return err
	}

	log.Printf("Labeling %d nodes with label %s=%s", numberToLabel, preferredLabelKey, label)

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
	labelKey := preferredLabelKey + "=" + label // Get nodes with the "preferred" label
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

	return timeout, ticker, stop
}

func (i *integration) getDriverControllerDeployment(driverType string, driverNamespace string) (*v1.Deployment, error) {
	log.Infof("Getting deployment for driver: %s", driverType)
	deployments, err := i.k8s.GetClient().AppsV1().Deployments(driverNamespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Errorln("Deployment list error: ", err)
		return nil, err
	}

	for _, deployment := range deployments.Items {
		log.Info("Found deployment: ", deployment.Name)
		if deployment.Name == driverType+"-controller" {
			return &deployment, nil
		}
	}

	return nil, errors.New("deployment not found")
}

func (i *integration) skipIfIsNotCompatibleWith(failure, driverType string) error {
	if driverType == "powerstore" {
		log.Infoln("Checking if the follwing test is compatible with PowerStore environment")

		deployment, err := i.getDriverControllerDeployment(driverType, driverType)
		if err != nil {
			return err
		}

		isCompatible := true
		for _, container := range deployment.Spec.Template.Spec.Containers {
			if strings.Contains(container.Image, "podmon") {
				for _, arg := range container.Args {
					if strings.Contains(arg, "skipArrayConnectionValidation") {
						split := strings.Split(arg, "=")
						if split[1] == "false" && failure == "kubeletdown" {
							log.Info("For running kubeletdown test on PowerStore, skipArrayConnectionValidation should be set to true")
							isCompatible = false
						}

						break
					}
				}

				break
			}
		}

		if !isCompatible {
			log.Warnf("Skipping this scenario. Test is not compatible with %s environment", driverType)
			return godog.ErrSkip
		}
	}
	return nil
}

func (i *integration) iSetTheCorrectDriverTypeTo(driverType string) error {
	log.Infof("Setting driver type to: %s", driverType)
	i.setDriverType(driverType)
	return nil
}

func (i *integration) iVerifyThatIsImmediateBinding(storageClassParam string) error {
	var err error
	i.storageClass, err = i.k8s.GetClient().StorageV1().StorageClasses().Get(context.Background(), storageClassParam, metav1.GetOptions{})
	if err != nil {
		message := fmt.Sprintf("getting storage class %s, error: %s", storageClassParam, err)
		return fmt.Errorf("%s", message)
	}

	if i.storageClass == nil || i.storageClass.VolumeBindingMode == nil {
		message := fmt.Sprintf("storage class %s not found", storageClassParam)
		return fmt.Errorf("%s", message)
	}

	if *i.storageClass.VolumeBindingMode != storagev1.VolumeBindingImmediate {
		message := fmt.Sprintf("storage class %s should have VolumeBindingMode set to Immediate", storageClassParam)
		return fmt.Errorf("%s", message)
	}

	return nil
}

func (i *integration) getTestNamespacePrefix(driverType string) string {
	var prefix string
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

	return prefix
}

func (i *integration) iDeployPvcOn(nVols int, _, driverType string) error {
	if i.storageClass == nil {
		return fmt.Errorf("[iDeployPvcOn] storage class not set; run step to verify binding first")
	}

	nsPrefix := i.getTestNamespacePrefix(driverType)

	// Check if the namespace exists
	_, err := i.k8s.GetClient().CoreV1().Namespaces().Get(context.Background(), nsPrefix, metav1.GetOptions{})
	if err != nil {
		log.Infof("Creating namespace: %s", nsPrefix)
		_, err := i.k8s.GetClient().CoreV1().Namespaces().Create(context.Background(), &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: nsPrefix,
			},
		}, metav1.CreateOptions{})
		if err != nil {
			log.Errorf("Failed to create namespace: %s", err)
		}
	}

	volumeMode := corev1.PersistentVolumeFilesystem

	for j := 0; j < nVols; j++ {
		name := fmt.Sprintf("pvc-%d", j)

		_, err := i.k8s.GetClient().CoreV1().PersistentVolumeClaims(nsPrefix).Create(context.Background(), &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				StorageClassName: &i.storageClass.Name,
				AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("8Gi"),
					},
				},
				VolumeMode: &volumeMode,
			},
		}, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create pvc %s: %w", name, err)
		}
	}

	return nil
}

func (i *integration) iEnsureAllVolumesOnForAreBound(storageClassParam, driverType string) error {
	maxAttempts := 5
	for j := 0; j < maxAttempts; j++ {
		err := i.allVolumesAreBound(storageClassParam, driverType)
		if err == nil {
			return nil
		}

		log.Warnf("Failed to verify all volumes are bound. Retrying...")
		time.Sleep(5 * time.Second)
	}

	return fmt.Errorf("Failed to verify all volumes are bound after %d attempts", maxAttempts)
}

func (i *integration) getAllPVCsInNamespace(storageClassParam, _, namespace string) ([]*corev1.PersistentVolumeClaim, error) {
	pvcList, err := i.k8s.GetClient().CoreV1().PersistentVolumeClaims(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		message := fmt.Sprintf("getting storage class %s, error: %s", storageClassParam, err)
		return nil, fmt.Errorf("%s", message)
	}

	var pvcs []*corev1.PersistentVolumeClaim
	for _, pvc := range pvcList.Items {
		if pvc.Spec.StorageClassName != nil && *pvc.Spec.StorageClassName == storageClassParam {
			pvcs = append(pvcs, &pvc)
		}
	}

	return pvcs, nil
}

func (i *integration) iEnsureThatAllMetroVolumesInNamespaceAreStable(storageClassParam, driverType string, namespace string) error {
	pvcList, err := i.getAllPVCsInNamespace(storageClassParam, driverType, namespace)
	if err != nil {
		return err
	}
	if len(pvcList) == 0 {
		return fmt.Errorf("no pvc found for storage class %s in namespace %s", storageClassParam, namespace)
	}

	maxAttempts := 20
	for j := 0; j < maxAttempts; j++ {
		var err error
		// Set the metroVolInfo to be used in other metro action execution.
		i.metroVolInfo, err = i.allMetroVolumeSessionsAreStable(pvcList)
		if err == nil {
			for name, data := range i.metroVolInfo {
				log.Infof("metro vol %s replication ID: %s", name, data.ReplicationSessions[0].ID)
			}
			return nil
		}

		log.Warnf("Array Metro sessions are not stable. Received error: %s. Retrying...", err.Error())
		time.Sleep(10 * time.Second)
	}

	return fmt.Errorf("Failed to verify all Metro volumes in namespace %s are stable after %d attempts", namespace, maxAttempts)
}

func (i *integration) iEnsureMetroVolumesStableInTestNamespaces(storageClassParam, driverType string) error {
	// i.podCount is actually the number of test namespaces. Because it maps to --instances when invoking the deploy script
	if i.storageClass == nil {
		err := i.setStorageClass(storageClassParam)
		if err != nil {
			return err
		}
	}
	nsPrefix := i.getTestNamespacePrefix(driverType)
	for podIdx := 1; podIdx <= i.podCount; podIdx++ {
		namespace := fmt.Sprintf("%s%d", nsPrefix, podIdx)
		log.Infof("Ensuring Metro volumes in namespace are stable: %s ", namespace)
		err := i.iEnsureThatAllMetroVolumesInNamespaceAreStable(storageClassParam, driverType, namespace)
		if err != nil {
			return err
		}

	}
	return nil
}

func (i *integration) getAllVolumesOfStorageClass(storageClassParam, driverType string) ([]*corev1.PersistentVolumeClaim, error) {
	nsPrefix := i.getTestNamespacePrefix(driverType)

	pvcList, err := i.k8s.GetClient().CoreV1().PersistentVolumeClaims(nsPrefix).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		message := fmt.Sprintf("getting storage class %s, error: %s", storageClassParam, err)
		return nil, fmt.Errorf("%s", message)
	}

	var pvcs []*corev1.PersistentVolumeClaim
	for _, pvc := range pvcList.Items {
		if pvc.Spec.StorageClassName != nil && *pvc.Spec.StorageClassName == storageClassParam {
			pvcs = append(pvcs, &pvc)
		}
	}

	return pvcs, nil
}

func (i *integration) allVolumesAreBound(storageClassParam, driverType string) error {
	pvcList, err := i.getAllVolumesOfStorageClass(storageClassParam, driverType)
	if err != nil {
		return err
	}

	for _, pvc := range pvcList {
		if pvc.Status.Phase != corev1.ClaimBound {
			return fmt.Errorf("pvc %s is not bound", pvc.Name)
		}
	}

	return nil
}

func (i *integration) iEnsureThatAllMetroVolumesOnForAreStable(storageClassParam, driverType string) error {
	pvcList, err := i.getAllVolumesOfStorageClass(storageClassParam, driverType)
	if err != nil {
		return err
	}
	if len(pvcList) == 0 {
		return fmt.Errorf("no pvc found for storage class %s", storageClassParam)
	}

	maxAttempts := 20
	for j := 0; j < maxAttempts; j++ {
		var err error
		// Set the metroVolInfo to be used in other metro action execution.
		i.metroVolInfo, err = i.allMetroVolumeSessionsAreStable(pvcList)
		if err == nil {
			return nil
		}

		log.Warnf("Array Metro sessions are not stable. Received error: %s. Retrying...", err.Error())
		time.Sleep(10 * time.Second)
	}

	return fmt.Errorf("Failed to verify all Metro volumes are stable after %d attempts", maxAttempts)
}

func (i *integration) allMetroVolumeSessionsAreStable(pvcList []*corev1.PersistentVolumeClaim) (map[string]volumeInformation, error) {
	log.Infof("checking if all metro Volume sessions are stable")
	result := make(map[string]volumeInformation)

	preferredArray, err := i.getPowerStoreArrayInfo(i.storageClass.Parameters[pstoreID.KeyArrayID])
	if err != nil || preferredArray == nil {
		return nil, fmt.Errorf("unable to get PowerStore secret: %s", err.Error())
	}

	endpoint := isolateIPAddress(preferredArray.Endpoint)

	for _, pvc := range pvcList {
		// Get the PV for further information.
		pv, err := i.k8s.GetClient().CoreV1().PersistentVolumes().Get(context.Background(), pvc.Spec.VolumeName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		pstcli := exec.Command("pstcli", "-d", endpoint, "-u", preferredArray.Username, "-p", preferredArray.Password, "-ssl", "accept",
			"volume", "-name", pv.Name, "show", "-output", "json", "-raw")
		log.Infof("attempting to get PowerStore volume information: %v", pstcli.Args)

		// execute the command and get the results
		response, err := pstcli.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("unable to get volume information: %s", err.Error())
		}

		// unmarshal the json response into something we can more easily parse
		volumeInformation := volumeInformation{}
		err = json.Unmarshal(response, &volumeInformation)
		if err != nil {
			return nil, fmt.Errorf("unable to unmarshal the response returned by pstcli when querying for iSCSI IPs: %s", err.Error())
		}

		if len(volumeInformation.ReplicationSessions) == 0 {
			return nil, fmt.Errorf("volume %s has no metro sessions", pv.Name)
		}

		if volumeInformation.ReplicationSessions[0].State != "OK" {
			return nil, fmt.Errorf("volume metro session %s is not stable, state: %s", volumeInformation.Name, volumeInformation.ReplicationSessions[0].State)
		}

		result[pv.Name] = volumeInformation
	}

	for name, data := range result {
		log.Infof("Metro volume %s has state: %s", name, data.ReplicationSessions[0].State)
	}

	return result, nil
}

func (i *integration) executeMetroAction(endpoint string, array *pstoreArray.PowerStoreArray, action string) error {
	for name, data := range i.metroVolInfo {
		pstcli := exec.Command("pstcli", "-d", endpoint, "-u", array.Username, "-p", array.Password, "-ssl", "accept",
			"replication_session", "-id", data.ReplicationSessions[0].ID, action) // #nosec G204

		// execute the command and get the results
		response, err := pstcli.CombinedOutput()
		if err != nil {
			return fmt.Errorf("unable to execute metro action [%s] on replication session for volume %s: %s", action, name, err.Error())
		}

		log.Infof("Executed %s on Metro volume: %s. Response %s", action, name, response)
	}

	return nil
}

func (i *integration) iExecuteMetroActionOnNonPreferredMetroVolumes(action, _, _ string) error {
	if i.storageClass == nil {
		return fmt.Errorf("[iExecuteMetroActionOnNonPreferredMetroVolumes] storage class not set; run step to verify binding first")
	}
	log.Infof("Executing action %s on nonPreferred Metro volume:", action)
	nonPreferredArray, err := i.getNonPreferredArray(i.storageClass)
	if err != nil {
		return err
	}

	endpoint := isolateIPAddress(nonPreferredArray.Endpoint)
	maxAttempts := 20
	for j := 0; j < maxAttempts; j++ {
		err = i.executeMetroAction(endpoint, nonPreferredArray, action)
		if err != nil {
			log.Warnf("[Metro Action] Received error: %s.", err.Error())
			time.Sleep(10 * time.Second)
			continue
		}

		return nil
	}

	return fmt.Errorf("[Metro Action] Failed to execute metro action on nonPreferred Array %s after %d attempts", action, maxAttempts)
}

func (i *integration) iExecuteMetroActionOnForForMetroVolumes(action, _, _ string) error {
	if i.storageClass == nil {
		return fmt.Errorf("[iExecuteMetroActionOnForForMetroVolumes] storage class not set; run step to verify binding first")
	}

	preferredArray, err := i.getPowerStoreArrayInfo(i.storageClass.Parameters[pstoreID.KeyArrayID])
	if err != nil || preferredArray == nil {
		return fmt.Errorf("unable to get PowerStore secret: %s", err.Error())
	}

	endpoint := isolateIPAddress(preferredArray.Endpoint)
	maxAttempts := 20
	for j := 0; j < maxAttempts; j++ {
		err = i.executeMetroAction(endpoint, preferredArray, action)
		if err != nil {
			log.Warnf("[Metro Action] Received error: %s.", err.Error())
			time.Sleep(10 * time.Second)
			continue
		}

		return nil
	}

	return fmt.Errorf("[Metro Action] Failed to execute metro action %s after %d attempts", action, maxAttempts)
}

// Waits for whatever is passed in amount of seconds
func (i integration) wait(seconds int) {
	log.Infof("Waiting %d seconds", seconds)
	time.Sleep(time.Duration(seconds) * time.Second)
}

func (i *integration) iSoftFractureTheMetroVolumesOnFor(_, _ string) error {
	var err error
	preferredArray, err := i.getPowerStoreArrayInfo(i.storageClass.Parameters[pstoreID.KeyArrayID])
	if err != nil || preferredArray == nil {
		return fmt.Errorf("unable to get PowerStore secret: %s", err.Error())
	}

	endpoint := isolateIPAddress(preferredArray.Endpoint)
	maxAttempts := 20
	for j := 0; j < maxAttempts; j++ {
		// Steps to Soft Fracture the Metro volume, i.e. pause and resume.
		err = i.executeMetroAction(endpoint, preferredArray, "pause")
		if err != nil {
			log.Warnf("[Soft Fracture] Received error: %s. Retrying...", err.Error())
			time.Sleep(10 * time.Second)
			continue
		}

		err = i.executeMetroAction(endpoint, preferredArray, "resume")
		if err != nil {
			log.Warnf("[Soft Fracture] Received error: %s. Retrying...", err.Error())
			time.Sleep(10 * time.Second)
			continue
		}

		err = nil
		break
	}

	if err != nil {
		return fmt.Errorf("[Soft Fracture] Unable to soft fracture all metro volumes after %d attempts", maxAttempts)
	}

	for name := range i.metroVolInfo {
		log.Infof("[Soft Fracture] Soft Fractured Metro volume: %s", name)

		pstcli := exec.Command("pstcli", "-d", endpoint, "-u", preferredArray.Username, "-p", preferredArray.Password, "-ssl", "accept",
			"volume", "-name", name, "show", "-output", "json", "-raw")
		log.Infof("attempting to get PowerStore volume information: %v", pstcli.Args)

		// execute the command and get the results
		response, err := pstcli.CombinedOutput()
		if err != nil {
			return fmt.Errorf("unable to get volume information: %s", err.Error())
		}

		// unmarshal the json response into something we can more easily parse
		volumeInformation := volumeInformation{}
		err = json.Unmarshal(response, &volumeInformation)
		if err != nil {
			return fmt.Errorf("unable to unmarshal the response returned by pstcli when querying for iSCSI IPs: %s", err.Error())
		}

		if volumeInformation.ReplicationSessions[0].State != "Fractured" {
			return fmt.Errorf("Replication sessions was not in fractured stated")
		}
	}

	log.Infoln("Soft Fracture complete")
	return nil
}

func (i *integration) iDeleteSnapshotsOfMetroVolumes(id string) error {
	preferredArray, err := i.getPowerStoreArrayInfo(i.storageClass.Parameters[pstoreID.KeyArrayID])
	if err != nil || preferredArray == nil {
		return fmt.Errorf("unable to get PowerStore secret: %s", err.Error())
	}

	// Initializing the gopowerstore client
	clientOptions := gopowerstore.NewClientOptions()
	clientOptions.SetDefaultTimeout(5 * time.Second)
	clientOptions.SetInsecure(true)
	pstoreClient, err := gopowerstore.NewClientWithArgs(preferredArray.Endpoint, preferredArray.Username, preferredArray.Password, clientOptions)
	if err != nil {
		return fmt.Errorf("unable to create PowerStore client: %s", err.Error())
	}

	pstoreClient.SetCustomHTTPHeaders(http.Header{
		"Application-Type": {fmt.Sprintf("%s/%s", pstoreID.VerboseName, core.SemVer)},
	})
	pstoreClient.SetLogger(&pstoreID.CustomLogger{})

	myCtx, cancel := context.WithDeadline(context.Background(), time.Now().Add(30*time.Second))
	defer cancel()

	snapshots, err := pstoreClient.GetSnapshotsByVolumeID(myCtx, id)
	if err != nil {
		log.Infof("unable to get the remote systems: %s", err.Error())
	}

	if len(snapshots) > 0 {
		for _, snapshot := range snapshots {
			log.Infof("Deleting snapshot: %s", snapshot.Name)
			_, err = pstoreClient.DeleteSnapshot(myCtx, nil, snapshot.ID)
			if err != nil {
				log.Infof("unable to delete snapshot: %s", err.Error())
			}
		}
	}

	return nil
}

func (i *integration) iDeployPodsForAllMetroVolumesOnFor(storageClassParam, driverType, preferredAffinity string) error {
	pvcList, err := i.getAllVolumesOfStorageClass(storageClassParam, driverType)
	if err != nil {
		return err
	}

	podmontestRegistry := os.Getenv("REGISTRY_HOST")
	if podmontestRegistry == "" {
		return fmt.Errorf("var REGISTRY_HOST is not set, unable to properly deploy podmontest pods")
	}

	podmontestVersion := os.Getenv("PODMONTEST_VERSION")
	if podmontestVersion == "" {
		return fmt.Errorf("var PODMONTEST_VERSION is not set, unable to properly deploy podmontest pods")
	}

	volMounts := []corev1.VolumeMount{}
	vols := []corev1.Volume{}
	volCount := 0
	for _, pvc := range pvcList {
		volMounts = append(volMounts, corev1.VolumeMount{
			Name:      "vol-" + strconv.Itoa(volCount),
			MountPath: "/data" + strconv.Itoa(volCount),
		})

		vols = append(vols, corev1.Volume{
			Name: "vol-" + strconv.Itoa(volCount),
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvc.Name,
				},
			},
		})

		volCount++
	}

	nsPrefix := i.getTestNamespacePrefix(driverType)

	replicas := int32(1)
	// Create a statefulset
	_, err = i.k8s.GetClient().AppsV1().StatefulSets(nsPrefix).Create(context.Background(), &v1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "metro-podmontest",
			Namespace: nsPrefix,
		},
		Spec: v1.StatefulSetSpec{
			Replicas:    &replicas,
			ServiceName: "2vols",
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "podmontest",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":                       "podmontest",
						"affinity":                  "affinity",
						"podmon.dellemc.com/driver": "csi-" + driverType,
					},
				},
				Spec: corev1.PodSpec{
					Affinity: &corev1.Affinity{
						NodeAffinity: &corev1.NodeAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{
								{
									Weight: 1,
									Preference: corev1.NodeSelectorTerm{
										MatchExpressions: []corev1.NodeSelectorRequirement{
											{
												Key:      preferredLabelKey,
												Operator: corev1.NodeSelectorOpIn,
												Values:   []string{preferredAffinity},
											},
										},
									},
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:            "podmontest",
							Image:           podmontestRegistry + "/podmontest:" + podmontestVersion,
							ImagePullPolicy: "IfNotPresent",
							Command: []string{
								"/podmontest",
							},
							Args: []string{
								"-doexit=true",
							},
							Env: []corev1.EnvVar{
								{
									Name:  "ROOT_DIR",
									Value: "/",
								},
							},
							VolumeMounts: volMounts,
						},
					},
					Volumes: vols,
				},
			},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("unable to create statefulset: %s", err.Error())
	}

	log.Infof("Waiting for deployed pods to be ready")
	time.Sleep(30 * time.Second)

	return nil
}

func (i *integration) iClearVolumeJournals() error {
	_, err := i.k8s.GetClient().CoreV1().RESTClient().Delete().AbsPath(customResourceDR).Name("volumejournals").DoRaw(context.Background())
	if err != nil {
		return fmt.Errorf("Unable to clean all volume journals: %s", err.Error())
	}
	return nil
}

func (i *integration) iCheckVolumeJournals(areVjs string) error {
	vjsObj := &customResource{}
	// should be either "are" or "are not"
	if areVjs != "are not" && areVjs != "are" {
		return fmt.Errorf("The passed in value must be `are` or `are not`")
	}

	vjs, err := i.k8s.GetClient().CoreV1().RESTClient().Get().AbsPath(customResourceDR).Resource("volumejournals").DoRaw(context.Background())
	if err != nil {
		return fmt.Errorf("problem happening while getting the volume journals: %s", err.Error())
	}

	err = json.Unmarshal(vjs, vjsObj)
	if err != nil {
		return fmt.Errorf("problem happening while unmarshling the volume journals: %s", err.Error())
	}

	// Check that there should be volume journals
	if areVjs == "are" && vjs != nil && len(vjsObj.Items) == 0 {
		return fmt.Errorf("volume journals are not present, but they should be")
	}

	if areVjs == "are not" && vjs != nil && len(vjsObj.Items) > 0 {
		return fmt.Errorf("volume journals are present, but they should not be")
	}

	return nil
}

func (i *integration) iCleanUpAllMetroPodsAndVolumes(driverType string) error {
	// For all metro volumes, delete and snapshots.
	for _, volume := range i.metroVolInfo {
		err := i.iDeleteSnapshotsOfMetroVolumes(volume.ID)
		if err != nil {
			return fmt.Errorf("unable to delete snapshots of Metro volumes: %s", err.Error())
		}
	}

	nsPrefix := i.getTestNamespacePrefix(driverType)

	// Delete namespace
	err := i.k8s.GetClient().CoreV1().Namespaces().Delete(context.Background(), nsPrefix, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("unable to delete namespace: %s", err.Error())
	}

	return nil
}

// blockUnblockNodeIPPort
// Drop forwarded traffic to IP and port
// eg.iptables -I FORWARD 1 -d 10.2.2.5 -p tcp --dport 443 -j DROP

func (i *integration) blockUnblockNodeIPPort(
	operation MetroConnection,
	localIPs []string,
	remoteIPs []string,
	port string,
) error {
	// Decide action
	action := "I"
	seq := "1"
	if operation == MetroConnectionRestore {
		action = "D"
		seq = ""
	}
	for _, localIP := range localIPs {
		for _, remoteIP := range remoteIPs {
			var dropPacketsCmd string
			if port != "" {
				dropPacketsCmd = fmt.Sprintf("iptables -%s FORWARD %s -j DROP -d %s -p tcp --dport %s -m comment --comment %q; ", action, seq, remoteIP, port, "metro testing; delete me")
			} else {
				dropPacketsCmd = fmt.Sprintf("iptables -%s FORWARD %s -j DROP -d %s -m comment --comment %q; ", action, remoteIP, seq, "metro testing; delete me")
			}
			log.Infof("executing %s on %s", dropPacketsCmd, localIP)
			client := i.getSSHClient(localIP)
			if _, err := i.SSHExec(client, localIP, dropPacketsCmd); err != nil {
				return fmt.Errorf("encountered an error while attempting to drop forward packets on preferred nodes: %s", err.Error())
			}
		}
	}
	return nil
}

func (i *integration) handleConnectionForNodeToArray(operation MetroConnection, storageClass string, preference string, port string) error {
	i.setStorageClass(storageClass)
	remoteArray, err := i.getNonPreferredArray(i.storageClass)
	labelKey := preferredLabelKey
	nodes, err := i.k8s.GetClient().CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{
		LabelSelector: labelKey + "=" + preference,
	})
	if err != nil {
		return fmt.Errorf("list nodes: %w", err)
	}
	var localIPs []string
	for _, node := range nodes.Items {
		// Get the node's IP address by looping over "status.addresses"
		// field in the K8s node resource and filtering by the "internalIP" type.
		var nodeIP string
		for _, address := range node.Status.Addresses {
			if address.Type == corev1.NodeInternalIP {
				nodeIP = address.Address
				localIPs = append(localIPs, nodeIP)
				break
			}
		}
	}
	var remoteIPs []string
	endpoint := remoteArray.Endpoint
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimSuffix(endpoint, "/")
	endpoint = strings.TrimSuffix(endpoint, "/api/rest")
	remoteIPs = append(remoteIPs, endpoint)
	err = i.blockUnblockNodeIPPort(operation, localIPs, remoteIPs, port)
	return err
}

func (i *integration) handleConnectionForNodeToLocalArray(operation MetroConnection, storageClass string, preference string, port string) error {
	i.setStorageClass(storageClass)
	preferredArray, err := i.getPowerStoreArrayInfo(i.storageClass.Parameters[pstoreID.KeyArrayID])
	if err != nil || preferredArray == nil {
		return fmt.Errorf("unable to get PowerStore secret: %s", err.Error())
	}
	labelKey := preferredLabelKey
	nodes, err := i.k8s.GetClient().CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{
		LabelSelector: labelKey + "=" + preference,
	})
	if err != nil {
		return fmt.Errorf("list nodes: %w", err)
	}
	var localIPs []string
	for _, node := range nodes.Items {
		// Get the node's IP address by looping over "status.addresses"
		// field in the K8s node resource and filtering by the "internalIP" type.
		var nodeIP string
		for _, address := range node.Status.Addresses {
			if address.Type == corev1.NodeInternalIP {
				nodeIP = address.Address
				localIPs = append(localIPs, nodeIP)
				break
			}
		}
	}
	var remoteIPs []string
	endpoint := preferredArray.Endpoint
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimSuffix(endpoint, "/")
	endpoint = strings.TrimSuffix(endpoint, "/api/rest")
	remoteIPs = append(remoteIPs, endpoint)
	err = i.blockUnblockNodeIPPort(operation, localIPs, remoteIPs, port)
	return err
}

// blockNodeArrayConnection utilizes iptables entries to simulate network failure between
// the non preferred storage array in a metro configuration and select worker nodes with the
// preferred=`labelValue` label.
func (i *integration) blockNodeArrayConnection(labelValue string, storageClass string) error {
	return i.handleConnectionForNodeToArray(MetroConnectionFail, storageClass, labelValue, "443")
}

// restoreNodeArrayConnection removes iptables entries added by failPreferredMetroConnection
// for worker nodes with preferred=`labelValue` label, restoring the network connection between the
// worker node and the preferred storage array in a metro configuration.
func (i *integration) restoreNodeArrayConnection(labelValue string, storageClass string) error {
	return i.handleConnectionForNodeToArray(MetroConnectionRestore, storageClass, labelValue, "443")
}

// blockNodeAndLocalArrayConnection utilizes iptables entries to simulate network failure between
// the preferred storage array in a metro configuration and select worker nodes with the
// preferred=`labelValue` label.
func (i *integration) blockNodeAndLocalArrayConnection(labelValue string, storageClass string) error {
	return i.handleConnectionForNodeToLocalArray(MetroConnectionFail, storageClass, labelValue, "443")
}

// restoreNodeAndLocalArrayConnection removes iptables entries added by blockNodeAndLocalArrayConnection
// for worker nodes with preferred=`labelValue` label, restoring the network connection between the
// worker node and the preferred storage array in a metro configuration.
func (i *integration) restoreNodeAndLocalArrayConnection(labelValue string, storageClass string) error {
	return i.handleConnectionForNodeToLocalArray(MetroConnectionRestore, storageClass, labelValue, "443")
}

func (i *integration) podIsTerminatingInTestNamespace(driverType string, label string) (bool, error) {
	namespace := fmt.Sprintf("%s%d", i.getTestNamespacePrefix(driverType), 1)
	log.Infof("Checking if pod in namespace %s is in the 'Terminating' state with label %s", namespace, label)
	pods, err := i.k8s.GetClient().CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return false, err
	}

	for _, pod := range pods.Items {
		// pod.DeletionTimestamp != nil means the pod is in the 'Terminating' state
		if pod.DeletionTimestamp != nil {
			nodeObj, err := i.k8s.GetClient().CoreV1().Nodes().Get(context.TODO(), pod.Spec.NodeName, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			if _, ok := nodeObj.ObjectMeta.Labels[label]; !ok {
				log.Infof("Pod is in the 'Terminating' state scheduled on the node with label: %s", label)
				return true, nil
			}
		}
	}
	return false, nil
}

func (i *integration) checkTerminatingPodWithLabel(driverType string, label string, wait int) error {
	terminating, err := i.podIsTerminatingInTestNamespace(driverType, label)
	if err != nil {
		return err
	}

	if terminating {
		log.Info("Pod is in the 'Terminating' state with label in test namespace.")
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
				log.Info("Timed out, but doing a last check to see if all test pods are in terminating state")
				// Check each of the test namespaces for terminating pod (final check)
				terminating, err = i.podIsTerminatingInTestNamespace(driverType, label)
				done <- true
			case <-ticker.C:
				log.Infof("Checking if all test pods are in terminating state (time left %v)", timeoutDuration-time.Since(start))
				// Check each of the test namespaces for terminating pod (final check)
				terminating, err = i.podIsTerminatingInTestNamespace(driverType, label)
				if terminating {
					done <- true
				}
			}
		}
	}()

	<-done
	timeout.Stop()
	ticker.Stop()
	log.Infof("Completed pod termination check after %v (terminating=%v)", time.Since(start), terminating)

	if !terminating {
		return fmt.Errorf("Pod is not in the 'Terminating' state")
	}
	return nil
}

func isolateIPAddress(endpoint string) string {
	// isolate the IP address for the API endpoint
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimSuffix(endpoint, "/")
	endpoint = strings.TrimSuffix(endpoint, "/api/rest")
	return endpoint
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
	context.Step(`^validate that all pods are not running within (\d+) seconds$`, i.allPodsAreNotRunningWithinSeconds)
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
	context.Step(`^finally cleanup everything except labels$`, i.finallyCleanupEverythingButLabels)
	context.Step(`^cluster is clean of test pods but may have labels$`, i.finallyCleanupEverythingButLabels)
	context.Step(`^cluster is clean of test pods$`, i.finallyCleanupEverything)
	context.Step(`^cluster is clean of test vms$`, i.finallyCleanupEverything)
	context.Step(`^test environmental variables are set$`, i.expectedEnvVariablesAreSet)
	context.Step(`^test metro environmental variables are set$`, i.expectedMetroEnvVariablesAreSet)
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
	context.Step(`^wait up to (\d+) seconds for pods to switch nodes$`, i.waitForPodsToSwitchNodes)
	context.Step(`^verify pods do not migrate for (\d+) seconds$`, i.verifyPodsDoNotMigrate)
	context.Step(`^"([^"]*)" is installed on this machine$`, i.cliToolIsInstalledOnThisMachine)
	context.Step(`^a driver namespace name "([^"]*)"$`, i.setDriverNamespaceName)
	context.Step(`^a driver secret name "([^"]*)"$`, i.setDriverSecretName)
	context.Step(`^the connection fails between the preferred metro array and the nodes with "([^"]*)" label$`, i.failPreferredMetroConnection)
	context.Step(`^the connection fails between the non preferred metro array and the nodes with "([^"]*)" label$`, i.failNonPreferredMetroConnection)
	context.Step(`^the connection is restored between the preferred metro array and the nodes with "([^"]*)" label$`, i.restorePreferredMetroConnection)
	context.Step(`^the connection is restored between the non preferred metro array and the nodes with "([^"]*)" label$`, i.restoreNonPreferredMetroConnection)
	context.Step(`^nodes with pods and with "([^"]*)" label have taint "([^"]*)" within (\d+) seconds$`, i.labeledNodesWithPodsAreTainted)
	context.Step(`^skip if "([^"]*)" is not compatible with "([^"]*)"$`, i.skipIfIsNotCompatibleWith)
	context.Step(`^I set the correct driver type to "([^"]*)"$`, i.iSetTheCorrectDriverTypeTo)
	context.Step(`^I fail non "([^"]*)" nodes with "([^"]*)" failure for (\d+) seconds$`, i.failNonpreferredNodesWithFailureForSeconds)
	context.Step(`^the arrays in storageclass "([^"]*)" are in non uniform configuration$`, i.theArraysInStorageclassAreInNonUniformConfiguration)
	context.Step(`^the arrays in storageclass "([^"]*)" are in uniform configuration$`, i.theArraysInStorageclassAreInUniformConfiguration)
	context.Step(`^there are nodes labelled "([^"]*)"$`, i.thereAreNodesLabelled)
	context.Step(`^I verify that "([^"]*)" is immediate binding$`, i.iVerifyThatIsImmediateBinding)
	context.Step(`^I deploy (\d+) on "([^"]*)" for "([^"]*)"$`, i.iDeployPvcOn)
	context.Step(`^I ensure all volumes on "([^"]*)" for "([^"]*)" are Bound$`, i.iEnsureAllVolumesOnForAreBound)
	context.Step(`^I ensure that all metro volumes on "([^"]*)" for "([^"]*)" are stable$`, i.iEnsureThatAllMetroVolumesOnForAreStable)
	context.Step(`^I ensure that all metro volumes in test namespaces on "([^"]*)" for "([^"]*)" are stable$`, i.iEnsureMetroVolumesStableInTestNamespaces)
	context.Step(`^I execute "([^"]*)" on preferred array on "([^"]*)" for "([^"]*)" for metro volumes$`, i.iExecuteMetroActionOnForForMetroVolumes)
	context.Step(`^I execute "([^"]*)" on non preferred array on "([^"]*)" for "([^"]*)" for metro volumes$`, i.iExecuteMetroActionOnNonPreferredMetroVolumes)
	context.Step(`^I soft fracture the metro volumes on "([^"]*)" for "([^"]*)"$`, i.iSoftFractureTheMetroVolumesOnFor)
	context.Step(`^I clean up all metro pods and volumes for "([^"]*)"$`, i.iCleanUpAllMetroPodsAndVolumes)
	context.Step(`^wait for (\d+) seconds$`, i.wait)
	context.Step(`^I deploy pods for all metro volumes on "([^"]*)" for "([^"]*)" with "([^"]*)" affinity$`, i.iDeployPodsForAllMetroVolumesOnFor)
	context.Step(`^I disrupt metro connectivity between arrays in storage class "([^"]*)"$`, i.disruptConnectivityBetweenMetroArrays)
	context.Step(`^I restore metro connectivity between arrays in storage class "([^"]*)"$`, i.restoreConnectivityBetweenMetroArrays)
	context.Step(`^clear out all volumejournals`, i.iClearVolumeJournals)
	context.Step(`^Check that there "([^"]*)" volumejournals$`, i.iCheckVolumeJournals)
	context.Step(`^block connection for "([^"]*)" node to remote array in "([^"]*)"$`, i.blockNodeArrayConnection)
	context.Step(`^restore connection for "([^"]*)" node to remote array in "([^"]*)"$`, i.restoreNodeArrayConnection)
	context.Step(`^block connection for "([^"]*)" node to local array in "([^"]*)"$`, i.blockNodeAndLocalArrayConnection)
	context.Step(`^restore connection for "([^"]*)" node to local array in "([^"]*)"$`, i.restoreNodeAndLocalArrayConnection)
	context.Step(`^check for terminating pod for "([^"]*)" with the "([^"]*)" label within (\d+) seconds$`, i.checkTerminatingPodWithLabel)
}
