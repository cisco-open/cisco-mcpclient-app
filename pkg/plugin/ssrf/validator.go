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
	"strings"
)

// ParseCIDRList parses a slice of CIDR strings into a slice of *net.IPNet.
// Supports both CIDR notation (e.g., "10.0.0.0/8") and single IPs (auto-adds /32 or /128).
func ParseCIDRList(cidrStrings []string) ([]*net.IPNet, error) {
	var cidrs []*net.IPNet

	for _, cidrStr := range cidrStrings {
		cidrStr = strings.TrimSpace(cidrStr)
		if cidrStr == "" {
			continue
		}

		_, ipNet, err := net.ParseCIDR(cidrStr)
		if err != nil {
			// Try as single IP (add /32 for IPv4 or /128 for IPv6)
			ip := net.ParseIP(cidrStr)
			if ip == nil {
				return nil, fmt.Errorf("invalid CIDR or IP address %q: %w", cidrStr, err)
			}

			// Determine if IPv4 or IPv6 and create appropriate CIDR
			if ip.To4() != nil {
				_, ipNet, _ = net.ParseCIDR(cidrStr + "/32")
			} else {
				_, ipNet, _ = net.ParseCIDR(cidrStr + "/128")
			}
		}

		cidrs = append(cidrs, ipNet)
	}

	return cidrs, nil
}

// IsIPAllowed checks if the given IP is contained in any of the allowed CIDRs.
// Handles IPv4-mapped IPv6 addresses (e.g., ::ffff:127.0.0.1) by normalizing to IPv4.
func IsIPAllowed(ip net.IP, allowedCIDRs []*net.IPNet) bool {
	if ip == nil {
		return false
	}

	// Normalize IPv4-mapped IPv6 addresses to IPv4
	// This handles addresses like ::ffff:127.0.0.1
	if ip4 := ip.To4(); ip4 != nil {
		ip = ip4
	}

	for _, cidr := range allowedCIDRs {
		if cidr.Contains(ip) {
			return true
		}
	}

	return false
}

// isLoopback checks if the given IP is a loopback address (127.x.x.x or ::1).
func isLoopback(ip net.IP) bool {
	if ip == nil {
		return false
	}

	// Normalize IPv4-mapped IPv6 addresses
	if ip4 := ip.To4(); ip4 != nil {
		ip = ip4
	}

	return ip.IsLoopback()
}
