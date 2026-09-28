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

// PodMonHealthCollector collects podmon health status metrics.
type PodMonHealthCollector struct {
	driverLabel  string
	nodeName     string
	podName      string
	namespace    string
	healthyGauge *prometheus.GaugeVec
	healthStatus bool
	mu           sync.RWMutex
}

// NewPodMonHealthCollector creates a new PodMonHealthCollector.
func NewPodMonHealthCollector(driverLabel, nodeName, podName, namespace string) *PodMonHealthCollector {
	return &PodMonHealthCollector{
		driverLabel:  driverLabel,
		nodeName:     nodeName,
		podName:      podName,
		namespace:    namespace,
		healthStatus: true, // Default to healthy
	}
}

// Name returns the collector name.
func (c *PodMonHealthCollector) Name() string {
	return "podmon_health_collector"
}

// Register registers all Prometheus descriptors with reg.
func (c *PodMonHealthCollector) Register(reg prometheus.Registerer) error {
	c.healthyGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: MetricPodmonHealthy,
		Help: "Podmon health status (1=healthy, 0=unhealthy).",
	}, []string{LabelDriver, LabelNode, LabelPod, LabelNamespace})

	if err := reg.Register(c.healthyGauge); err != nil {
		return fmt.Errorf("failed to register podmon healthy gauge: %w", err)
	}
	return nil
}

// Collect updates the podmon health metric.
func (c *PodMonHealthCollector) Collect(_ context.Context) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	value := 0.0
	if c.healthStatus {
		value = 1.0
	}
	c.healthyGauge.WithLabelValues(c.driverLabel, c.nodeName, c.podName, c.namespace).Set(value)
	return nil
}

// SetHealth updates the health status.
func (c *PodMonHealthCollector) SetHealth(healthy bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.healthStatus = healthy
}

// GetHealth returns the current health status.
func (c *PodMonHealthCollector) GetHealth() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.healthStatus
}
