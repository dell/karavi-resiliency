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

func TestConnectivityCollector(t *testing.T) {
	reg := prometheus.NewRegistry()
	collector := NewConnectivityCollector(ModulePowerStore)

	err := collector.Register(reg)
	require.NoError(t, err)

	// Record some connectivity checks
	collector.RecordCheck(true)
	collector.RecordCheck(false)
	collector.RecordCheck(true)

	// Collect to update success ratio
	err = collector.Collect(context.Background())
	require.NoError(t, err)

	// Verify success ratio (2 success out of 3 total)
	assert.Equal(t, 2.0/3.0, collector.GetSuccessRatio())

	// Verify metrics are registered
	metrics, err := reg.Gather()
	require.NoError(t, err)
	assert.True(t, len(metrics) > 0)
}

func TestConnectivityCollectorName(t *testing.T) {
	collector := NewConnectivityCollector(ModulePowerStore)
	assert.Equal(t, "connectivity_collector", collector.Name())
}

func TestConnectivityCollectorReset(t *testing.T) {
	reg := prometheus.NewRegistry()
	collector := NewConnectivityCollector(ModulePowerStore)

	err := collector.Register(reg)
	require.NoError(t, err)

	// Record some checks
	collector.RecordCheck(true)
	collector.RecordCheck(true)

	// Verify success ratio
	assert.Equal(t, 1.0, collector.GetSuccessRatio())

	// Reset
	collector.Reset()

	// Verify success ratio is 0 after reset
	assert.Equal(t, 0.0, collector.GetSuccessRatio())
}

func TestConnectivityCollectorPowerScale(t *testing.T) {
	reg := prometheus.NewRegistry()
	collector := NewConnectivityCollector(ModulePowerScale)

	err := collector.Register(reg)
	require.NoError(t, err)

	// Record some connectivity checks
	collector.RecordCheck(true)
	collector.RecordCheck(false)
	collector.RecordCheck(true)

	// Collect to update success ratio
	err = collector.Collect(context.Background())
	require.NoError(t, err)

	// Verify success ratio (2 success out of 3 total)
	assert.Equal(t, 2.0/3.0, collector.GetSuccessRatio())

	// Verify metrics are registered
	metrics, err := reg.Gather()
	require.NoError(t, err)
	assert.True(t, len(metrics) > 0)
}
