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

func TestResiliencyMetricsManagerCollectAll(t *testing.T) {
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

	// Register collectors with a registry to enable metric recording
	reg := prometheus.NewRegistry()
	err = healthCollector.Register(reg)
	require.NoError(t, err)
	err = connectivityCollector.Register(reg)
	require.NoError(t, err)
	err = failoverCollector.Register(reg)
	require.NoError(t, err)

	// Record some data in collectors
	connectivityCollector.RecordCheck(true)
	failoverCollector.RecordFailover(LabelStatusSuccess, LabelReasonConnectivityLoss)

	// Test CollectAll
	err = manager.CollectAll(context.Background())
	require.NoError(t, err)

	// Verify collectors were called
	assert.Equal(t, 1.0, connectivityCollector.GetSuccessRatio())
	assert.Equal(t, 1.0, failoverCollector.GetSuccessRatio())
}

func TestResiliencyMetricsManagerCollectAllWithNilCollector(t *testing.T) {
	manager := NewResiliencyMetricsManager(1 * time.Second)

	healthCollector := NewPodMonHealthCollector(ModulePowerStore, "node-1", "pod-1", "default")

	err := manager.Register(healthCollector)
	require.NoError(t, err)

	// Register collector with a registry to enable metric recording
	reg := prometheus.NewRegistry()
	err = healthCollector.Register(reg)
	require.NoError(t, err)

	// Manually add a nil collector to test handling
	manager.collectors = append(manager.collectors, nil)

	// Test CollectAll with nil collector - should not panic
	err = manager.CollectAll(context.Background())
	require.NoError(t, err)
}
