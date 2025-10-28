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
	"net/http"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/resource/httpadapter"
	"github.com/grafana/mcpclient/pkg/metrics"
	"github.com/grafana/mcpclient/pkg/plugin/health"
)

// Make sure App implements required interfaces. This is important to do
// since otherwise we will only get a not implemented error response from plugin in
// runtime. Plugin should not implement all these interfaces - only those which are
// required for a particular task.
var (
	_ backend.CallResourceHandler   = (*App)(nil)
	_ instancemgmt.InstanceDisposer = (*App)(nil)
	_ backend.CheckHealthHandler    = (*App)(nil)
)

// App is an example app backend plugin which can respond to data queries.
type App struct {
	backend.CallResourceHandler
	appSettings   *backend.AppInstanceSettings
	healthChecker *health.Checker
}

// NewApp creates a new example *App instance.
func NewApp(_ context.Context, settings backend.AppInstanceSettings) (instancemgmt.Instance, error) {
	// Initialize metrics first, before any other setup
	metrics.Initialize("mcpclient")

	var app App
	app.appSettings = &settings

	// Use a httpadapter (provided by the SDK) for resource calls. This allows us
	// to use a *http.ServeMux for resource calls, so we can map multiple routes
	// to CallResource without having to implement extra logic.
	mux := http.NewServeMux()
	app.registerRoutes(mux)
	app.CallResourceHandler = httpadapter.New(mux)

	// Start background health checker (60 second interval per CONTEXT.md)
	serverProvider := &mcpServerProvider{app: &app}
	connectionTester := &mcpConnectionTester{}
	app.healthChecker = health.NewChecker(60*time.Second, serverProvider, connectionTester)
	app.healthChecker.Start()

	return &app, nil
}

// Dispose here tells plugin SDK that plugin wants to clean up resources when a new instance
// created.
func (a *App) Dispose() {
	// Stop health checker gracefully
	if a.healthChecker != nil {
		a.healthChecker.Stop()
	}
}

// CheckHealth handles health checks sent from Grafana to the plugin.
func (a *App) CheckHealth(_ context.Context, _ *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	if a.healthChecker == nil {
		metrics.SetPluginUp(false)
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "health checker not initialized",
		}, nil
	}

	summary := a.healthChecker.Summary()
	enabledServers := 0
	for _, server := range getServersSnapshot() {
		if server.Enabled {
			enabledServers++
		}
	}

	if enabledServers == 0 {
		metrics.SetPluginUp(true)
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusOk,
			Message: "ok (no enabled MCP servers)",
		}, nil
	}

	if summary.LastCheck.IsZero() || summary.Enabled == 0 {
		metrics.SetPluginUp(false)
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "health checks not completed yet",
		}, nil
	}

	if summary.Disconnected > 0 || summary.Enabled < enabledServers {
		metrics.SetPluginUp(false)
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "MCP connectivity degraded: some enabled servers are disconnected",
		}, nil
	}

	metrics.SetPluginUp(true)
	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: "ok",
	}, nil
}
