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

package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResiliencyMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := NewResiliencyMetrics(ModulePowerStore)

	err := metrics.Register(reg)
	require.NoError(t, err)

	// Test podmon health
	metrics.SetPodmonHealth(true)
	assert.True(t, metrics.GetPodmonHealth())

	metrics.SetPodmonHealth(false)
	assert.False(t, metrics.GetPodmonHealth())

	// Test connectivity checks
	metrics.RecordConnectivityCheck(true)
	metrics.RecordConnectivityCheck(false)
	assert.Equal(t, 0.5, metrics.GetConnectivitySuccessRatio())

	// Test failover operations
	metrics.RecordFailover(LabelStatusSuccess, LabelReasonConnectivityLoss)
	metrics.RecordFailover(LabelStatusFailure, LabelReasonNodeUnreachable)
	assert.Equal(t, 0.5, metrics.GetFailoverSuccessRatio())
	assert.Equal(t, int64(2), metrics.GetTotalFailovers())

	// Test starting and stopping metrics collection
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	metrics.Start(ctx)
	time.Sleep(50 * time.Millisecond) // Let collection run briefly
	metrics.Stop()
}

func TestResiliencyMetricsManager(t *testing.T) {
	manager := NewResiliencyMetricsManager(1 * time.Second)

	healthCollector := NewPodMonHealthCollector(ModulePowerStore, "node-1", "pod-1", "default")
	connectivityCollector := NewConnectivityCollector(ModulePowerStore)
	failoverCollector := NewFailoverCollector(ModulePowerStore)

	err := manager.Register(healthCollector)
	require.NoError(t, err)

	err = manager.Register(connectivityCollector)
	require.NoError(t, err)

	err = manager.Register(failoverCollector)
	require.NoError(t, err)

	// Test collector names
	collectors := manager.GetCollectors()
	assert.Contains(t, collectors, "podmon_health_collector")
	assert.Contains(t, collectors, "connectivity_collector")
	assert.Contains(t, collectors, "failover_collector")

	// Test unregister
	manager.Unregister("connectivity_collector")
	collectors = manager.GetCollectors()
	assert.NotContains(t, collectors, "connectivity_collector")
	assert.Contains(t, collectors, "podmon_health_collector")
	assert.Contains(t, collectors, "failover_collector")
}

func TestResiliencyMetricsPowerScale(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := NewResiliencyMetrics(ModulePowerScale)

	err := metrics.Register(reg)
	require.NoError(t, err)

	// Test podmon health
	metrics.SetPodmonHealth(true)
	assert.True(t, metrics.GetPodmonHealth())

	metrics.SetPodmonHealth(false)
	assert.False(t, metrics.GetPodmonHealth())

	// Test connectivity checks
	metrics.RecordConnectivityCheck(true)
	metrics.RecordConnectivityCheck(false)
	assert.Equal(t, 0.5, metrics.GetConnectivitySuccessRatio())

	// Test failover operations
	metrics.RecordFailover(LabelStatusSuccess, LabelReasonConnectivityLoss)
	metrics.RecordFailover(LabelStatusFailure, LabelReasonNodeUnreachable)
	assert.Equal(t, 0.5, metrics.GetFailoverSuccessRatio())
	assert.Equal(t, int64(2), metrics.GetTotalFailovers())

	// Test starting and stopping metrics collection
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	metrics.Start(ctx)
	time.Sleep(50 * time.Millisecond) // Let collection run briefly
	metrics.Stop()
}

func TestResiliencyMetricsRecordCleanupOperation(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := NewResiliencyMetrics(ModulePowerStore)

	err := metrics.Register(reg)
	require.NoError(t, err)

	// Test ArrayConnectivityLoss cleanup
	metrics.RecordCleanupOperation("ArrayConnectivityLoss", true)
	assert.Equal(t, 1.0, metrics.GetFailoverSuccessRatio())
	assert.Equal(t, int64(1), metrics.GetTotalFailovers())

	// Test NodeUnreachable cleanup
	metrics.RecordCleanupOperation("NodeUnreachable", false)
	assert.Equal(t, 0.5, metrics.GetFailoverSuccessRatio())
	assert.Equal(t, int64(2), metrics.GetTotalFailovers())

	// Test NodeFailure cleanup
	metrics.RecordCleanupOperation("NodeFailure", true)
	assert.Equal(t, 2.0/3.0, metrics.GetFailoverSuccessRatio())
	assert.Equal(t, int64(3), metrics.GetTotalFailovers())

	// Test other cleanup reason
	metrics.RecordCleanupOperation("OtherReason", true)
	assert.Equal(t, 3.0/4.0, metrics.GetFailoverSuccessRatio())
	assert.Equal(t, int64(4), metrics.GetTotalFailovers())
}
