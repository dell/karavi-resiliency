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

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFailoverCollector(t *testing.T) {
	reg := prometheus.NewRegistry()
	collector := NewFailoverCollector(ModulePowerStore)

	err := collector.Register(reg)
	require.NoError(t, err)

	// Record some failover operations
	collector.RecordFailover(LabelStatusSuccess, LabelReasonConnectivityLoss)
	collector.RecordFailover(LabelStatusFailure, LabelReasonNodeUnreachable)
	collector.RecordFailover(LabelStatusSuccess, LabelReasonOther)

	// Collect to update success ratio
	err = collector.Collect(context.Background())
	require.NoError(t, err)

	// Verify success ratio (2 success out of 3 total)
	assert.Equal(t, 2.0/3.0, collector.GetSuccessRatio())

	// Verify total failovers
	assert.Equal(t, int64(3), collector.GetTotalFailovers())

	// Verify metrics are registered
	metrics, err := reg.Gather()
	require.NoError(t, err)
	assert.True(t, len(metrics) > 0)
}

func TestFailoverCollectorName(t *testing.T) {
	collector := NewFailoverCollector(ModulePowerStore)
	assert.Equal(t, "failover_collector", collector.Name())
}

func TestFailoverCollectorReset(t *testing.T) {
	reg := prometheus.NewRegistry()
	collector := NewFailoverCollector(ModulePowerStore)

	err := collector.Register(reg)
	require.NoError(t, err)

	// Record some failovers
	collector.RecordFailover(LabelStatusSuccess, LabelReasonConnectivityLoss)
	collector.RecordFailover(LabelStatusFailure, LabelReasonNodeUnreachable)

	// Verify total failovers
	assert.Equal(t, int64(2), collector.GetTotalFailovers())

	// Reset
	collector.Reset()

	// Verify total failovers is 0 after reset
	assert.Equal(t, int64(0), collector.GetTotalFailovers())
}

func TestFailoverCollectorPowerScale(t *testing.T) {
	reg := prometheus.NewRegistry()
	collector := NewFailoverCollector(ModulePowerScale)

	err := collector.Register(reg)
	require.NoError(t, err)

	// Record some failover operations
	collector.RecordFailover(LabelStatusSuccess, LabelReasonConnectivityLoss)
	collector.RecordFailover(LabelStatusFailure, LabelReasonNodeUnreachable)
	collector.RecordFailover(LabelStatusSuccess, LabelReasonOther)

	// Collect to update success ratio
	err = collector.Collect(context.Background())
	require.NoError(t, err)

	// Verify success ratio (2 success out of 3 total)
	assert.Equal(t, 2.0/3.0, collector.GetSuccessRatio())

	// Verify total failovers
	assert.Equal(t, int64(3), collector.GetTotalFailovers())

	// Verify metrics are registered
	metrics, err := reg.Gather()
	require.NoError(t, err)
	assert.True(t, len(metrics) > 0)
}
