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

// Package testutil provides test fixtures and helpers for mcpclient tests.
package testutil

// TestServerConfig represents test server configuration data.
// This is used to create test fixtures for handler tests.
type TestServerConfig struct {
	ID          string
	Name        string
	URL         string
	Type        string
	Enabled     bool
	Description string
	AuthType    string
	AuthToken   string
	AuthUser    string
	AuthPass    string
	Status      string
}

// TestServer creates a TestServerConfig with sensible defaults.
// Parameters name and url are required; all other fields use defaults.
func TestServer(name, url string) TestServerConfig {
	return TestServerConfig{
		ID:          "test-" + name,
		Name:        name,
		URL:         url,
		Type:        "remote",
		Enabled:     true,
		Description: "Test server: " + name,
		AuthType:    "none",
		Status:      "unknown",
	}
}

// TestServerWithAuth creates a TestServerConfig with authentication.
// Supports bearer token authentication.
func TestServerWithAuth(name, url, authType, token string) TestServerConfig {
	cfg := TestServer(name, url)
	cfg.AuthType = authType
	cfg.AuthToken = token
	return cfg
}

// TestServerWithBasicAuth creates a TestServerConfig with basic auth.
func TestServerWithBasicAuth(name, url, username, password string) TestServerConfig {
	cfg := TestServer(name, url)
	cfg.AuthType = "basic"
	cfg.AuthUser = username
	cfg.AuthPass = password
	return cfg
}

// TestServerDisabled creates a disabled TestServerConfig.
func TestServerDisabled(name, url string) TestServerConfig {
	cfg := TestServer(name, url)
	cfg.Enabled = false
	return cfg
}

// ToJSON returns a JSON string representation of the server config.
// Useful for constructing POST/PUT request bodies in tests.
func (c TestServerConfig) ToJSON() string {
	// Note: Simple JSON construction to avoid import cycles with encoding/json
	return `{"id":"` + c.ID + `","name":"` + c.Name + `","url":"` + c.URL + `","type":"` + c.Type + `","enabled":` + boolToString(c.Enabled) + `,"description":"` + c.Description + `","authType":"` + c.AuthType + `","status":"` + c.Status + `"}`
}

// ToJSONMinimal returns minimal JSON with just name and url.
// Useful for testing POST requests that auto-assign IDs.
func (c TestServerConfig) ToJSONMinimal() string {
	return `{"name":"` + c.Name + `","url":"` + c.URL + `"}`
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
