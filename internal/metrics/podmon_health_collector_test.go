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

func TestPodMonHealthCollector(t *testing.T) {
	reg := prometheus.NewRegistry()
	collector := NewPodMonHealthCollector(ModulePowerStore, "node-1", "pod-1", "default")

	err := collector.Register(reg)
	require.NoError(t, err)

	// Test initial health (default healthy)
	err = collector.Collect(context.Background())
	require.NoError(t, err)

	// Verify metric is registered
	metrics, err := reg.Gather()
	require.NoError(t, err)
	assert.True(t, len(metrics) > 0)

	// Test setting unhealthy
	collector.SetHealth(false)
	err = collector.Collect(context.Background())
	require.NoError(t, err)

	assert.False(t, collector.GetHealth())

	// Test setting healthy
	collector.SetHealth(true)
	err = collector.Collect(context.Background())
	require.NoError(t, err)

	assert.True(t, collector.GetHealth())
}

func TestPodMonHealthCollectorName(t *testing.T) {
	collector := NewPodMonHealthCollector(ModulePowerStore, "node-1", "pod-1", "default")
	assert.Equal(t, "podmon_health_collector", collector.Name())
}

func TestPodMonHealthCollectorPowerScale(t *testing.T) {
	reg := prometheus.NewRegistry()
	collector := NewPodMonHealthCollector(ModulePowerScale, "node-1", "pod-1", "default")

	err := collector.Register(reg)
	require.NoError(t, err)

	// Test initial health (default healthy)
	err = collector.Collect(context.Background())
	require.NoError(t, err)

	// Verify metric is registered
	metrics, err := reg.Gather()
	require.NoError(t, err)
	assert.True(t, len(metrics) > 0)

	// Test setting unhealthy
	collector.SetHealth(false)
	err = collector.Collect(context.Background())
	require.NoError(t, err)

	assert.False(t, collector.GetHealth())

	// Test setting healthy
	collector.SetHealth(true)
	err = collector.Collect(context.Background())
	require.NoError(t, err)

	assert.True(t, collector.GetHealth())
}
