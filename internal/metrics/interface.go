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
	"os"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// ResiliencyMetrics provides a unified interface for recording resiliency metrics.
// This wraps the collector architecture to provide a simple API for the monitor package.
type ResiliencyMetrics struct {
	// Collectors
	healthCollector       *PodMonHealthCollector
	connectivityCollector *ConnectivityCollector
	failoverCollector     *FailoverCollector

	// Manager for collector lifecycle
	Manager *ResiliencyMetricsManager

	mu sync.RWMutex
}

// NewResiliencyMetrics creates a new ResiliencyMetrics instance with collectors.
func NewResiliencyMetrics(driverLabel string) *ResiliencyMetrics {
	return NewResiliencyMetricsWithInterval(driverLabel, 15*time.Second)
}

// NewResiliencyMetricsWithInterval creates a new ResiliencyMetrics instance with a custom collection interval.
func NewResiliencyMetricsWithInterval(driverLabel string, collectionInterval time.Duration) *ResiliencyMetrics {
	m := &ResiliencyMetrics{
		Manager: NewResiliencyMetricsManager(collectionInterval),
	}

	// Read pod identity from environment variables
	nodeName := os.Getenv("MY_NODE_NAME")
	podName := os.Getenv("MY_POD_NAME")
	namespace := os.Getenv("MY_POD_NAMESPACE")

	// Create collectors
	m.healthCollector = NewPodMonHealthCollector(driverLabel, nodeName, podName, namespace)
	m.connectivityCollector = NewConnectivityCollector(driverLabel)
	m.failoverCollector = NewFailoverCollector(driverLabel)

	// Register collectors with manager
	m.Manager.Register(m.healthCollector)
	m.Manager.Register(m.connectivityCollector)
	m.Manager.Register(m.failoverCollector)

	return m
}

// Register registers all collectors with the provided Prometheus registry.
func (m *ResiliencyMetrics) Register(reg prometheus.Registerer) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.healthCollector.Register(reg); err != nil {
		return err
	}
	if err := m.connectivityCollector.Register(reg); err != nil {
		return err
	}
	if err := m.failoverCollector.Register(reg); err != nil {
		return err
	}
	return nil
}

// Start begins background metrics collection.
func (m *ResiliencyMetrics) Start(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Manager.Start(ctx)
}

// Stop stops background metrics collection.
func (m *ResiliencyMetrics) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Manager.Stop()
}

// SetPodmonHealth sets the podmon health status.
func (m *ResiliencyMetrics) SetPodmonHealth(healthy bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.healthCollector.SetHealth(healthy)
}

// GetPodmonHealth returns the current podmon health status.
func (m *ResiliencyMetrics) GetPodmonHealth() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.healthCollector.GetHealth()
}

// RecordConnectivityCheck records a connectivity check.
func (m *ResiliencyMetrics) RecordConnectivityCheck(success bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.connectivityCollector.RecordCheck(success)
}

// GetConnectivitySuccessRatio returns the success ratio.
func (m *ResiliencyMetrics) GetConnectivitySuccessRatio() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connectivityCollector.GetSuccessRatio()
}

// RecordFailover records a failover operation.
func (m *ResiliencyMetrics) RecordFailover(status, reason string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.failoverCollector.RecordFailover(status, reason)
}

// GetFailoverSuccessRatio returns the overall failover success ratio.
func (m *ResiliencyMetrics) GetFailoverSuccessRatio() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.failoverCollector.GetSuccessRatio()
}

// GetTotalFailovers returns the total number of failovers.
func (m *ResiliencyMetrics) GetTotalFailovers() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.failoverCollector.GetTotalFailovers()
}

// RecordCleanupOperation records a pod cleanup operation as a failover event.
// Maps cleanup reason to failover reason label.
func (m *ResiliencyMetrics) RecordCleanupOperation(cleanupReason string, success bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Map cleanup reason to failover reason label
	failoverReason := LabelReasonOther
	if cleanupReason == "ArrayConnectivityLoss" {
		failoverReason = LabelReasonConnectivityLoss
	} else if cleanupReason == "NodeUnreachable" || cleanupReason == "NodeFailure" {
		failoverReason = LabelReasonNodeUnreachable
	}

	// Determine status
	failoverStatus := LabelStatusFailure
	if success {
		failoverStatus = LabelStatusSuccess
	}

	m.failoverCollector.RecordFailover(failoverStatus, failoverReason)
}
