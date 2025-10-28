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
	"testing"
)

func TestParseCIDRList(t *testing.T) {
	tests := []struct {
		name        string
		input       []string
		wantCount   int
		wantErr     bool
		description string
	}{
		{
			name:        "valid CIDR",
			input:       []string{"10.0.0.0/8"},
			wantCount:   1,
			wantErr:     false,
			description: "Standard CIDR notation should parse correctly",
		},
		{
			name:        "valid single IP auto-adds /32",
			input:       []string{"192.168.1.1"},
			wantCount:   1,
			wantErr:     false,
			description: "Single IPv4 should auto-add /32",
		},
		{
			name:        "valid IPv6 CIDR",
			input:       []string{"::1/128"},
			wantCount:   1,
			wantErr:     false,
			description: "IPv6 CIDR notation should parse correctly",
		},
		{
			name:        "invalid CIDR",
			input:       []string{"invalid"},
			wantCount:   0,
			wantErr:     true,
			description: "Invalid CIDR string should return error",
		},
		{
			name:        "empty string skipped",
			input:       []string{"", "10.0.0.0/8", ""},
			wantCount:   1,
			wantErr:     false,
			description: "Empty strings should be skipped without error",
		},
		{
			name:        "whitespace trimmed",
			input:       []string{"  10.0.0.0/8  ", " 192.168.0.0/16 "},
			wantCount:   2,
			wantErr:     false,
			description: "Whitespace should be trimmed from CIDR strings",
		},
		{
			name:        "mixed valid/invalid returns error on first invalid",
			input:       []string{"10.0.0.0/8", "invalid", "192.168.0.0/16"},
			wantCount:   0,
			wantErr:     true,
			description: "Should return error when any CIDR is invalid",
		},
		{
			name:        "multiple valid CIDRs",
			input:       []string{"10.0.0.0/8", "192.168.0.0/16", "172.16.0.0/12"},
			wantCount:   3,
			wantErr:     false,
			description: "Multiple valid CIDRs should all parse",
		},
		{
			name:        "IPv6 single address auto-adds /128",
			input:       []string{"2001:db8::1"},
			wantCount:   1,
			wantErr:     false,
			description: "Single IPv6 should auto-add /128",
		},
		{
			name:        "empty input",
			input:       []string{},
			wantCount:   0,
			wantErr:     false,
			description: "Empty input should return empty slice without error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cidrs, err := ParseCIDRList(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCIDRList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(cidrs) != tt.wantCount {
				t.Errorf("ParseCIDRList() got %d CIDRs, want %d", len(cidrs), tt.wantCount)
			}
		})
	}
}

func TestIsIPAllowed(t *testing.T) {
	tests := []struct {
		name         string
		ip           string
		allowedCIDRs []string
		want         bool
		description  string
	}{
		{
			name:         "IP in allowed CIDR",
			ip:           "10.0.0.5",
			allowedCIDRs: []string{"10.0.0.0/8"},
			want:         true,
			description:  "IP within allowed CIDR should return true",
		},
		{
			name:         "IP not in allowed CIDR",
			ip:           "192.168.1.1",
			allowedCIDRs: []string{"10.0.0.0/8"},
			want:         false,
			description:  "IP outside allowed CIDR should return false",
		},
		{
			name:         "empty allowlist allows nothing",
			ip:           "10.0.0.1",
			allowedCIDRs: []string{},
			want:         false,
			description:  "Empty allowlist means no IPs match (caller should check AllowlistEnabled)",
		},
		{
			name:         "multiple CIDRs - matches first",
			ip:           "10.0.0.1",
			allowedCIDRs: []string{"10.0.0.0/8", "192.168.0.0/16"},
			want:         true,
			description:  "IP should match if in any of the allowed CIDRs",
		},
		{
			name:         "multiple CIDRs - matches second",
			ip:           "192.168.1.1",
			allowedCIDRs: []string{"10.0.0.0/8", "192.168.0.0/16"},
			want:         true,
			description:  "IP should match if in any of the allowed CIDRs",
		},
		{
			name:         "multiple CIDRs - matches none",
			ip:           "172.16.0.1",
			allowedCIDRs: []string{"10.0.0.0/8", "192.168.0.0/16"},
			want:         false,
			description:  "IP should not match if not in any allowed CIDR",
		},
		{
			name:         "nil IP returns false",
			ip:           "",
			allowedCIDRs: []string{"0.0.0.0/0"},
			want:         false,
			description:  "Nil IP should return false even with catch-all CIDR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cidrs, _ := ParseCIDRList(tt.allowedCIDRs)

			var ip net.IP
			if tt.ip != "" {
				ip = net.ParseIP(tt.ip)
			}

			got := IsIPAllowed(ip, cidrs)
			if got != tt.want {
				t.Errorf("IsIPAllowed(%s) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

func TestIPv4MappedIPv6(t *testing.T) {
	tests := []struct {
		name         string
		ip           string
		allowedCIDRs []string
		want         bool
		description  string
	}{
		{
			name:         "IPv4-mapped loopback matches IPv4 loopback CIDR",
			ip:           "::ffff:127.0.0.1",
			allowedCIDRs: []string{"127.0.0.0/8"},
			want:         true,
			description:  "::ffff:127.0.0.1 should normalize to 127.0.0.1 and match",
		},
		{
			name:         "IPv4-mapped private IP matches IPv4 CIDR (bypass attempt blocked)",
			ip:           "::ffff:10.0.0.1",
			allowedCIDRs: []string{"10.0.0.0/8"},
			want:         true,
			description:  "::ffff:10.0.0.1 should normalize to 10.0.0.1 and match",
		},
		{
			name:         "IPv4-mapped private IP blocked when not in allowlist",
			ip:           "::ffff:10.0.0.1",
			allowedCIDRs: []string{"192.168.0.0/16"},
			want:         false,
			description:  "IPv4-mapped 10.x.x.x should be blocked when only 192.168.x.x allowed",
		},
		{
			name:         "pure IPv6 does not match IPv4 CIDR",
			ip:           "2001:db8::1",
			allowedCIDRs: []string{"10.0.0.0/8", "192.168.0.0/16"},
			want:         false,
			description:  "Pure IPv6 address should not match IPv4 CIDRs",
		},
		{
			name:         "pure IPv6 matches IPv6 CIDR",
			ip:           "2001:db8::1",
			allowedCIDRs: []string{"2001:db8::/32"},
			want:         true,
			description:  "Pure IPv6 address should match IPv6 CIDR",
		},
		{
			name:         "IPv4-mapped metadata endpoint (bypass attempt)",
			ip:           "::ffff:169.254.169.254",
			allowedCIDRs: []string{"10.0.0.0/8"},
			want:         false,
			description:  "IPv4-mapped cloud metadata endpoint should be blocked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cidrs, _ := ParseCIDRList(tt.allowedCIDRs)
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("Failed to parse IP: %s", tt.ip)
			}

			got := IsIPAllowed(ip, cidrs)
			if got != tt.want {
				t.Errorf("IsIPAllowed(%s) = %v, want %v - %s", tt.ip, got, tt.want, tt.description)
			}
		})
	}
}

func TestSSRFAttackVectors(t *testing.T) {
	// Table-driven tests for SSRF attack payloads
	tests := []struct {
		name         string
		ip           string
		allowedCIDRs []string
		want         bool
		description  string
	}{
		// Private IP attacks
		{
			name:         "private 10.x.x.x not in allowlist",
			ip:           "10.0.0.1",
			allowedCIDRs: []string{"192.168.0.0/16"},
			want:         false,
			description:  "Private IP 10.x.x.x should be blocked when not in allowlist",
		},
		{
			name:         "private 172.16.x.x not in allowlist",
			ip:           "172.16.0.1",
			allowedCIDRs: []string{"10.0.0.0/8"},
			want:         false,
			description:  "Private IP 172.16.x.x should be blocked when not in allowlist",
		},
		{
			name:         "private 192.168.x.x not in allowlist",
			ip:           "192.168.1.1",
			allowedCIDRs: []string{"10.0.0.0/8"},
			want:         false,
			description:  "Private IP 192.168.x.x should be blocked when not in allowlist",
		},

		// Cloud metadata endpoint attacks
		{
			name:         "AWS metadata endpoint blocked",
			ip:           "169.254.169.254",
			allowedCIDRs: []string{"10.0.0.0/8"},
			want:         false,
			description:  "Cloud metadata endpoint 169.254.169.254 should be blocked",
		},
		{
			name:         "AWS metadata IPv4-mapped IPv6 blocked",
			ip:           "::ffff:169.254.169.254",
			allowedCIDRs: []string{"10.0.0.0/8"},
			want:         false,
			description:  "IPv4-mapped metadata endpoint should be blocked",
		},
		{
			name:         "link-local address blocked",
			ip:           "169.254.1.1",
			allowedCIDRs: []string{"10.0.0.0/8"},
			want:         false,
			description:  "Link-local addresses (169.254.x.x) should be blocked",
		},

		// Localhost access
		{
			name:         "localhost in allowlist",
			ip:           "127.0.0.1",
			allowedCIDRs: []string{"10.0.0.0/8", "127.0.0.0/8"},
			want:         true,
			description:  "Localhost should be allowed when in allowlist",
		},
		{
			name:         "localhost not in allowlist",
			ip:           "127.0.0.1",
			allowedCIDRs: []string{"10.0.0.0/8"},
			want:         false,
			description:  "Localhost should be blocked when not in allowlist",
		},
		{
			name:         "IPv6 localhost in allowlist",
			ip:           "::1",
			allowedCIDRs: []string{"::1/128"},
			want:         true,
			description:  "IPv6 localhost should be allowed when in allowlist",
		},

		// Public IP (baseline - should work with proper allowlist)
		{
			name:         "public IP in wide allowlist",
			ip:           "8.8.8.8",
			allowedCIDRs: []string{"0.0.0.0/0"},
			want:         true,
			description:  "Public IP should be allowed with catch-all CIDR",
		},
		{
			name:         "public IP blocked when not in allowlist",
			ip:           "8.8.8.8",
			allowedCIDRs: []string{"10.0.0.0/8"},
			want:         false,
			description:  "Public IP should be blocked when not in allowlist",
		},

		// Edge cases
		{
			name:         "0.0.0.0 blocked when not in allowlist",
			ip:           "0.0.0.0",
			allowedCIDRs: []string{"10.0.0.0/8"},
			want:         false,
			description:  "0.0.0.0 should be blocked when not in allowlist",
		},
		{
			name:         "broadcast address blocked",
			ip:           "255.255.255.255",
			allowedCIDRs: []string{"10.0.0.0/8"},
			want:         false,
			description:  "Broadcast address should be blocked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cidrs, err := ParseCIDRList(tt.allowedCIDRs)
			if err != nil {
				t.Fatalf("Failed to parse CIDRs: %v", err)
			}

			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("Failed to parse IP: %s", tt.ip)
			}

			got := IsIPAllowed(ip, cidrs)
			if got != tt.want {
				t.Errorf("IsIPAllowed(%s) = %v, want %v - %s", tt.ip, got, tt.want, tt.description)
			}
		})
	}
}

func TestIsLoopback(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{"IPv4 loopback", "127.0.0.1", true},
		{"IPv4 loopback range", "127.255.255.255", true},
		{"IPv6 loopback", "::1", true},
		{"IPv4-mapped loopback", "::ffff:127.0.0.1", true},
		{"private not loopback", "10.0.0.1", false},
		{"public not loopback", "8.8.8.8", false},
		{"empty IP", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ip net.IP
			if tt.ip != "" {
				ip = net.ParseIP(tt.ip)
			}

			got := isLoopback(ip)
			if got != tt.want {
				t.Errorf("isLoopback(%s) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}
