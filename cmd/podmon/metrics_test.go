//  Copyright © 2025-2026 Dell Inc. or its subsidiaries. All Rights Reserved.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//       http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"podmon/internal/metrics"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resFreePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	p := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return p
}

// /metrics returns dell_csm_resiliency_* metrics.
func TestIntegration_RES_ResiliencyMetricsScrape(t *testing.T) {
	t.Setenv("X_CSI_METRICS_ENABLED", "true")

	reg := prometheus.NewRegistry()

	// Use the new collector-based metrics
	resiliencyMetrics := metrics.NewResiliencyMetrics("csi-powerstore")
	err := resiliencyMetrics.Register(reg)
	require.NoError(t, err)

	// Record some sample data
	resiliencyMetrics.SetPodmonHealth(true)
	resiliencyMetrics.RecordConnectivityCheck(true)
	resiliencyMetrics.RecordConnectivityCheck(false)
	resiliencyMetrics.RecordFailover("success", "connectivity_loss")
	resiliencyMetrics.RecordFailover("failure", "node_unreachable")

	// Trigger collection to update gauges
	ctx := context.Background()
	err = resiliencyMetrics.Manager.CollectAll(ctx)
	require.NoError(t, err)

	port := resFreePort(t)
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	srv := &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: mux}
	go func() { _ = srv.ListenAndServe() }()
	t.Cleanup(func() { _ = srv.Shutdown(context.Background()) })
	time.Sleep(80 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/metrics", port))
	require.NoError(t, err)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	assert.Equal(t, http.StatusOK, resp.StatusCode, "metrics endpoint must return HTTP 200")
	assert.Contains(t, string(body), "dell_csm_resiliency_podmon_healthy",
		"scrape must contain podmon health metric")
	assert.Contains(t, string(body), "driver=\"csi-powerstore\"",
		"scrape must contain driver label")
	assert.Contains(t, string(body), "dell_csm_resiliency_connectivity_check_total",
		"scrape must contain connectivity check counter")
	assert.Contains(t, string(body), "dell_csm_resiliency_connectivity_success_ratio",
		"scrape must contain connectivity success ratio gauge")
	assert.Contains(t, string(body), "dell_csm_resiliency_failover_total",
		"scrape must contain failover total counter")
	assert.Contains(t, string(body), `reason="connectivity_loss"`,
		"scrape must contain failover reason label")
	assert.Contains(t, string(body), "dell_csm_resiliency_failover_success_ratio",
		"scrape must contain failover success ratio gauge")
}
