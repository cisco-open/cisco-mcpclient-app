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

package config

import (
	"encoding/json"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

// MCPServerConfig represents configuration for an MCP server
type MCPServerConfig struct {
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
}

// ProvisioningMCPServer represents an MCP server from provisioning configuration
type ProvisioningMCPServer struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	Type        string `json:"type"`
	Enabled     bool   `json:"enabled"`
	Description string `json:"description,omitempty"`
	AuthType    string `json:"authType,omitempty"`
	AuthToken   string `json:"authToken,omitempty"` // Will be populated from secureJsonData
}

// ProvisioningConfigLoader handles loading MCP server configurations from Grafana provisioning
type ProvisioningConfigLoader struct {
	appSettings *backend.AppInstanceSettings
}

// NewProvisioningConfigLoader creates a new provisioning configuration loader
func NewProvisioningConfigLoader(appSettings *backend.AppInstanceSettings) *ProvisioningConfigLoader {
	return &ProvisioningConfigLoader{
		appSettings: appSettings,
	}
}

// LoadServers loads MCP server configurations from Grafana app settings
func (loader *ProvisioningConfigLoader) LoadServers() ([]MCPServerConfig, error) {
	if loader.appSettings == nil {
		return nil, fmt.Errorf("app settings not available")
	}

	// Parse jsonData to get MCP servers configuration
	var jsonData map[string]interface{}
	if err := json.Unmarshal(loader.appSettings.JSONData, &jsonData); err != nil {
		log.DefaultLogger.Error("Failed to parse app jsonData", "error", err)
		return nil, fmt.Errorf("failed to parse app configuration: %w", err)
	}

	// Extract MCP servers from jsonData
	mcpServersData, exists := jsonData["mcpServers"]
	if !exists {
		log.DefaultLogger.Info("No mcpServers configuration found in app settings")
		return []MCPServerConfig{}, nil
	}

	// Convert to slice of interfaces
	serversSlice, ok := mcpServersData.([]interface{})
	if !ok {
		return nil, fmt.Errorf("mcpServers configuration is not a valid array")
	}

	// Log what secure keys are available (without values)
	if loader.appSettings.DecryptedSecureJSONData != nil {
		keys := make([]string, 0, len(loader.appSettings.DecryptedSecureJSONData))
		for k := range loader.appSettings.DecryptedSecureJSONData {
			keys = append(keys, k)
		}
		log.DefaultLogger.Info("Available secureJsonData keys", "keys", keys)
	}

	var servers []MCPServerConfig
	for i, serverData := range serversSlice {
		serverMap, ok := serverData.(map[string]interface{})
		if !ok {
			log.DefaultLogger.Warn("Invalid server configuration at index", "index", i)
			continue
		}

		server := MCPServerConfig{
			ID:          getStringValue(serverMap, "id"),
			Name:        getStringValue(serverMap, "name"),
			URL:         getStringValue(serverMap, "url"),
			Type:        getStringValue(serverMap, "type"),
			Enabled:     getBoolValue(serverMap, "enabled"),
			Description: getStringValue(serverMap, "description"),
			AuthType:    getStringValue(serverMap, "authType"),
		}

		// Get auth credentials from app-level secureJsonData using server.id prefix
		if server.ID != "" && loader.appSettings.DecryptedSecureJSONData != nil {
			switch server.AuthType {
			case "bearer":
				tokenKey := fmt.Sprintf("%s.token", server.ID)
				token, exists := loader.appSettings.DecryptedSecureJSONData[tokenKey]
				log.DefaultLogger.Info("Token lookup", "serverID", server.ID, "tokenKey", tokenKey, "exists", exists, "isEmpty", token == "")
				if exists && token != "" {
					server.AuthToken = token
				}
			case "basic":
				usernameKey := fmt.Sprintf("%s.username", server.ID)
				passwordKey := fmt.Sprintf("%s.password", server.ID)
				if username, exists := loader.appSettings.DecryptedSecureJSONData[usernameKey]; exists && username != "" {
					server.AuthUser = username
				}
				if password, exists := loader.appSettings.DecryptedSecureJSONData[passwordKey]; exists && password != "" {
					server.AuthPass = password
				}
			}
		}

		// Skip servers with missing required fields
		if server.ID == "" || server.Name == "" || server.URL == "" {
			log.DefaultLogger.Warn("Skipping server with missing required fields",
				"id", server.ID,
				"name", server.Name,
				"url", server.URL,
			)
			continue
		}

		servers = append(servers, server)
	}

	log.DefaultLogger.Info("Loaded MCP servers from provisioning", "count", len(servers))
	return servers, nil
}

// GetConfigPath returns the source of configuration (for logging purposes)
func (loader *ProvisioningConfigLoader) GetConfigPath() string {
	return "Grafana App Provisioning (apps.yaml)"
}

// Helper functions for type conversion

func getStringValue(m map[string]interface{}, key string) string {
	if val, exists := m[key]; exists {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getBoolValue(m map[string]interface{}, key string) bool {
	if val, exists := m[key]; exists {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}