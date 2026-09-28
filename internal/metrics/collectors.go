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

// Package metrics provides collector-based metrics for karavi-resiliency.
package metrics

import (
	"context"
	"sync"
	"time"

	"github.com/dell/csmlog"
	"github.com/prometheus/client_golang/prometheus"
)

// ResiliencyCollector is the interface for all resiliency metrics collectors.
type ResiliencyCollector interface {
	// Collect updates metrics based on current state.
	Collect(ctx context.Context) error
	// Register registers all Prometheus descriptors with reg.
	Register(reg prometheus.Registerer) error
	// Name returns a human-readable identifier for this collector.
	Name() string
}

// ResiliencyMetricsManager manages the lifecycle of resiliency metrics collectors.
type ResiliencyMetricsManager struct {
	collectors      []ResiliencyCollector
	collectInterval time.Duration
	stopCh          chan struct{}
	wg              sync.WaitGroup
	mu              sync.RWMutex
}

// NewResiliencyMetricsManager creates a new metrics manager.
func NewResiliencyMetricsManager(interval time.Duration) *ResiliencyMetricsManager {
	return &ResiliencyMetricsManager{
		collectors:      make([]ResiliencyCollector, 0),
		collectInterval: interval,
		stopCh:          make(chan struct{}),
	}
}

// Register adds a collector to the manager.
func (m *ResiliencyMetricsManager) Register(collector ResiliencyCollector) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.collectors = append(m.collectors, collector)
	return nil
}

// Unregister removes a collector from the manager by name.
func (m *ResiliencyMetricsManager) Unregister(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, c := range m.collectors {
		if c != nil && c.Name() == name {
			m.collectors = append(m.collectors[:i], m.collectors[i+1:]...)
			return
		}
	}
}

// Start begins background collection for all registered collectors.
func (m *ResiliencyMetricsManager) Start(ctx context.Context) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, c := range m.collectors {
		if c == nil {
			continue
		}
		m.wg.Add(1)
		go m.runCollector(ctx, c)
	}
}

// Stop signals all collector goroutines to stop and waits for them to finish.
func (m *ResiliencyMetricsManager) Stop() {
	close(m.stopCh)
	m.wg.Wait()
}

// CollectAll triggers an immediate collection from all registered collectors.
func (m *ResiliencyMetricsManager) CollectAll(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, c := range m.collectors {
		if c == nil {
			continue
		}
		// Ignore collection errors to allow other collectors to continue
		_ = c.Collect(ctx)
	}
	return nil
}

// GetCollectors returns a list of registered collector names.
func (m *ResiliencyMetricsManager) GetCollectors() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.collectors))
	for _, c := range m.collectors {
		if c != nil {
			names = append(names, c.Name())
		}
	}
	return names
}

func (m *ResiliencyMetricsManager) runCollector(ctx context.Context, c ResiliencyCollector) {
	defer m.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			csmlog.WithFields(csmlog.Fields{"panic": r, "collector": c.Name()}).Error("Collector panic recovered")
		}
	}()

	// Collect immediately on start
	collectCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	_ = c.Collect(collectCtx)
	cancel()

	ticker := time.NewTicker(m.collectInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			collectCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			_ = c.Collect(collectCtx)
			cancel()
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		}
	}
}
