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

// Package health provides background health checking for MCP servers.
package health

import (
	"context"
	"sync"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/mcpclient/pkg/metrics"
)

// ServerInfo contains the information needed to connect to an MCP server.
type ServerInfo struct {
	Name      string
	URL       string
	AuthType  string
	AuthToken string
	AuthUser  string
	AuthPass  string
}

// ServerProvider provides access to enabled MCP servers for health checking.
type ServerProvider interface {
	GetEnabledServers() []ServerInfo
}

// ConnectionTester tests connectivity to an MCP server.
// This interface allows for dependency injection in testing.
type ConnectionTester interface {
	TestConnection(ctx context.Context, server ServerInfo) error
}

// Checker performs periodic health checks on MCP servers.
type Checker struct {
	interval time.Duration
	servers  ServerProvider
	tester   ConnectionTester
	cancel   context.CancelFunc
	wg       sync.WaitGroup

	// statusCache tracks previous connection status for change detection
	statusCache map[string]bool
	lastCheck   time.Time
	mu          sync.RWMutex
}

// Summary provides the latest aggregated health snapshot across enabled servers.
type Summary struct {
	Enabled      int
	Connected    int
	Disconnected int
	LastCheck    time.Time
}

// NewChecker creates a new health checker with the given interval and server provider.
// The checker does not start automatically; call Start() to begin health checks.
func NewChecker(interval time.Duration, servers ServerProvider, tester ConnectionTester) *Checker {
	return &Checker{
		interval:    interval,
		servers:     servers,
		tester:      tester,
		statusCache: make(map[string]bool),
	}
}

// Start begins the background health check loop.
// It performs an initial check immediately, then checks at the configured interval.
func (c *Checker) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel

	c.wg.Add(1)
	go c.run(ctx)

	log.DefaultLogger.Info("Health checker started", "interval", c.interval.String())
}

// Stop gracefully stops the health checker and waits for the goroutine to exit.
func (c *Checker) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
	c.wg.Wait()
	log.DefaultLogger.Info("Health checker stopped")
}

// run is the main health check loop.
func (c *Checker) run(ctx context.Context) {
	defer c.wg.Done()

	// Perform initial check immediately
	c.checkAll(ctx)

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.checkAll(ctx)
		}
	}
}

// checkAll checks the connection status of all enabled servers.
func (c *Checker) checkAll(ctx context.Context) {
	servers := c.servers.GetEnabledServers()
	seen := make(map[string]struct{}, len(servers))

	for _, server := range servers {
		seen[server.Name] = struct{}{}
		c.checkServer(ctx, server)
	}

	c.mu.Lock()
	for serverName := range c.statusCache {
		if _, exists := seen[serverName]; !exists {
			delete(c.statusCache, serverName)
		}
	}
	c.lastCheck = time.Now()
	healthy := true
	for _, connected := range c.statusCache {
		if !connected {
			healthy = false
			break
		}
	}
	c.mu.Unlock()
	metrics.SetPluginUp(healthy)
}

// checkServer tests connectivity to a single server and updates metrics.
func (c *Checker) checkServer(ctx context.Context, server ServerInfo) {
	// Create timeout context for connection test (10 seconds)
	testCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Test connection
	err := c.tester.TestConnection(testCtx, server)

	// Determine connection status
	connected := err == nil
	statusValue := float64(0)
	if connected {
		statusValue = 1
	}

	// Update connection_status metric
	metrics.ConnectionStatus.WithLabelValues(server.Name).Set(statusValue)

	// Check for status change and log at warn level
	c.mu.Lock()
	previousStatus, exists := c.statusCache[server.Name]
	if !exists || previousStatus != connected {
		// Status changed or first check
		if connected {
			log.DefaultLogger.Warn("MCP server connection status changed", "server", server.Name, "status", "connected")
		} else {
			log.DefaultLogger.Warn("MCP server connection status changed", "server", server.Name, "status", "disconnected", "error", err)
		}
		c.statusCache[server.Name] = connected
	}
	c.mu.Unlock()
}

// Summary returns the latest connection summary from background health checks.
func (c *Checker) Summary() Summary {
	c.mu.RLock()
	defer c.mu.RUnlock()

	summary := Summary{
		Enabled:   len(c.statusCache),
		LastCheck: c.lastCheck,
	}
	for _, connected := range c.statusCache {
		if connected {
			summary.Connected++
		} else {
			summary.Disconnected++
		}
	}
	return summary
}
