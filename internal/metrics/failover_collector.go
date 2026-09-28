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

// FailoverCollector collects failover operation metrics.
type FailoverCollector struct {
	driverLabel string

	// Metrics
	failoverTotal *prometheus.CounterVec
	successRatio  *prometheus.GaugeVec

	// State for tracking success ratio
	totalFailovers   map[string]int64 // key: status
	successFailovers map[string]int64 // key: status

	mu sync.RWMutex
}

// NewFailoverCollector creates a new FailoverCollector.
func NewFailoverCollector(driverLabel string) *FailoverCollector {
	return &FailoverCollector{
		driverLabel:      driverLabel,
		totalFailovers:   make(map[string]int64),
		successFailovers: make(map[string]int64),
	}
}

// Name returns the collector name.
func (c *FailoverCollector) Name() string {
	return "failover_collector"
}

// Register registers all Prometheus descriptors with reg.
func (c *FailoverCollector) Register(reg prometheus.Registerer) error {
	c.failoverTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: MetricFailoverTotal,
		Help: "Total failover operations performed.",
	}, []string{LabelDriver, LabelStatus, LabelReason})

	c.successRatio = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: MetricFailoverSuccessRatio,
		Help: "Failover operation success ratio.",
	}, []string{LabelDriver})

	if err := reg.Register(c.failoverTotal); err != nil {
		return fmt.Errorf("failed to register failover total: %w", err)
	}
	if err := reg.Register(c.successRatio); err != nil {
		return fmt.Errorf("failed to register failover success ratio: %w", err)
	}
	return nil
}

// Collect updates the failover success ratio metrics.
func (c *FailoverCollector) Collect(_ context.Context) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Calculate overall success ratio
	totalAll := int64(0)
	successAll := int64(0)

	for status, total := range c.totalFailovers {
		totalAll += total
		if status == LabelStatusSuccess {
			successAll = total
		}
	}

	if totalAll > 0 {
		ratio := float64(successAll) / float64(totalAll)
		c.successRatio.WithLabelValues(c.driverLabel).Set(ratio)
	}

	return nil
}

// RecordFailover records a failover operation.
func (c *FailoverCollector) RecordFailover(status, reason string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Increment total failovers for this status
	c.totalFailovers[status]++

	// Update counter with both status and reason
	c.failoverTotal.WithLabelValues(c.driverLabel, status, reason).Inc()
}

// GetSuccessRatio returns the current overall success ratio.
func (c *FailoverCollector) GetSuccessRatio() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	totalAll := int64(0)
	successAll := int64(0)

	for status, total := range c.totalFailovers {
		totalAll += total
		if status == LabelStatusSuccess {
			successAll = total
		}
	}

	if totalAll == 0 {
		return 0.0
	}
	return float64(successAll) / float64(totalAll)
}

// GetTotalFailovers returns the total number of failovers.
func (c *FailoverCollector) GetTotalFailovers() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := int64(0)
	for _, count := range c.totalFailovers {
		total += count
	}
	return total
}

// Reset resets all failover counters.
func (c *FailoverCollector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.totalFailovers = make(map[string]int64)
	c.successFailovers = make(map[string]int64)
}
