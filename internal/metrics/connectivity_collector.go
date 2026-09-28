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
	"fmt"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// ConnectivityCollector collects volume connectivity check metrics.
type ConnectivityCollector struct {
	driverLabel string

	// Metrics
	checkTotal   *prometheus.CounterVec
	successRatio *prometheus.GaugeVec

	// State for tracking success ratio
	totalChecks   int64
	successChecks int64

	mu sync.RWMutex
}

// NewConnectivityCollector creates a new ConnectivityCollector.
func NewConnectivityCollector(driverLabel string) *ConnectivityCollector {
	return &ConnectivityCollector{
		driverLabel: driverLabel,
	}
}

// Name returns the collector name.
func (c *ConnectivityCollector) Name() string {
	return "connectivity_collector"
}

// Register registers all Prometheus descriptors with reg.
func (c *ConnectivityCollector) Register(reg prometheus.Registerer) error {
	c.checkTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: MetricConnectivityCheckTotal,
		Help: "Total connectivity checks performed.",
	}, []string{LabelDriver})

	c.successRatio = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: MetricConnectivitySuccessRatio,
		Help: "Connectivity check success ratio.",
	}, []string{LabelDriver})

	if err := reg.Register(c.checkTotal); err != nil {
		return fmt.Errorf("failed to register connectivity check total: %w", err)
	}
	if err := reg.Register(c.successRatio); err != nil {
		return fmt.Errorf("failed to register connectivity success ratio: %w", err)
	}
	return nil
}

// Collect updates the connectivity success ratio metrics.
func (c *ConnectivityCollector) Collect(_ context.Context) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Update success ratio
	if c.totalChecks > 0 {
		ratio := float64(c.successChecks) / float64(c.totalChecks)
		c.successRatio.WithLabelValues(c.driverLabel).Set(ratio)
	}

	return nil
}

// RecordCheck records a connectivity check.
func (c *ConnectivityCollector) RecordCheck(success bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Increment total checks
	c.totalChecks++

	// Increment success checks if successful
	if success {
		c.successChecks++
	}

	// Update counter
	c.checkTotal.WithLabelValues(c.driverLabel).Inc()
}

// GetSuccessRatio returns the current success ratio.
func (c *ConnectivityCollector) GetSuccessRatio() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.totalChecks == 0 {
		return 0.0
	}
	return float64(c.successChecks) / float64(c.totalChecks)
}

// Reset resets the counters.
func (c *ConnectivityCollector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.totalChecks = 0
	c.successChecks = 0
}
