/*
 *
 * Copyright © 2024 Dell Inc. or its subsidiaries. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

// Package metrics provides naming constants for karavi-resiliency metrics.
package metrics

// Resiliency metric name constants
const (
	// PodMon health metrics
	MetricPodmonHealthy = "dell_csm_resiliency_podmon_healthy"

	// Connectivity check metrics
	MetricConnectivityCheckTotal   = "dell_csm_resiliency_connectivity_check_total"
	MetricConnectivitySuccessRatio = "dell_csm_resiliency_connectivity_success_ratio"

	// Failover operation metrics
	MetricFailoverTotal        = "dell_csm_resiliency_failover_total"
	MetricFailoverSuccessRatio = "dell_csm_resiliency_failover_success_ratio"
)

// Label key constants for resiliency metrics
const (
	LabelDriver    = "driver"
	LabelNode      = "node"
	LabelPod       = "pod"
	LabelNamespace = "namespace"
	LabelStatus    = "status"
	LabelReason    = "reason"
)

// Label value constants
const (
	LabelStatusSuccess = "success"
	LabelStatusFailure = "failure"
	LabelStatusUnknown = "unknown"

	// Reason values for failover operations
	LabelReasonConnectivityLoss = "connectivity_loss"
	LabelReasonNodeUnreachable  = "node_unreachable"
	LabelReasonOther            = "other"

	// Module values (matching CSI driver names)
	ModulePowerStore = "csi-powerstore"
	ModuleVxFlexOS   = "csi-vxflexos"
	ModuleUnity      = "csi-unity"
	ModulePowerScale = "csi-powerscale"
	ModulePowerMax   = "csi-powermax"
)
