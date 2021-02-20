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

package main

import (
	"context"
	"flag"
	csiext "github.com/dell/dell-csi-extensions/podmon"
	"github.com/kubernetes-csi/csi-lib-utils/leaderelection"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"podmon/internal/csiapi"
	"podmon/internal/k8sapi"
	"podmon/internal/monitor"
	"strings"
	"sync"
	"time"
)

type leaderElection interface {
	Run() error
	WithNamespace(namespace string)
}

const (
	arrayConnectivityPollRate                = 15
	arrayConnectivityConnectionLossThreshold = 3
	csisock                                  = ""
	enableLeaderElection                     = true
	kubeconfig                               = ""
	labelKey                                 = "podmon.dellemc.com/driver"
	labelValue                               = "csi-vxflexos"
	mode                                     = "controller"
	skipArrayConnectionValidation            = false
	driverPath                               = "csi-vxflexos.dellemc.com"
)

//K8sAPI is reference to the internal Kubernetes wrapper client
var K8sAPI k8sapi.K8sAPI = &k8sapi.K8sClient

//LeaderElection is a reference to function returning a leaderElection object
var LeaderElection = k8sLeaderElection

//StartAPIMonitorFn is are reference to the function that initiates the APIMonitor
var StartAPIMonitorFn = monitor.StartAPIMonitor

//StartPodMonitorFn is are reference to the function that initiates the PodMonitor
var StartPodMonitorFn = monitor.StartPodMonitor

//StartNodeMonitorFn is are reference to the function that initiates the NodeMonitor
var StartNodeMonitorFn = monitor.StartNodeMonitor

//ArrayConnMonitorFc is are reference to the function that initiates the ArrayConnectivityMonitor
var ArrayConnMonitorFc = monitor.PodMonitor.ArrayConnectivityMonitor

//PodMonWait is reference to a function that handles podmon monitoring loop
var PodMonWait = podMonWait

//GetCSIClient is reference to a function that returns a new CSIClient
var GetCSIClient = csiapi.NewCSIClient
var createArgsOnce sync.Once

func main() {
	log.SetFormatter(&log.TextFormatter{
		DisableColors:   true,
		FullTimestamp:   true,
		TimestampFormat: time.RFC1123,
	})
	getArgs()
	switch *args.mode {
	case "controller":
		monitor.PodMonitor.Mode = *args.mode
	case "node":
		monitor.PodMonitor.Mode = *args.mode
	case "standalone":
		monitor.PodMonitor.Mode = *args.mode
	default:
		log.Error("invalid mode; choose controller, node, or standalone")
		return
	}
	log.Infof("Running in %s mode", monitor.PodMonitor.Mode)
	if strings.Contains(*args.driverPath, "unity") {
		log.Infof("Unity driver")
		monitor.Driver = new(monitor.UnityDriver)
	} else {
		log.Infof("VxFlex OS driver")
		monitor.Driver = new(monitor.VxflexDriver)
	}
	monitor.ArrayConnectivityPollRate = time.Duration(*args.arrayConnectivityPollRate) * time.Second
	monitor.ArrayConnectivityConnectionLossThreshold = *args.arrayConnectivityConnectionLossThreshold
	err := K8sAPI.Connect(args.kubeconfig)
	if err != nil {
		log.Errorf("kubernetes connection error: %s", err)
		return
	}
	monitor.K8sAPI = K8sAPI
	if *args.csisock != "" {
		clientOpts := []grpc.DialOption{
			grpc.WithInsecure(),
			grpc.WithBackoffMaxDelay(time.Second),
			grpc.WithBlock(),
			grpc.WithTimeout(10 * time.Second),
		}
		log.Infof("Attempting driver connection at: %s", *args.csisock)
		monitor.CSIApi, err = GetCSIClient(*args.csisock, clientOpts...)
		defer monitor.CSIApi.Close()
		if *args.skipArrayConnectionValidation {
			monitor.PodMonitor.SkipArrayConnectionValidation = true
			log.Infof("Skipping array connection validation")
		}
		// Check if CSI Extensions are present
		req := &csiext.ValidateVolumeHostConnectivityRequest{}
		_, err := monitor.CSIApi.ValidateVolumeHostConnectivity(context.Background(), req)
		if err != nil {
			log.Errorf("Error checking presence of ValidateVolumeHostConnectivity: %s", err.Error())
		} else {
			monitor.PodMonitor.CSIExtensionsPresent = true
		}
	}
	monitor.PodMonitor.DriverPathStr = *args.driverPath
	log.Infof("PodMonitor.DriverPathStr = %s", monitor.PodMonitor.DriverPathStr)
	run := func(context.Context) {
		if *args.mode == "node" {
			err := StartAPIMonitorFn(K8sAPI, monitor.APICheckFirstTryTimeout, monitor.APICheckRetryTimeout, monitor.APICheckInterval, monitor.APIMonitorWait)
			if err != nil {
				log.Errorf("Couldn't start API monitor: %s", err.Error())
				return
			}
		} else if *args.mode == "controller" {
			if monitor.PodMonitor.CSIExtensionsPresent {
				go ArrayConnMonitorFc(monitor.ArrayConnectivityPollRate)
			}
			// monitor all the nodes with no label required
			go StartNodeMonitorFn(K8sAPI, k8sapi.K8sClient.Client, "", "", monitor.MonitorRestartTimeDelay)
		}

		// monitor the pods with the designated label key/value
		go StartPodMonitorFn(K8sAPI, k8sapi.K8sClient.Client, *args.labelKey, *args.labelValue, monitor.MonitorRestartTimeDelay)
		for {
			log.Printf("podmon alive...")
			if stop := PodMonWait(); stop {
				break
			}
		}
	}
	log.Printf("leader election: %t", *args.enableLeaderElection)
	if *args.enableLeaderElection {
		le := LeaderElection(run)
		if err := le.Run(); err != nil {
			log.Printf("failed to initialize leader election: %v", err)
		}
	} else {
		run(context.Background())
	}
}

//PodmonArgs is structure holding the podmon command arguments
type PodmonArgs struct {
	arrayConnectivityPollRate                *int    // time in seconds
	arrayConnectivityConnectionLossThreshold *int    // number of failed attempts before declaring connection loss
	csisock                                  *string // path to CSI socket
	enableLeaderElection                     *bool   // enable leader election
	kubeconfig                               *string // kubeconfig absolute path for running as stand-alone program (testing)
	labelKey                                 *string // labelKey for annotating objects to be watched/processed
	labelValue                               *string // label value for annotating objects to be watched/processed
	mode                                     *string // running mode, either "controller" for controller sidecar, "node" node sidecar, "standalone"
	skipArrayConnectionValidation            *bool   // skip the validation that array connectivity has been lost
	driverPath                               *string // driverPath to use for parsing csi.volume.kubernetes.io/nodeid annotation
}

var args PodmonArgs

func getArgs() {
	createArgsOnce.Do(func() {
		// -- Use Once so that we can run unit tests against main --
		args.arrayConnectivityPollRate = flag.Int("arrayConnectivityPollRate", arrayConnectivityPollRate, "time in seconds to poll for array connection status")
		args.arrayConnectivityConnectionLossThreshold = flag.Int("arrayConnectivityConnectionLossThreshold", arrayConnectivityConnectionLossThreshold, "number of failed connection polls to declare connection lost")
		args.csisock = flag.String("csisock", csisock, "path to csi.sock like unix:/var/run/unix.sock")
		args.enableLeaderElection = flag.Bool("leaderelection", enableLeaderElection, "boolean to enable leader election")
		args.kubeconfig = flag.String("kubeconfig", kubeconfig, "absolute path to the kubeconfig file")
		args.labelKey = flag.String("labelkey", labelKey, "label key for pods or other objects to be monitored")
		args.labelValue = flag.String("labelvalue", labelValue, "label value for pods or other objects to be monitored")
		args.mode = flag.String("mode", mode, "operating mode: controller (default), node, or standalone")
		args.skipArrayConnectionValidation = flag.Bool("skipArrayConnectionValidation", skipArrayConnectionValidation, "skip validation of array connectivity loss before killing pod")
		args.driverPath = flag.String("driverPath", driverPath, "driverPath to use for parsing csi.volume.kubernetes.io/nodeid annotation")
	})

	// -- For testing purposes. Re-default the values since main will be called multiple times --
	*args.arrayConnectivityPollRate = arrayConnectivityPollRate
	*args.arrayConnectivityConnectionLossThreshold = arrayConnectivityConnectionLossThreshold
	*args.csisock = csisock
	*args.enableLeaderElection = enableLeaderElection
	*args.kubeconfig = kubeconfig
	*args.labelKey = labelKey
	*args.labelValue = labelValue
	*args.mode = mode
	*args.skipArrayConnectionValidation = skipArrayConnectionValidation
	*args.driverPath = driverPath
	flag.Parse()
}

func k8sLeaderElection(runFunc func(ctx context.Context)) leaderElection {
	return leaderelection.NewLeaderElection(k8sapi.K8sClient.Client, "podmon-1", runFunc)
}

func podMonWait() bool {
	time.Sleep(10 * time.Minute)
	return false
}
