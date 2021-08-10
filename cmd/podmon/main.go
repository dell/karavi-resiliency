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
	"fmt"
	csiext "github.com/dell/dell-csi-extensions/podmon"
	"github.com/fsnotify/fsnotify"
	"github.com/kubernetes-csi/csi-lib-utils/leaderelection"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"podmon/internal/csiapi"
	"podmon/internal/k8sapi"
	"podmon/internal/monitor"
	"strconv"
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
	driverConfigParamsDefault                = "resources/driver-config-params.yaml"
	// -- Below are constants for dynamic configuration --
	defaultLogLevel                                = log.DebugLevel
	podmonArrayConnectivityPollRate                = "PODMON_ARRAY_CONNECTIVITY_POLL_RATE"
	podmonArrayConnectivityConnectionLossThreshold = "PODMON_ARRAY_CONNECTIVITY_CONNECTION_LOSS_THRESHOLD"
	podmonControllerLogFormat                      = "PODMON_CONTROLLER_LOG_FORMAT"
	podmonControllerLogLevel                       = "PODMON_CONTROLLER_LOG_LEVEL"
	podmonNodeLogFormat                            = "PODMON_NODE_LOG_FORMAT"
	podmonNodeLogLevel                             = "PODMON_NODE_LOG_LEVEL"
	podmonSkipArrayConnectionValidation            = "PODMON_SKIP_ARRAY_CONNECTION_VALIDATION"
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

	if err := setupDynamicConfigUpdate(); err != nil {
		// There was some error with setting up the configuration update, so exit now.
		return
	}

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
		log.Infof("CSI Driver for Unity")
		monitor.Driver = new(monitor.UnityDriver)
	} else {
		log.Infof("CSI Driver for VxFlex OS")
		monitor.Driver = new(monitor.VxflexDriver)
	}
	monitor.PodmonTaintKey = fmt.Sprintf("%s.%s", monitor.Driver.GetDriverName(), monitor.PodmonTaintKeySuffix)
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
		if monitor.PodMonitor.SkipArrayConnectionValidation {
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
	driverConfigParamsFile                   *string // Set the location of the driver ConfigMap
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
		args.driverConfigParamsFile = flag.String("driver-config-params", driverConfigParamsDefault, "Full path to the YAML file containing the driver ConfigMap")
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
	*args.driverConfigParamsFile = driverConfigParamsDefault
	flag.Parse()
}

func k8sLeaderElection(runFunc func(ctx context.Context)) leaderElection {
	return leaderelection.NewLeaderElection(k8sapi.K8sClient.Client, "podmon-1", runFunc)
}

func podMonWait() bool {
	time.Sleep(10 * time.Minute)
	return false
}

// setupDynamicConfigUpdate will read the driver parameter file contain the ConfigMap. It will extract
// parameters to be set for Resiliency. It will also set up a watch against the file, so that updates
// to the file will trigger dynamic updates to Resiliency parameters.
func setupDynamicConfigUpdate() error {
	if *args.driverConfigParamsFile == "" {
		message := "--driver-config-params cannot be empty"
		log.Error(message)
		return fmt.Errorf(message)
	}

	vc := viper.New()
	vc.AutomaticEnv()
	vc.SetConfigFile(*args.driverConfigParamsFile)
	if err := vc.ReadInConfig(); err != nil {
		log.WithError(err).Errorf("unable to read driver config file: %s", *args.driverConfigParamsFile)
		return err
	}

	if err := updateConfiguration(vc); err != nil {
		log.WithError(err).Errorf("error with configuration parameters")
		return err
	}

	vc.WatchConfig()
	vc.OnConfigChange(func(in fsnotify.Event) {
		log.WithField("file", *args.driverConfigParamsFile).Infof("configuration file has changed")
		if err := updateConfiguration(vc); err != nil {
			log.Warn(err)
		}
	})

	return nil
}

// updateConfiguration is the function for reading from a ConfigMap object, extracting parameters and
// setting the appropriate Resiliency parameters. Returns error in case of issues.
func updateConfiguration(vc *viper.Viper) error {
	if *args.mode == "controller" {
		if err := setLoggingParameters(vc, podmonControllerLogFormat, podmonControllerLogLevel); err != nil {
			return err
		}
	}

	if *args.mode == "node" {
		if err := setLoggingParameters(vc, podmonNodeLogFormat, podmonNodeLogLevel); err != nil {
			return err
		}
	}

	pollRate := *args.arrayConnectivityPollRate
	if vc.IsSet(podmonArrayConnectivityPollRate) {
		pollRateStr := vc.GetString(podmonArrayConnectivityPollRate)
		value, err := strconv.Atoi(pollRateStr)
		if err != nil {
			return fmt.Errorf("parsing %s failed: value was %s", podmonArrayConnectivityPollRate,
				pollRateStr)
		}
		pollRate = value
		log.WithField(podmonArrayConnectivityPollRate, pollRate).Infof("configuration has been set.")
	}
	monitor.ArrayConnectivityPollRate = time.Duration(pollRate) * time.Second

	lossThreshold := *args.arrayConnectivityConnectionLossThreshold
	if vc.IsSet(podmonArrayConnectivityConnectionLossThreshold) {
		lossThresholdStr := vc.GetString(podmonArrayConnectivityConnectionLossThreshold)
		value, err := strconv.Atoi(lossThresholdStr)
		if err != nil {
			return fmt.Errorf("parsing %s failed: value was %s", podmonArrayConnectivityConnectionLossThreshold,
				lossThresholdStr)
		}
		lossThreshold = value
		log.WithField(podmonArrayConnectivityConnectionLossThreshold, lossThreshold).Info("configuration has been set.")
	}
	monitor.ArrayConnectivityConnectionLossThreshold = lossThreshold

	skipArrayConnectionCheck := *args.skipArrayConnectionValidation
	if vc.IsSet(podmonSkipArrayConnectionValidation) {
		skipArrayConnectionCheckStr := vc.GetString(podmonSkipArrayConnectionValidation)
		value, err := strconv.ParseBool(skipArrayConnectionCheckStr)
		if err != nil {
			return fmt.Errorf("parsing %s failed: value was %s", podmonSkipArrayConnectionValidation,
				skipArrayConnectionCheckStr)
		}
		skipArrayConnectionCheck = value
		log.WithField(podmonSkipArrayConnectionValidation, skipArrayConnectionCheck).Info("configuration has been set.")
	}
	monitor.PodMonitor.SkipArrayConnectionValidation = skipArrayConnectionCheck

	return nil
}

// setLoggingParameters is generic function for extracting logging parameters. The podmon sidecar can run in
// two different environments, controller or node mode. There are different parameters names for each
// mode, so this is a generic way to read from a parameters and set the log level and format.
func setLoggingParameters(vc *viper.Viper, formatParam, logLevelParam string) error {
	if vc.IsSet(formatParam) {
		logFormat := vc.GetString(formatParam)
		log.WithField("format", logFormat).Infof("Read %s from log configuration file", formatParam)
		if strings.EqualFold(logFormat, "json") {
			log.SetFormatter(&log.JSONFormatter{})
		} else {
			if !strings.EqualFold(logFormat, "text") {
				log.WithField("format", logFormat).Warnf("Unexpected format %s for %s. Using text format instead.", logFormat, formatParam)
			}
			log.SetFormatter(&log.TextFormatter{})
		}
	}

	level := defaultLogLevel
	if vc.IsSet(logLevelParam) {
		logLevel := vc.GetString(logLevelParam)
		if logLevel != "" {
			logLevel = strings.ToLower(logLevel)
			log.WithField("level", logLevel).Infof("Read %s from log configuration file", logLevelParam)
			var err error
			level, err = log.ParseLevel(logLevel)
			if err != nil {
				log.WithError(err).Errorf("%s %s value not recognized, setting to debug error: %s ",
					logLevelParam, logLevel, err.Error())
				log.SetLevel(defaultLogLevel)
				return fmt.Errorf("input log level %q is not valid", logLevel)
			}
		}
	}
	log.SetLevel(level)

	return nil
}
