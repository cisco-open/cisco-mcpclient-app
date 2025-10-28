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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewSafeTransport(t *testing.T) {
	config := DefaultConfig()

	transport := NewSafeTransport(config)

	if transport == nil {
		t.Fatal("NewSafeTransport returned nil")
	}

	// Verify transport has DialContext configured
	if transport.DialContext == nil {
		t.Error("Transport should have DialContext configured")
	}
}

func TestNewSafeClient(t *testing.T) {
	config := DefaultConfig()
	config.RequestTimeout = 10 * time.Second

	client := NewSafeClient(config)

	if client == nil {
		t.Fatal("NewSafeClient returned nil")
	}

	// Verify client has configured timeout
	if client.Timeout != 10*time.Second {
		t.Errorf("Client timeout = %v, want %v", client.Timeout, 10*time.Second)
	}

	// Verify client has Transport
	if client.Transport == nil {
		t.Error("Client should have Transport configured")
	}

	// Verify client has CheckRedirect
	if client.CheckRedirect == nil {
		t.Error("Client should have CheckRedirect configured")
	}
}

func TestRedirectBlocking(t *testing.T) {
	// Create a test server that returns a redirect
	redirectTarget := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("redirected"))
	}))
	defer redirectTarget.Close()

	// Create a server that redirects to the target
	redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, redirectTarget.URL, http.StatusFound)
	}))
	defer redirectServer.Close()

	// Use default config (no allowlist - allows all)
	config := DefaultConfig()
	client := NewSafeClient(config)

	// Make request to redirect server
	resp, err := client.Get(redirectServer.URL)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	// SafeClient should NOT follow redirect - returns original 302 response
	if resp.StatusCode != http.StatusFound {
		t.Errorf("Expected status %d (302 Found), got %d", http.StatusFound, resp.StatusCode)
	}

	// Verify Location header points to redirect target
	location := resp.Header.Get("Location")
	if location != redirectTarget.URL {
		t.Errorf("Location header = %s, want %s", location, redirectTarget.URL)
	}
}

func TestRedirectBlockingChain(t *testing.T) {
	// Test multiple redirect chain - should stop at first redirect
	hop3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("final destination"))
	}))
	defer hop3.Close()

	hop2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, hop3.URL, http.StatusFound)
	}))
	defer hop2.Close()

	hop1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, hop2.URL, http.StatusFound)
	}))
	defer hop1.Close()

	config := DefaultConfig()
	client := NewSafeClient(config)

	resp, err := client.Get(hop1.URL)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	// Should stop at first redirect, not follow chain
	if resp.StatusCode != http.StatusFound {
		t.Errorf("Expected status 302, got %d", resp.StatusCode)
	}

	// Location should point to hop2, not hop3
	location := resp.Header.Get("Location")
	if location != hop2.URL {
		t.Errorf("Location header = %s, want %s", location, hop2.URL)
	}
}

func TestControlHookBlocking(t *testing.T) {
	// This test verifies that the Control hook blocks connections to IPs not in allowlist
	// httptest.Server listens on 127.0.0.1, so we configure an allowlist that excludes it

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("should not reach here"))
	}))
	defer server.Close()

	// Configure allowlist that DOES NOT include 127.0.0.1 (localhost)
	// Using 10.0.0.0/8 which excludes 127.x.x.x
	cidrs, err := ParseCIDRList([]string{"10.0.0.0/8"})
	if err != nil {
		t.Fatalf("Failed to parse CIDRs: %v", err)
	}

	config := &SecurityConfig{
		AllowedCIDRs:     cidrs,
		AllowlistEnabled: true,
		RequestTimeout:   5 * time.Second,
	}

	client := NewSafeClient(config)

	// Attempt connection - should fail because 127.0.0.1 not in allowlist
	_, err = client.Get(server.URL)
	if err == nil {
		t.Fatal("Expected connection to be blocked, but succeeded")
	}

	// Verify error message indicates blocking
	errStr := err.Error()
	if !strings.Contains(errStr, "blocked") && !strings.Contains(errStr, "not in allowed") {
		t.Errorf("Expected error to indicate blocked connection, got: %v", err)
	}
}

func TestControlHookAllows(t *testing.T) {
	// Verify that connections ARE allowed when IP is in allowlist

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("success"))
	}))
	defer server.Close()

	// Configure allowlist that includes 127.0.0.1 (localhost)
	cidrs, err := ParseCIDRList([]string{"127.0.0.0/8"})
	if err != nil {
		t.Fatalf("Failed to parse CIDRs: %v", err)
	}

	config := &SecurityConfig{
		AllowedCIDRs:     cidrs,
		AllowlistEnabled: true,
		RequestTimeout:   5 * time.Second,
	}

	client := NewSafeClient(config)

	// Connection should succeed
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Expected connection to succeed, got error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestNoAllowlistAllowsAll(t *testing.T) {
	// When no allowlist is configured (AllowlistEnabled=false), all connections allowed

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("success"))
	}))
	defer server.Close()

	// Default config has no allowlist
	config := DefaultConfig()
	if config.AllowlistEnabled {
		t.Fatal("DefaultConfig should have AllowlistEnabled=false")
	}

	client := NewSafeClient(config)

	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Expected connection to succeed with no allowlist, got error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestTimeoutConfiguration(t *testing.T) {
	// Create a server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.Write([]byte("delayed"))
	}))
	defer server.Close()

	// Configure very short timeout
	config := DefaultConfig()
	config.RequestTimeout = 50 * time.Millisecond

	client := NewSafeClient(config)

	_, err := client.Get(server.URL)

	// Should timeout
	if err == nil {
		t.Fatal("Expected timeout error, but request succeeded")
	}

	// Verify it's a timeout-related error
	errStr := err.Error()
	if !strings.Contains(errStr, "timeout") && !strings.Contains(errStr, "deadline exceeded") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

func TestDefaultConfigValues(t *testing.T) {
	config := DefaultConfig()

	if config.AllowlistEnabled {
		t.Error("DefaultConfig should have AllowlistEnabled=false")
	}

	if len(config.AllowedCIDRs) != 0 {
		t.Errorf("DefaultConfig should have empty AllowedCIDRs, got %d", len(config.AllowedCIDRs))
	}

	if config.RequestTimeout != DefaultRequestTimeout {
		t.Errorf("DefaultConfig timeout = %v, want %v", config.RequestTimeout, DefaultRequestTimeout)
	}
}
