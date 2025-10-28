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

package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// MCP-specific metric collectors.
var (
	// ToolCallsTotal counts MCP tool calls by server, tool name, and status.
	// status is "success" or "error".
	ToolCallsTotal *prometheus.CounterVec

	// ConnectionStatus indicates the connection state for each MCP server.
	// Value 0 means disconnected, 1 means connected.
	ConnectionStatus *prometheus.GaugeVec

	// RequestLatency tracks the duration of MCP tool calls in seconds.
	RequestLatency *prometheus.HistogramVec

	// ErrorsTotal counts errors by error type.
	ErrorsTotal *prometheus.CounterVec

	// mcpOnce ensures MCP metrics are initialized only once.
	mcpOnce sync.Once
)

// MCPBuckets defines histogram buckets optimized for fast MCP API calls.
// Range: 10ms to 10s, covering typical MCP tool execution times.
var MCPBuckets = []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

// InitializeMCPMetrics registers MCP-specific metrics to the custom registry.
// It is safe to call multiple times; only the first call has effect.
// The namespace parameter is used for consistent metric naming (e.g., "grafana_mcpclient").
func InitializeMCPMetrics(namespace string) {
	mcpOnce.Do(func() {
		// ToolCallsTotal: counter for MCP tool invocations
		ToolCallsTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "tool_calls_total",
				Help:      "Total number of MCP tool calls, labeled by server, tool, and status (success/error).",
			},
			[]string{"server", "tool", "status"},
		)
		mustRegister(ToolCallsTotal)

		// ConnectionStatus: gauge for MCP server connection state
		ConnectionStatus = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "connection_status",
				Help:      "MCP server connection status (0=disconnected, 1=connected).",
			},
			[]string{"server"},
		)
		mustRegister(ConnectionStatus)

		// RequestLatency: histogram for tool call duration
		RequestLatency = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "request_latency_seconds",
				Help:      "Duration of MCP tool calls in seconds.",
				Buckets:   MCPBuckets,
			},
			[]string{"server", "tool"},
		)
		mustRegister(RequestLatency)

		// ErrorsTotal: counter for errors by type
		ErrorsTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "errors_total",
				Help:      "Total number of MCP errors, labeled by error_type.",
			},
			[]string{"error_type"},
		)
		mustRegister(ErrorsTotal)
	})
}
