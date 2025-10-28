// Copyright 2025 Cisco Systems, Inc. and its affiliates
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

// Package metrics provides Prometheus metrics infrastructure for the MCP Client plugin.
// It exposes Go runtime metrics, process metrics, and a plugin health gauge.
package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// Package-level variables for metrics registry and collectors.
var (
	// Registry is the gatherer used for tests and internal metric inspection.
	// Metrics are registered on Prometheus default registerer so Grafana can
	// collect them via the plugin SDK diagnostics (CollectMetrics) path.
	Registry = prometheus.DefaultGatherer

	// registerer is the metrics registerer used by this plugin.
	registerer = prometheus.DefaultRegisterer

	// PluginUp is a gauge indicating plugin health status.
	// Value 1 indicates the plugin is healthy, 0 indicates unhealthy.
	PluginUp prometheus.Gauge

	// once ensures Initialize is only executed once.
	once sync.Once

	initialized bool
)

// LLMBuckets defines histogram buckets optimized for LLM response times.
// These buckets cover the typical range of LLM API latencies from sub-second
// to multi-minute responses for complex operations.
var LLMBuckets = []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60, 120}

// Initialize sets up the Prometheus registry with standard collectors.
// It is safe to call multiple times; only the first call has effect.
func Initialize(namespace string) {
	once.Do(func() {
		// Register baseline runtime/process/build collectors on default registry.
		registerIfMissing(prometheus.NewGoCollector())
		registerIfMissing(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
		registerIfMissing(prometheus.NewBuildInfoCollector())

		// Create and register the plugin health gauge
		PluginUp = prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "plugin_up",
			Help:      "Indicates whether the MCP Client plugin is healthy (1) or unhealthy (0).",
		})
		mustRegister(PluginUp)

		// Set initial value to healthy
		PluginUp.Set(1)

		// Initialize MCP-specific metrics
		InitializeMCPMetrics(namespace)
		initialized = true
	})
}

// IsInitialized returns true if the metrics registry has been initialized.
func IsInitialized() bool {
	return initialized
}

// SetPluginUp updates the plugin health gauge.
func SetPluginUp(isHealthy bool) {
	if PluginUp == nil {
		return
	}
	if isHealthy {
		PluginUp.Set(1)
		return
	}
	PluginUp.Set(0)
}

func mustRegister(collector prometheus.Collector) {
	if err := registerer.Register(collector); err != nil {
		if _, exists := err.(prometheus.AlreadyRegisteredError); exists {
			return
		}
		panic(err)
	}
}

func registerIfMissing(collector prometheus.Collector) {
	if err := registerer.Register(collector); err != nil {
		if _, exists := err.(prometheus.AlreadyRegisteredError); exists {
			return
		}
		panic(err)
	}
}
