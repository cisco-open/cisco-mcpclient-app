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
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInitializeMCPMetrics_Idempotent verifies that InitializeMCPMetrics can be called
// multiple times without panicking (idempotent initialization via sync.Once).
func TestInitializeMCPMetrics_Idempotent(t *testing.T) {
	// Initialize is already called via init() in metrics.go, but calling again should be safe
	Initialize("mcpclient")

	// Verify all MCP metrics are initialized
	assert.NotNil(t, ToolCallsTotal, "ToolCallsTotal should not be nil after Initialize")
	assert.NotNil(t, ConnectionStatus, "ConnectionStatus should not be nil after Initialize")
	assert.NotNil(t, RequestLatency, "RequestLatency should not be nil after Initialize")
	assert.NotNil(t, ErrorsTotal, "ErrorsTotal should not be nil after Initialize")

	// Calling again should not panic
	assert.NotPanics(t, func() {
		Initialize("mcpclient")
	}, "Second Initialize call should not panic")
}

// TestToolCallsTotal_Recording verifies that tool_calls_total counter can record values.
func TestToolCallsTotal_Recording(t *testing.T) {
	Initialize("mcpclient")

	// Record a tool call
	ToolCallsTotal.WithLabelValues("test-server", "test-tool", "success").Inc()

	// Verify the metric is registered
	count, err := testutil.GatherAndCount(Registry, "mcpclient_tool_calls_total")
	require.NoError(t, err, "Should be able to gather tool_calls_total metric")
	assert.GreaterOrEqual(t, count, 1, "Should have at least 1 metric series")
}

// TestConnectionStatus_Recording verifies that connection_status gauge can record values.
func TestConnectionStatus_Recording(t *testing.T) {
	Initialize("mcpclient")

	// Set connection status for a test server
	ConnectionStatus.WithLabelValues("test-server-health").Set(1)

	// Verify the metric is registered
	count, err := testutil.GatherAndCount(Registry, "mcpclient_connection_status")
	require.NoError(t, err, "Should be able to gather connection_status metric")
	assert.GreaterOrEqual(t, count, 1, "Should have at least 1 metric series")

	// Verify the value is correct
	value := testutil.ToFloat64(ConnectionStatus.WithLabelValues("test-server-health"))
	assert.Equal(t, float64(1), value, "Connection status should be 1 (connected)")
}

// TestRequestLatency_Recording verifies that request_latency_seconds histogram can record values.
func TestRequestLatency_Recording(t *testing.T) {
	Initialize("mcpclient")

	// Record a latency observation
	RequestLatency.WithLabelValues("test-server-latency", "test-tool-latency").Observe(0.5)

	// Verify the metric is registered
	count, err := testutil.GatherAndCount(Registry, "mcpclient_request_latency_seconds")
	require.NoError(t, err, "Should be able to gather request_latency_seconds metric")
	assert.GreaterOrEqual(t, count, 1, "Should have at least 1 metric series")
}

// TestErrorsTotal_Recording verifies that errors_total counter can record values.
func TestErrorsTotal_Recording(t *testing.T) {
	Initialize("mcpclient")

	// Record an error
	ErrorsTotal.WithLabelValues("test_error_type").Inc()

	// Verify the metric is registered
	count, err := testutil.GatherAndCount(Registry, "mcpclient_errors_total")
	require.NoError(t, err, "Should be able to gather errors_total metric")
	assert.GreaterOrEqual(t, count, 1, "Should have at least 1 metric series")
}

// TestMCPBuckets_Values verifies the MCPBuckets histogram buckets have correct values.
func TestMCPBuckets_Values(t *testing.T) {
	// Verify MCPBuckets is not empty
	require.NotEmpty(t, MCPBuckets, "MCPBuckets should not be empty")

	// Verify expected bucket count (10 buckets: 10ms to 10s)
	assert.Len(t, MCPBuckets, 10, "MCPBuckets should have 10 buckets")

	// Verify first bucket is 0.01 (10ms)
	assert.Equal(t, 0.01, MCPBuckets[0], "First MCP bucket should be 0.01 (10ms)")

	// Verify last bucket is 10 (10 seconds)
	assert.Equal(t, 10.0, MCPBuckets[len(MCPBuckets)-1], "Last MCP bucket should be 10 (10s)")

	// Verify buckets are in ascending order
	for i := 1; i < len(MCPBuckets); i++ {
		assert.Greater(t, MCPBuckets[i], MCPBuckets[i-1],
			"MCPBuckets should be in ascending order")
	}

	// Verify specific expected values
	expectedBuckets := []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}
	assert.Equal(t, expectedBuckets, MCPBuckets, "MCPBuckets should have expected values")
}

// TestMCPMetrics_Labels verifies that metrics have the correct label names.
func TestMCPMetrics_Labels(t *testing.T) {
	Initialize("mcpclient")

	// Gather all metrics
	metrics, err := Registry.Gather()
	require.NoError(t, err, "Registry.Gather() should not error")

	// Find and verify each MCP metric
	metricLabels := map[string][]string{
		"mcpclient_tool_calls_total":         {"server", "tool", "status"},
		"mcpclient_connection_status":        {"server"},
		"mcpclient_request_latency_seconds":  {"server", "tool"},
		"mcpclient_errors_total":             {"error_type"},
	}

	for _, mf := range metrics {
		name := mf.GetName()
		if expectedLabels, ok := metricLabels[name]; ok {
			// Get actual labels from first metric in the family
			if len(mf.GetMetric()) > 0 {
				actualLabels := make([]string, 0)
				for _, lp := range mf.GetMetric()[0].GetLabel() {
					actualLabels = append(actualLabels, lp.GetName())
				}
				assert.ElementsMatch(t, expectedLabels, actualLabels,
					"Metric %s should have expected labels", name)
			}
		}
	}
}
