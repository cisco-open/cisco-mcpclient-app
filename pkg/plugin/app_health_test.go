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

package plugin

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/mcpclient/pkg/metrics"
	"github.com/grafana/mcpclient/pkg/plugin/health"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type staticServerProvider struct {
	servers []health.ServerInfo
}

func (p staticServerProvider) GetEnabledServers() []health.ServerInfo {
	return p.servers
}

type staticConnectionTester struct {
	err error
}

func (t staticConnectionTester) TestConnection(_ context.Context, _ health.ServerInfo) error {
	return t.err
}

func waitForSummary(t *testing.T, checker *health.Checker, wantEnabled int) {
	t.Helper()
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		s := checker.Summary()
		if !s.LastCheck.IsZero() && s.Enabled == wantEnabled {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	s := checker.Summary()
	t.Fatalf("health summary not ready: enabled=%d lastCheck=%v", s.Enabled, s.LastCheck)
}

func initTestMetrics() {
	metrics.Initialize("mcpclient")
}

func TestCheckHealth_FailsWhenCheckerMissing(t *testing.T) {
	resetTestState()
	t.Cleanup(resetTestState)
	initTestMetrics()

	app := &App{}
	res, err := app.CheckHealth(context.Background(), &backend.CheckHealthRequest{})
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, backend.HealthStatusError, res.Status)
	assert.Contains(t, res.Message, "not initialized")
}

func TestCheckHealth_OKWhenNoEnabledServers(t *testing.T) {
	resetTestState()
	t.Cleanup(resetTestState)
	initTestMetrics()

	checker := health.NewChecker(10*time.Millisecond, staticServerProvider{}, staticConnectionTester{})
	checker.Start()
	t.Cleanup(checker.Stop)
	waitForSummary(t, checker, 0)

	app := &App{healthChecker: checker}
	res, err := app.CheckHealth(context.Background(), &backend.CheckHealthRequest{})
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, backend.HealthStatusOk, res.Status)
	assert.Contains(t, res.Message, "no enabled MCP servers")
}

func TestCheckHealth_ErrWhenEnabledServerDisconnected(t *testing.T) {
	resetTestState()
	t.Cleanup(resetTestState)
	initTestMetrics()

	server := testServer("s1", "Server-1", "http://example.test")
	server.Enabled = true
	mcpServers = []MCPServer{server}

	checker := health.NewChecker(
		10*time.Millisecond,
		staticServerProvider{
			servers: []health.ServerInfo{
				{Name: server.Name, URL: server.URL},
			},
		},
		staticConnectionTester{err: errors.New("connect failed")},
	)
	checker.Start()
	t.Cleanup(checker.Stop)
	waitForSummary(t, checker, 1)

	app := &App{healthChecker: checker}
	res, err := app.CheckHealth(context.Background(), &backend.CheckHealthRequest{})
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, backend.HealthStatusError, res.Status)
	assert.Contains(t, res.Message, "degraded")
}

func TestCheckHealth_OKWhenEnabledServerConnected(t *testing.T) {
	resetTestState()
	t.Cleanup(resetTestState)
	initTestMetrics()

	server := testServer("s1", "Server-1", "http://example.test")
	server.Enabled = true
	mcpServers = []MCPServer{server}

	checker := health.NewChecker(
		10*time.Millisecond,
		staticServerProvider{
			servers: []health.ServerInfo{
				{Name: server.Name, URL: server.URL},
			},
		},
		staticConnectionTester{},
	)
	checker.Start()
	t.Cleanup(checker.Stop)
	waitForSummary(t, checker, 1)

	app := &App{healthChecker: checker}
	res, err := app.CheckHealth(context.Background(), &backend.CheckHealthRequest{})
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, backend.HealthStatusOk, res.Status)
	assert.Equal(t, "ok", res.Message)
}
