//  Copyright © 2021-2023 Dell Inc. or its subsidiaries. All Rights Reserved.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//       http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package main

import (
	"context"
	"flag"
	"fmt"
	"podmon/internal/csiapi"
	"podmon/internal/k8sapi"
	"podmon/internal/metrics"
	"podmon/internal/monitor"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dell/csm-metrics-common/pkg/server"
	"github.com/dell/csmlog"
	csiext "github.com/dell/dell-csi-extensions/podmon"
	"github.com/fsnotify/fsnotify"
	"github.com/kubernetes-csi/csi-lib-utils/leaderelection"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
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
	ignoreVolumelessPods                     = false
	// -- Below are constants for dynamic configuration --
	defaultLogLevel                                = csmlog.InfoLevel
	csiLogFormat                                   = "CSI_LOG_FORMAT"
	csiLogLevel                                    = "CSI_LOG_LEVEL"
	podmonArrayConnectivityPollRate                = "PODMON_ARRAY_CONNECTIVITY_POLL_RATE"
	podmonArrayConnectivityConnectionLossThreshold = "PODMON_ARRAY_CONNECTIVITY_CONNECTION_LOSS_THRESHOLD"
	podmonSkipArrayConnectionValidation            = "PODMON_SKIP_ARRAY_CONNECTION_VALIDATION"
	driverPodLabelKey                              = "driver.dellemc.com"
	driverPodLabelValue                            = "dell-storage"
	// Metrics configuration
	defaultMetricsPort = 8444
)

// K8sAPI is reference to the internal Kubernetes wrapper client
var K8sAPI k8sapi.K8sAPI = &k8sapi.K8sClient

// LeaderElection is a reference to function returning a leaderElection object
var LeaderElection = k8sLeaderElection

// StartAPIMonitorFn is are reference to the function that initiates the APIMonitor
var StartAPIMonitorFn = monitor.StartAPIMonitor

// StartPodMonitorFn is are reference to the function that initiates the PodMonitor
var StartPodMonitorFn = monitor.StartPodMonitor

// StartNodeMonitorFn is are reference to the function that initiates the NodeMonitor
var StartNodeMonitorFn = monitor.StartNodeMonitor

// ArrayConnMonitorFc is are reference to the function that initiates the ArrayConnectivityMonitor
var ArrayConnMonitorFc = monitor.PodMonitor.ArrayConnectivityMonitor

// PodMonWait is reference to a function that handles podmon monitoring loop
var PodMonWait = podMonWait

// GetCSIClient is reference to a function that returns a new CSIClient
var (
	GetCSIClient   = csiapi.NewCSIClient
	createArgsOnce sync.Once
)

// MetricsRegistry holds the Prometheus registry for metrics
var MetricsRegistry *prometheus.Registry

// MetricsServer holds the HTTP server for metrics (using csm-metrics-common)
var MetricsServer *server.MetricsServer

// ResiliencyMetrics holds the resiliency metrics instance
var ResiliencyMetrics *metrics.ResiliencyMetrics

func main() {
	getArgs()

	// Enable viper to read environment variables
	viper.AutomaticEnv()

	// Initialize metrics server if enabled
	if err := initMetricsServer(); err != nil {
		csmlog.WithFields(csmlog.Fields{"error": err}).Error("Failed to initialize metrics server")
		// Continue without metrics if initialization fails
	}

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
		csmlog.Error("invalid mode; choose controller, node, or standalone")
		return
	}
	csmlog.Infof("Running in %s mode", monitor.PodMonitor.Mode)
	switch {
	case strings.Contains(*args.driverPath, "unity"):
		csmlog.Infof("CSI Driver for Unity")
		monitor.Driver = new(monitor.UnityDriver)
	case strings.Contains(*args.driverPath, "isilon"):
		// added condition to create instance of PowerScale driver
		csmlog.Infof("CSI Driver for PowerScale")
		monitor.Driver = new(monitor.PScaleDriver)
	case strings.Contains(*args.driverPath, "powerstore"):
		csmlog.Infof("CSI Driver for PowerStore")
		monitor.Driver = new(monitor.PStoreDriver)
	case strings.Contains(*args.driverPath, "powermax"):
		csmlog.Infof("CSI Driver for PowerMax")
		monitor.Driver = new(monitor.PMaxDriver)
	default:
		csmlog.Infof("CSI Driver for VxFlex OS")
		monitor.Driver = new(monitor.VxflexDriver)
	}

	monitor.PodmonTaintKey = fmt.Sprintf("%s.%s", monitor.Driver.GetDriverName(), monitor.PodmonTaintKeySuffix)
	monitor.SetArrayConnectivityPollRate(time.Duration(*args.arrayConnectivityPollRate) * time.Second)
	monitor.ArrayConnectivityConnectionLossThreshold = *args.arrayConnectivityConnectionLossThreshold
	monitor.IgnoreVolumelessPods = *args.ignoreVolumelessPods
	err := K8sAPI.Connect(args.kubeconfig)
	if err != nil {
		csmlog.Errorf("kubernetes connection error: %s", err)
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
		csmlog.Infof("Attempting driver connection at: %s", *args.csisock)
		monitor.CSIApi, err = GetCSIClient(*args.csisock, clientOpts...)
		defer monitor.CSIApi.Close()
		if monitor.PodMonitor.SkipArrayConnectionValidation {
			csmlog.Infof("Skipping array connection validation")
		}
		// Check if CSI Extensions are present
		req := &csiext.ValidateVolumeHostConnectivityRequest{}
		_, err := monitor.CSIApi.ValidateVolumeHostConnectivity(context.Background(), req)
		if err != nil {
			csmlog.Errorf("Error checking presence of ValidateVolumeHostConnectivity: %s", err.Error())
		} else {
			monitor.PodMonitor.CSIExtensionsPresent = true
		}
	}
	monitor.PodMonitor.DriverPathStr = *args.driverPath
	csmlog.Infof("PodMonitor.DriverPathStr = %s", monitor.PodMonitor.DriverPathStr)
	run := func(context.Context) {
		if *args.mode == "node" {
			err := StartAPIMonitorFn(K8sAPI, monitor.APICheckFirstTryTimeout, monitor.APICheckRetryTimeout, monitor.APICheckInterval, monitor.APIMonitorWait)
			if err != nil {
				csmlog.Errorf("Couldn't start API monitor: %s", err.Error())
				return
			}
		} else if *args.mode == "controller" {
			if monitor.PodMonitor.CSIExtensionsPresent {
				go ArrayConnMonitorFc()
			}
			// monitor all the nodes with no label required
			go StartNodeMonitorFn(K8sAPI, k8sapi.K8sClient.Client, "", "", monitor.MonitorRestartTimeDelay)

			// monitor the driver node pods
			go StartPodMonitorFn(K8sAPI, k8sapi.K8sClient.Client, *args.driverPodLabelKey, *args.driverPodLabelValue, monitor.MonitorRestartTimeDelay)
		}

		// monitor the pods with the designated label key/value
		go StartPodMonitorFn(K8sAPI, k8sapi.K8sClient.Client, *args.labelKey, *args.labelValue, monitor.MonitorRestartTimeDelay)

		for {
			csmlog.Infof("podmon alive...")
			if stop := PodMonWait(); stop {
				break
			}
		}
	}
	csmlog.Infof("leader election: %t", *args.enableLeaderElection)
	if *args.enableLeaderElection {
		le := LeaderElection(run)
		if err := le.Run(); err != nil {
			csmlog.Errorf("failed to initialize leader election: %v", err)
		}
	} else {
		run(context.Background())
	}
}

// initMetricsServer initializes the Prometheus metrics server if X_CSI_METRICS_ENABLED is set
func initMetricsServer() error {
	metricsEnabled := viper.GetString("X_CSI_METRICS_ENABLED")
	if metricsEnabled != "true" {
		csmlog.Info("Metrics server disabled (X_CSI_METRICS_ENABLED not set to true)")
		return nil
	}

	// Get metrics port from environment variable, default to 8444
	metricsPortStr := viper.GetString("X_CSI_METRICS_PORT")
	metricsPort := fmt.Sprintf(":%d", defaultMetricsPort)
	if metricsPortStr != "" {
		port, err := strconv.Atoi(metricsPortStr)
		if err != nil {
			csmlog.WithFields(csmlog.Fields{"error": err}).Warnf("Invalid X_CSI_METRICS_PORT value: %s, using default %d", metricsPortStr, defaultMetricsPort)
		} else {
			metricsPort = fmt.Sprintf(":%d", port)
		}
	}

	// Get TLS configuration
	certFile := viper.GetString("X_CSI_METRICS_TLS_CERT_FILE")
	keyFile := viper.GetString("X_CSI_METRICS_TLS_KEY_FILE")

	// Create Prometheus registry
	MetricsRegistry = prometheus.NewRegistry()

	// Get the driver name from the driver path for the driver label
	driverPath := *args.driverPath
	driverLabel := metrics.ModulePowerStore

	if strings.Contains(driverPath, "vxflexos") {
		driverLabel = metrics.ModuleVxFlexOS
	} else if strings.Contains(driverPath, "unity") {
		driverLabel = metrics.ModuleUnity
	} else if strings.Contains(driverPath, "isilon") {
		driverLabel = metrics.ModulePowerScale
	} else if strings.Contains(driverPath, "powermax") {
		driverLabel = metrics.ModulePowerMax
	}

	// Get metrics collection interval from environment variable, default to 30 seconds
	collectionInterval := 30 * time.Second
	if collectionIntervalStr := viper.GetString("X_CSI_METRICS_COLLECTION_INTERVAL"); collectionIntervalStr != "" {
		if interval, err := time.ParseDuration(collectionIntervalStr); err == nil {
			collectionInterval = interval
		} else {
			csmlog.WithFields(csmlog.Fields{"error": err}).Warnf("Invalid X_CSI_METRICS_COLLECTION_INTERVAL value: %s, using default 30s", collectionIntervalStr)
		}
	}

	// Create resiliency metrics with collectors
	ResiliencyMetrics = metrics.NewResiliencyMetricsWithInterval(driverLabel, collectionInterval)
	csmlog.Infof("Resiliency metrics collectors created with collection interval: %v", collectionInterval)

	// Register collectors with Prometheus registry
	if err := ResiliencyMetrics.Register(MetricsRegistry); err != nil {
		return fmt.Errorf("failed to register metrics collectors: %w", err)
	}
	csmlog.Info("Resiliency metrics collectors registered")

	// Make metrics available to monitor package
	monitor.SetResiliencyMetrics(ResiliencyMetrics)

	// Set initial podmon health to healthy
	ResiliencyMetrics.SetPodmonHealth(true)
	csmlog.Infof("Initialized podmon health metric for driver: %s", driverLabel)

	// Create metrics server using csm-metrics-common
	serverCfg := server.Config{
		Port:            metricsPort,
		CertFile:        certFile,
		KeyFile:         keyFile,
		Registry:        MetricsRegistry,
		MinTLSVersion:   0, // Use default TLS12
		StaleMetricName: "dell_csm_resiliency_metrics_stale",
		StaleLabels:     []string{metrics.LabelDriver},
	}

	MetricsServer = server.NewMetricsServer(serverCfg)

	// Start metrics server in background
	ctx := context.Background()
	go func() {
		csmlog.Infof("Starting metrics server on port %s", metricsPort)
		if err := MetricsServer.Start(ctx); err != nil && ctx.Err() == nil {
			csmlog.WithFields(csmlog.Fields{"error": err}).Error("Metrics server failed")
		}
	}()

	// Start metrics collection
	ResiliencyMetrics.Start(ctx)

	csmlog.Infof("Metrics server initialized on port %s with driver: %s", metricsPort, driverLabel)
	return nil
}

// PodmonArgs is structure holding the podmon command arguments
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
	driverPodLabelKey                        *string // driverPodLabelKey for annotating driver node pods to be watched/processed
	driverPodLabelValue                      *string // driverPodLabelValue value for annotating driver node pods to be watched/processed
	ignoreVolumelessPods                     *bool   // Ignore volumeless pods even if those has Resiliency label
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
		args.driverPodLabelKey = flag.String("driverPodLabelKey", driverPodLabelKey, "label key for pods or other objects to be monitored")
		args.driverPodLabelValue = flag.String("driverPodLabelValue", driverPodLabelValue, "label value for pods or other objects to be monitored")
		args.ignoreVolumelessPods = flag.Bool("ignoreVolumelessPods", ignoreVolumelessPods, "ingnore volumeless pods even though they have podmon label")
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
	*args.driverPodLabelKey = driverPodLabelKey
	*args.driverPodLabelValue = driverPodLabelValue
	*args.ignoreVolumelessPods = ignoreVolumelessPods
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
		csmlog.Error(message)
		return fmt.Errorf("%s", message)
	}

	vc := viper.New()
	vc.AutomaticEnv()
	vc.SetConfigFile(*args.driverConfigParamsFile)
	if err := vc.ReadInConfig(); err != nil {
		csmlog.WithFields(csmlog.Fields{"error": err}).Errorf("unable to read driver config file: %s", *args.driverConfigParamsFile)
		return err
	}

	if err := updateConfiguration(vc); err != nil {
		csmlog.WithFields(csmlog.Fields{"error": err}).Errorf("error with configuration parameters")
		return err
	}

	vc.WatchConfig()
	vc.OnConfigChange(func(_ fsnotify.Event) {
		csmlog.WithFields(csmlog.Fields{"file": *args.driverConfigParamsFile}).Infof("configuration file has changed")
		if err := updateConfiguration(vc); err != nil {
			csmlog.Warnf("%v", err)
		}
	})

	return nil
}

// updateConfiguration is the function for reading from a ConfigMap object, extracting parameters and
// setting the appropriate Resiliency parameters. Returns error in case of issues.
func updateConfiguration(vc *viper.Viper) error {
	defer func() {
		message := "parameter value after config file processing"
		// Dump the values of the parameters at the end
		csmlog.WithFields(csmlog.Fields{csiLogLevel: csmlog.GetLevel()}).Info(message)
		csmlog.WithFields(csmlog.Fields{"monitor.ArrayConnectivityPollRate": monitor.GetArrayConnectivityPollRate()}).Info(message)
		csmlog.WithFields(csmlog.Fields{"monitor.ArrayConnectivityConnectionLossThreshold": monitor.ArrayConnectivityConnectionLossThreshold}).Info(message)
		csmlog.WithFields(csmlog.Fields{"monitor.PodMonitor.SkipArrayConnectionValidation": monitor.PodMonitor.SkipArrayConnectionValidation}).Info(message)
	}()

	// Read log level and format from the driver's CSI_LOG_LEVEL and CSI_LOG_FORMAT settings
	if err := setLoggingParameters(vc, csiLogFormat, csiLogLevel); err != nil {
		return err
	}

	pollRate := *args.arrayConnectivityPollRate
	if vc.IsSet(podmonArrayConnectivityPollRate) {
		pollRateStr := vc.GetString(podmonArrayConnectivityPollRate)
		value, err := strconv.Atoi(pollRateStr)
		if err != nil {
			return fmt.Errorf("parsing %s failed: value was %s", podmonArrayConnectivityPollRate,
				pollRateStr)
		}
		if value <= 0 {
			return fmt.Errorf("%s should be greater than zero, but was %d", podmonArrayConnectivityPollRate, value)
		}
		pollRate = value
		csmlog.WithFields(csmlog.Fields{podmonArrayConnectivityPollRate: pollRate}).Infof("configuration has been set.")
	}
	monitor.SetArrayConnectivityPollRate(time.Duration(pollRate) * time.Second)

	lossThreshold := *args.arrayConnectivityConnectionLossThreshold
	if vc.IsSet(podmonArrayConnectivityConnectionLossThreshold) {
		lossThresholdStr := vc.GetString(podmonArrayConnectivityConnectionLossThreshold)
		value, err := strconv.Atoi(lossThresholdStr)
		if err != nil {
			return fmt.Errorf("parsing %s failed: value was %s", podmonArrayConnectivityConnectionLossThreshold,
				lossThresholdStr)
		}
		if value <= 0 {
			return fmt.Errorf("%s should be greater than zero, but was %d", podmonArrayConnectivityConnectionLossThreshold, value)
		}
		lossThreshold = value
		csmlog.WithFields(csmlog.Fields{podmonArrayConnectivityConnectionLossThreshold: lossThreshold}).Info("configuration has been set.")
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
		csmlog.WithFields(csmlog.Fields{podmonSkipArrayConnectionValidation: skipArrayConnectionCheck}).Info("configuration has been set.")
	}
	monitor.PodMonitor.SkipArrayConnectionValidation = skipArrayConnectionCheck

	return nil
}

// setLoggingParameters reads log level and format from the driver's ConfigMap (CSI_LOG_LEVEL, CSI_LOG_FORMAT)
// and applies them to podmon's logger. This ensures podmon inherits the same logging configuration as the CSI driver.
func setLoggingParameters(vc *viper.Viper, formatParam, logLevelParam string) error {
	format := "json"
	configuredFormat := strings.ToLower(strings.TrimSpace(vc.GetString(formatParam)))
	switch configuredFormat {
	case "":
		csmlog.WithFields(csmlog.Fields{"format": format}).Infof("%s not set, using default JSON format", formatParam)
	case "json", "text":
		format = configuredFormat
	default:
		csmlog.WithFields(csmlog.Fields{"format": configuredFormat}).Warnf("Unexpected format %s for %s. Defaulting to JSON.", configuredFormat, formatParam)
	}
	csmlog.SetFormat(format)

	level := defaultLogLevel
	configuredLevel := strings.ToLower(strings.TrimSpace(vc.GetString(logLevelParam)))
	if configuredLevel == "" {
		csmlog.WithFields(csmlog.Fields{"level": defaultLogLevel.String()}).Infof("%s not set, using default INFO level", logLevelParam)
	} else {
		parsedLevel, err := csmlog.ParseLevel(configuredLevel)
		if err != nil {
			csmlog.WithFields(csmlog.Fields{"level": configuredLevel, "error": err}).Warnf("Unexpected level %s for %s. Defaulting to INFO.", configuredLevel, logLevelParam)
		} else {
			level = parsedLevel
		}
	}
	csmlog.SetLevel(level)

	return nil
}
