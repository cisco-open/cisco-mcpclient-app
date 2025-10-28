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

package ssrf

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"syscall"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

// NewSafeTransport creates an HTTP transport with SSRF protection.
// It uses a custom Dialer with a Control hook that validates resolved IP addresses
// at connection time, preventing DNS rebinding attacks.
func NewSafeTransport(config *SecurityConfig) *http.Transport {
	dialer := &net.Dialer{
		Control: createControlHook(config),
	}

	return &http.Transport{
		DialContext: dialer.DialContext,
	}
}

// NewSafeClient creates an HTTP client with SSRF protection.
// It uses NewSafeTransport for IP validation and disables HTTP redirects.
func NewSafeClient(config *SecurityConfig) *http.Client {
	return &http.Client{
		Transport: NewSafeTransport(config),
		Timeout:   config.RequestTimeout,
		// Disable HTTP redirects - MCP servers should not redirect
		// This prevents redirect-based SSRF bypasses
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// createControlHook returns a Dialer Control function that validates IP addresses
// at connection time against the configured allowlist.
func createControlHook(config *SecurityConfig) func(network, address string, c syscall.RawConn) error {
	return func(network, address string, c syscall.RawConn) error {
		// Only allow TCP connections
		if !strings.HasPrefix(network, "tcp") {
			log.DefaultLogger.Warn("SSRF blocked", "network", network, "reason", "non-TCP network not allowed")
			return fmt.Errorf("network %s not allowed: only TCP connections permitted", network)
		}

		// Extract IP from address (format: "ip:port")
		host, _, err := net.SplitHostPort(address)
		if err != nil {
			log.DefaultLogger.Warn("SSRF blocked", "address", address, "reason", "invalid address format")
			return fmt.Errorf("invalid address format: %w", err)
		}

		ip := net.ParseIP(host)
		if ip == nil {
			log.DefaultLogger.Warn("SSRF blocked", "host", host, "reason", "invalid IP address")
			return fmt.Errorf("invalid IP address: %s", host)
		}

		// If no allowlist configured, allow all connections
		if !config.AllowlistEnabled {
			return nil
		}

		// Check if IP is in allowed CIDRs
		if !IsIPAllowed(ip, config.AllowedCIDRs) {
			log.DefaultLogger.Warn("SSRF blocked", "ip", ip.String(), "reason", "not in allowed ranges")
			return fmt.Errorf("connection to %s blocked: IP not in allowed ranges", ip)
		}

		return nil
	}
}
