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
	"net"
	"os"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

const (
	// EnvAllowedHosts is the environment variable for allowed CIDR ranges
	EnvAllowedHosts = "GF_PLUGIN_MCPCLIENT_ALLOWED_HOSTS"

	// EnvRequestTimeout is the environment variable for HTTP request timeout
	EnvRequestTimeout = "GF_PLUGIN_MCPCLIENT_REQUEST_TIMEOUT"

	// DefaultRequestTimeout is the default HTTP request timeout
	DefaultRequestTimeout = 30 * time.Second
)

// SecurityConfig holds SSRF protection configuration.
type SecurityConfig struct {
	// AllowedCIDRs contains the parsed CIDR ranges that connections are allowed to.
	// If empty and AllowlistEnabled is false, all connections are allowed.
	AllowedCIDRs []*net.IPNet

	// RequestTimeout is the HTTP request timeout.
	RequestTimeout time.Duration

	// AllowlistEnabled indicates whether an allowlist is configured.
	// When true, only IPs in AllowedCIDRs are allowed.
	// When false, all connections are allowed (maximum flexibility).
	AllowlistEnabled bool
}

// LoadSecurityConfig loads security configuration from environment variables.
// - GF_PLUGIN_MCPCLIENT_ALLOWED_HOSTS: comma-separated CIDRs (optional)
// - GF_PLUGIN_MCPCLIENT_REQUEST_TIMEOUT: duration string (optional, default "30s")
func LoadSecurityConfig() (*SecurityConfig, error) {
	config := &SecurityConfig{
		RequestTimeout: DefaultRequestTimeout,
	}

	// Parse allowed hosts
	allowedHosts := os.Getenv(EnvAllowedHosts)
	if allowedHosts != "" {
		// Split comma-separated CIDRs
		cidrStrings := strings.Split(allowedHosts, ",")

		cidrs, err := ParseCIDRList(cidrStrings)
		if err != nil {
			return nil, err
		}

		// Auto-add localhost ranges when allowlist is configured (per CONTEXT.md)
		loopback4, _ := ParseCIDRList([]string{"127.0.0.0/8"})
		loopback6, _ := ParseCIDRList([]string{"::1/128"})

		cidrs = append(cidrs, loopback4...)
		cidrs = append(cidrs, loopback6...)

		config.AllowedCIDRs = cidrs
		config.AllowlistEnabled = true
	}

	// Parse request timeout
	timeoutStr := os.Getenv(EnvRequestTimeout)
	if timeoutStr != "" {
		timeout, err := time.ParseDuration(timeoutStr)
		if err != nil {
			log.DefaultLogger.Warn("Invalid request timeout, using default",
				"value", timeoutStr,
				"default", DefaultRequestTimeout.String(),
				"error", err.Error())
		} else {
			config.RequestTimeout = timeout
		}
	}

	// Log loaded configuration
	log.DefaultLogger.Info("SSRF protection loaded",
		"allowlistEnabled", config.AllowlistEnabled,
		"cidrsCount", len(config.AllowedCIDRs),
		"timeout", config.RequestTimeout.String())

	return config, nil
}

// DefaultConfig returns a SecurityConfig with no allowlist and default timeout.
// Useful for testing or as a fallback configuration.
func DefaultConfig() *SecurityConfig {
	return &SecurityConfig{
		AllowedCIDRs:     nil,
		RequestTimeout:   DefaultRequestTimeout,
		AllowlistEnabled: false,
	}
}
