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
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/mcpclient/pkg/config"
	"github.com/grafana/mcpclient/pkg/metrics"
	"github.com/grafana/mcpclient/pkg/plugin/health"
	"github.com/grafana/mcpclient/pkg/plugin/validation"
	"github.com/grafana/mcpclient/pkg/telemetry"
)

// MCPServer represents an MCP server configuration
type MCPServer struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	URL          string            `json:"url"`
	Type         string            `json:"type"` // "local" or "remote"
	Enabled      bool              `json:"enabled"`
	Description  string            `json:"description,omitempty"`
	AuthType     string            `json:"authType,omitempty"` // "none", "bearer", "basic"
	AuthToken    string            `json:"-"`                  // Bearer token (not exposed in JSON)
	AuthUser     string            `json:"-"`                  // Basic auth username (not exposed in JSON)
	AuthPass     string            `json:"-"`                  // Basic auth password (not exposed in JSON)
	Capabilities []string          `json:"capabilities,omitempty"`
	Tools        []MCPTool         `json:"tools,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Status       string            `json:"status"` // "connected", "disconnected", "error"
	LastChecked  string            `json:"lastChecked,omitempty"`
}

// MCPTool represents an available MCP tool
type MCPTool struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Parameters  map[string]string `json:"parameters,omitempty"`
}

// ServerListResponse represents the response for listing servers
type ServerListResponse struct {
	Servers []MCPServer `json:"servers"`
	Total   int         `json:"total"`
}

// ServerStatusResponse represents the server status check response
type ServerStatusResponse struct {
	Status       string    `json:"status"`
	Message      string    `json:"message,omitempty"`
	Capabilities []string  `json:"capabilities,omitempty"`
	Tools        []MCPTool `json:"tools,omitempty"`
}

// serverWriteRequest represents incoming server create/update payload.
// It supports both `authUser`/`authPass` and legacy `username`/`password`.
type serverWriteRequest struct {
	ID          *string `json:"id,omitempty"`
	Name        *string `json:"name,omitempty"`
	URL         *string `json:"url,omitempty"`
	Type        *string `json:"type,omitempty"`
	Enabled     *bool   `json:"enabled,omitempty"`
	Description *string `json:"description,omitempty"`
	AuthType    *string `json:"authType,omitempty"`
	AuthToken   *string `json:"authToken,omitempty"`
	AuthUser    *string `json:"authUser,omitempty"`
	AuthPass    *string `json:"authPass,omitempty"`
	Username    *string `json:"username,omitempty"`
	Password    *string `json:"password,omitempty"`
}

func (r serverWriteRequest) resolvedAuthUser() (string, bool) {
	if r.AuthUser != nil {
		return *r.AuthUser, true
	}
	if r.Username != nil {
		return *r.Username, true
	}
	return "", false
}

func (r serverWriteRequest) resolvedAuthPass() (string, bool) {
	if r.AuthPass != nil {
		return *r.AuthPass, true
	}
	if r.Password != nil {
		return *r.Password, true
	}
	return "", false
}

// Global variables for configuration management
var (
	mcpServers        = []MCPServer{} // In-memory storage loaded from provisioning
	configLoader      *config.ProvisioningConfigLoader
	configInitialized = false
	clientCache       = make(map[string]*MCPClient)     // Cache of connected MCP clients by server URL + auth
	schemaValidator   = validation.NewSchemaValidator() // Validator for tool arguments against inputSchema
	stateMu           sync.RWMutex
)

func getServersSnapshot() []MCPServer {
	stateMu.RLock()
	defer stateMu.RUnlock()

	servers := make([]MCPServer, len(mcpServers))
	copy(servers, mcpServers)
	return servers
}

func getServerByID(serverID string) (MCPServer, bool) {
	stateMu.RLock()
	defer stateMu.RUnlock()

	for _, server := range mcpServers {
		if server.ID == serverID {
			return server, true
		}
	}
	return MCPServer{}, false
}

func upsertServer(updated MCPServer) bool {
	stateMu.Lock()
	defer stateMu.Unlock()

	for i := range mcpServers {
		if mcpServers[i].ID == updated.ID {
			mcpServers[i] = updated
			return true
		}
	}
	return false
}

func deleteServerByID(serverID string) (MCPServer, bool) {
	stateMu.Lock()
	defer stateMu.Unlock()

	for i, server := range mcpServers {
		if server.ID == serverID {
			mcpServers = append(mcpServers[:i], mcpServers[i+1:]...)
			return server, true
		}
	}
	return MCPServer{}, false
}

func generateServerIDLocked() string {
	next := len(mcpServers) + 1
	for {
		id := fmt.Sprintf("server-%d", next)
		exists := false
		for _, s := range mcpServers {
			if s.ID == id {
				exists = true
				break
			}
		}
		if !exists {
			return id
		}
		next++
	}
}

func normalizeAuthType(authType string) string {
	switch strings.ToLower(strings.TrimSpace(authType)) {
	case "", "none":
		return "none"
	case "bearer":
		return "bearer"
	case "basic":
		return "basic"
	default:
		return "none"
	}
}

func applyServerWriteRequest(base MCPServer, req serverWriteRequest, preserveSecrets bool) MCPServer {
	updated := base

	if req.ID != nil {
		updated.ID = strings.TrimSpace(*req.ID)
	}
	if req.Name != nil {
		updated.Name = strings.TrimSpace(*req.Name)
	}
	if req.URL != nil {
		updated.URL = strings.TrimSpace(*req.URL)
	}
	if req.Type != nil {
		updated.Type = strings.TrimSpace(*req.Type)
	}
	if req.Enabled != nil {
		updated.Enabled = *req.Enabled
	}
	if req.Description != nil {
		updated.Description = strings.TrimSpace(*req.Description)
	}
	if req.AuthType != nil {
		updated.AuthType = normalizeAuthType(*req.AuthType)
	} else {
		updated.AuthType = normalizeAuthType(updated.AuthType)
	}

	authUser, hasAuthUser := req.resolvedAuthUser()
	authPass, hasAuthPass := req.resolvedAuthPass()

	switch updated.AuthType {
	case "bearer":
		if req.AuthToken != nil {
			updated.AuthToken = *req.AuthToken
		} else if !preserveSecrets {
			updated.AuthToken = ""
		}
		updated.AuthUser = ""
		updated.AuthPass = ""
	case "basic":
		if hasAuthUser {
			updated.AuthUser = authUser
		} else if !preserveSecrets {
			updated.AuthUser = ""
		}
		if hasAuthPass {
			updated.AuthPass = authPass
		} else if !preserveSecrets {
			updated.AuthPass = ""
		}
		updated.AuthToken = ""
	default:
		updated.AuthType = "none"
		updated.AuthToken = ""
		updated.AuthUser = ""
		updated.AuthPass = ""
	}

	if updated.Type == "" {
		updated.Type = "remote"
	}
	if updated.Status == "" {
		updated.Status = "disconnected"
	}
	if updated.Metadata == nil {
		updated.Metadata = make(map[string]string)
	}

	return updated
}

func validateServerConfig(server MCPServer) error {
	if strings.TrimSpace(server.Name) == "" {
		return fmt.Errorf("server name is required")
	}
	if strings.TrimSpace(server.URL) == "" {
		return fmt.Errorf("server URL is required")
	}
	return nil
}

func serverConnectionConfigChanged(before, after MCPServer) bool {
	return before.URL != after.URL ||
		before.AuthType != after.AuthType ||
		before.AuthToken != after.AuthToken ||
		before.AuthUser != after.AuthUser ||
		before.AuthPass != after.AuthPass
}

func buildClientCacheKey(serverURL, authType, authToken, authUser, authPass string) string {
	secretMaterial := authType + "\x00" + authToken + "\x00" + authUser + "\x00" + authPass
	mac := hmac.New(sha256.New, []byte(serverURL))
	mac.Write([]byte(secretMaterial))
	return serverURL + "|" + hex.EncodeToString(mac.Sum(nil))
}

// initializeConfiguration loads MCP server configurations from provisioning
func (a *App) initializeConfiguration() error {
	stateMu.RLock()
	if configInitialized {
		stateMu.RUnlock()
		return nil
	}
	stateMu.RUnlock()

	// Ensure we have app settings
	if a.appSettings == nil {
		return fmt.Errorf("app settings not available - MCP server configuration requires Grafana provisioning")
	}

	provisioningLoader := config.NewProvisioningConfigLoader(a.appSettings)

	if err := a.loadServersFromProvisioning(provisioningLoader); err != nil {
		log.DefaultLogger.Error("Failed to load servers from provisioning", "error", err)
		return err
	}

	stateMu.Lock()
	configLoader = provisioningLoader
	configInitialized = true
	serversLoaded := len(mcpServers)
	stateMu.Unlock()

	log.DefaultLogger.Info("Configuration initialized from provisioning", "servers_loaded", serversLoaded)
	return nil
}

// loadServersFromProvisioning loads servers from Grafana provisioning configuration
func (a *App) loadServersFromProvisioning(loader *config.ProvisioningConfigLoader) error {
	log.DefaultLogger.Info("Starting to load servers from provisioning")

	servers, err := loader.LoadServers()
	if err != nil {
		log.DefaultLogger.Error("Error loading servers from provisioning", "error", err)
		return fmt.Errorf("failed to load servers from provisioning: %w", err)
	}

	log.DefaultLogger.Info("Raw servers loaded from provisioning", "count", len(servers))

	// Convert config.MCPServerConfig to plugin.MCPServer
	loadedServers := make([]MCPServer, 0, len(servers))
	for _, server := range servers {
		log.DefaultLogger.Info("Converting server from provisioning", "id", server.ID, "name", server.Name, "url", server.URL, "enabled", server.Enabled)

		mcpServer := MCPServer{
			ID:          server.ID,
			Name:        server.Name,
			URL:         server.URL,
			Type:        server.Type,
			Enabled:     server.Enabled,
			Description: server.Description,
			AuthType:    server.AuthType,
			AuthToken:   server.AuthToken,
			AuthUser:    server.AuthUser,
			AuthPass:    server.AuthPass,
			Status:      "unknown", // Will be updated by status checks
			Metadata:    make(map[string]string),
		}
		loadedServers = append(loadedServers, mcpServer)
	}

	stateMu.Lock()
	mcpServers = loadedServers
	stateMu.Unlock()

	log.DefaultLogger.Info("Servers loaded from provisioning", "final_count", len(loadedServers))
	return nil
}

// getStatusMessage returns a human-readable message for the given status
func getStatusMessage(status string) string {
	switch status {
	case "connected":
		return "Connection test successful - MCP server is responding"
	case "disconnected":
		return "Connection failed - MCP server is not reachable"
	case "error":
		return "Connection error - Unable to establish MCP protocol communication"
	case "unknown":
		return "Server status has not been checked yet"
	default:
		return "Unknown server status"
	}
}

// getConfigPath returns configuration path
func getConfigPath() string {
	stateMu.RLock()
	defer stateMu.RUnlock()

	if configLoader != nil {
		return configLoader.GetConfigPath()
	}
	return "Grafana App Provisioning"
}

// getOrCreateClient retrieves an existing connected client from cache or creates a new one
func getOrCreateClient(ctx context.Context, serverURL string, authType string, authToken string, authUser string, authPass string) (*MCPClient, error) {
	cacheKey := buildClientCacheKey(serverURL, authType, authToken, authUser, authPass)

	// Check if we already have a connected client for this server
	stateMu.RLock()
	client, exists := clientCache[cacheKey]
	stateMu.RUnlock()
	if exists {
		log.DefaultLogger.Debug("Reusing cached MCP client", "url", serverURL, "sessionID", client.sessionID)
		return client, nil
	}

	// Create a new client and establish connection
	log.DefaultLogger.Debug("Creating new MCP client and establishing session", "url", serverURL, "authType", authType, "hasToken", authToken != "", "hasBasicAuth", authUser != "")
	client = NewMCPClient(serverURL, authType, authToken, authUser, authPass)

	// Establish connection and session (TestConnection creates session via initialize)
	if err := client.TestConnection(ctx); err != nil {
		return nil, fmt.Errorf("failed to establish connection: %w", err)
	}

	// Call Connect to complete the initialization (it will skip re-initialization if session exists)
	if err := client.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	// Cache the connected client for reuse unless another goroutine won the race.
	stateMu.Lock()
	if existing, alreadyCached := clientCache[cacheKey]; alreadyCached {
		stateMu.Unlock()
		client.Close()
		log.DefaultLogger.Debug("Reusing concurrently cached MCP client", "url", serverURL, "sessionID", existing.sessionID)
		return existing, nil
	}
	clientCache[cacheKey] = client
	stateMu.Unlock()
	log.DefaultLogger.Debug("MCP client connected and cached", "url", serverURL, "sessionID", client.sessionID)

	return client, nil
}

// invalidateClientForConfig removes a specific client from cache and clears schema cache.
func invalidateClientForConfig(serverURL string, authType string, authToken string, authUser string, authPass string) {
	cacheKey := buildClientCacheKey(serverURL, authType, authToken, authUser, authPass)

	stateMu.Lock()
	client, exists := clientCache[cacheKey]
	if exists {
		delete(clientCache, cacheKey)
	}
	stateMu.Unlock()

	if exists {
		client.Close()
		// Clear schema cache when client is invalidated (tools may have changed)
		// This handles: MCP server restarts, tools/list_changed notifications, connection errors
		schemaValidator.ClearCache()
		log.DefaultLogger.Debug("Invalidated MCP client and schema cache", "url", serverURL, "cacheKey", cacheKey)
	}
}

// invalidateClientsByURL removes all cached clients for a URL.
func invalidateClientsByURL(serverURL string) {
	prefix := serverURL + "|"

	stateMu.Lock()
	removed := make([]*MCPClient, 0)
	for cacheKey, client := range clientCache {
		if strings.HasPrefix(cacheKey, prefix) {
			removed = append(removed, client)
			delete(clientCache, cacheKey)
		}
	}
	stateMu.Unlock()

	if len(removed) > 0 {
		for _, client := range removed {
			client.Close()
		}
		schemaValidator.ClearCache()
		log.DefaultLogger.Debug("Invalidated MCP clients and schema cache by URL", "url", serverURL, "removed", len(removed))
	}
}

// invalidateClient removes all cached clients for a URL.
// Kept for compatibility with existing tests and call sites.
func invalidateClient(serverURL string) {
	invalidateClientsByURL(serverURL)
}

// refreshServerStatus updates server status by attempting real MCP connection
func refreshServerStatus(server *MCPServer) {
	log.DefaultLogger.Debug("Refreshing server status", "serverId", server.ID, "url", server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Try to get or create a connected client (this establishes and caches the session)
	client, err := getOrCreateClient(ctx, server.URL, server.AuthType, server.AuthToken, server.AuthUser, server.AuthPass)
	if err != nil {
		log.DefaultLogger.Error("MCP server connection failed", "serverId", server.ID, "error", err)
		server.Status = "error"
		server.Tools = []MCPTool{}
		server.Capabilities = []string{}
		server.LastChecked = time.Now().Format(time.RFC3339)
		invalidateClientForConfig(server.URL, server.AuthType, server.AuthToken, server.AuthUser, server.AuthPass)
		return
	}

	// Get tools from real server using the connected client
	tools, err := client.ListTools(ctx)
	if err != nil {
		log.DefaultLogger.Error("Failed to list MCP tools", "serverId", server.ID, "error", err)
		server.Status = "connected"
		server.Tools = []MCPTool{}
		server.Capabilities = []string{"tools"} // We know it has tools capability since we connected
		server.LastChecked = time.Now().Format(time.RFC3339)
		return
	}

	// Convert MCP tools to our format
	mcpTools := make([]MCPTool, len(tools))
	for i, tool := range tools {
		// Convert input schema to parameters map for backward compatibility
		parameters := make(map[string]string)
		if tool.InputSchema != nil {
			if props, ok := tool.InputSchema["properties"].(map[string]interface{}); ok {
				for key, prop := range props {
					if propMap, ok := prop.(map[string]interface{}); ok {
						if desc, ok := propMap["description"].(string); ok {
							parameters[key] = desc
						} else if propType, ok := propMap["type"].(string); ok {
							parameters[key] = fmt.Sprintf("Type: %s", propType)
						} else {
							parameters[key] = "Parameter"
						}
					}
				}
			}
		}

		mcpTools[i] = MCPTool{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  parameters,
		}
	}

	// Update server with real data
	server.Status = "connected"
	server.Tools = mcpTools
	server.Capabilities = []string{"tools", "resources"}
	server.LastChecked = time.Now().Format(time.RFC3339)

	log.DefaultLogger.Debug("Successfully updated server status", "serverId", server.ID, "toolCount", len(mcpTools))
}

// refreshAllServers updates status for all enabled servers
func refreshAllServers() {
	servers := getServersSnapshot()
	for i := range servers {
		if servers[i].Enabled {
			refreshServerStatus(&servers[i])
			upsertServer(servers[i])
		}
	}
}

// handleServers handles GET /servers - list all MCP servers
func (a *App) handleServers(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch req.Method {
	case http.MethodGet:
		// Refresh server status with real connections before returning
		refreshAllServers()

		servers := getServersSnapshot()
		response := ServerListResponse{
			Servers: servers,
			Total:   len(servers),
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	case http.MethodPost:
		var createReq serverWriteRequest
		if err := json.NewDecoder(req.Body).Decode(&createReq); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		server := applyServerWriteRequest(MCPServer{}, createReq, false)
		if err := validateServerConfig(server); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Test real connection to new server
		if server.Enabled {
			refreshServerStatus(&server)
		}

		stateMu.Lock()
		if strings.TrimSpace(server.ID) == "" {
			server.ID = generateServerIDLocked()
		}
		for _, existing := range mcpServers {
			if existing.ID == server.ID {
				stateMu.Unlock()
				http.Error(w, "Server with this ID already exists", http.StatusConflict)
				return
			}
		}
		mcpServers = append(mcpServers, server)
		stateMu.Unlock()

		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(server); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleServerByID handles operations on specific servers GET/PUT/DELETE /servers/{id}
func (a *App) handleServerByID(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extract server ID from path
	path := strings.TrimPrefix(req.URL.Path, "/servers/")
	serverID := strings.Split(path, "/")[0]

	if serverID == "" {
		http.Error(w, "Server ID required", http.StatusBadRequest)
		return
	}

	server, found := getServerByID(serverID)

	switch req.Method {
	case http.MethodGet:
		if !found {
			http.Error(w, "Server not found", http.StatusNotFound)
			return
		}
		if err := json.NewEncoder(w).Encode(server); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	case http.MethodPut:
		if !found {
			http.Error(w, "Server not found", http.StatusNotFound)
			return
		}

		var updateReq serverWriteRequest
		if err := json.NewDecoder(req.Body).Decode(&updateReq); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		updatedServer := applyServerWriteRequest(server, updateReq, true)
		updatedServer.ID = serverID
		if err := validateServerConfig(updatedServer); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		_ = upsertServer(updatedServer)
		if serverConnectionConfigChanged(server, updatedServer) {
			invalidateClientForConfig(server.URL, server.AuthType, server.AuthToken, server.AuthUser, server.AuthPass)
		}

		if err := json.NewEncoder(w).Encode(updatedServer); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	case http.MethodDelete:
		if !found {
			http.Error(w, "Server not found", http.StatusNotFound)
			return
		}

		removedServer, removed := deleteServerByID(serverID)
		if !removed {
			http.Error(w, "Server not found", http.StatusNotFound)
			return
		}
		invalidateClientForConfig(
			removedServer.URL,
			removedServer.AuthType,
			removedServer.AuthToken,
			removedServer.AuthUser,
			removedServer.AuthPass,
		)

		// Return success response with JSON
		w.Header().Set("Content-Type", "application/json")
		response := map[string]interface{}{
			"success":  true,
			"message":  "Server deleted successfully",
			"serverId": serverID,
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleServerStatus handles GET /servers/{id}/status - check server status
func (a *App) handleServerStatus(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Extract server ID from path
	path := strings.TrimPrefix(req.URL.Path, "/servers/")
	serverID := strings.Split(path, "/")[0]

	server, found := getServerByID(serverID)
	if !found {
		http.Error(w, "Server not found", http.StatusNotFound)
		return
	}

	// Simulate status check (in real implementation, would ping the MCP server)
	response := ServerStatusResponse{
		Status:       server.Status,
		Message:      "Status check successful",
		Capabilities: server.Capabilities,
		Tools:        server.Tools,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleServerTest handles POST /servers/{id}/test - test server connection
func (a *App) handleServerTest(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Extract server ID from path
	path := strings.TrimPrefix(req.URL.Path, "/servers/")
	serverID := strings.Split(path, "/")[0]

	server, found := getServerByID(serverID)
	if !found {
		http.Error(w, "Server not found", http.StatusNotFound)
		return
	}

	// Perform real MCP server connection test
	refreshServerStatus(&server)
	_ = upsertServer(server)

	response := ServerStatusResponse{
		Status:       server.Status,
		Message:      getStatusMessage(server.Status),
		Capabilities: server.Capabilities,
		Tools:        server.Tools,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleTools handles GET /tools - list all available tools from all servers
func (a *App) handleTools(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Refresh server status to get latest tools from real MCP servers
	refreshAllServers()

	// Collect tools from all connected servers (now using real MCP data)
	var allTools []MCPTool
	for _, server := range getServersSnapshot() {
		if server.Enabled && server.Status == "connected" {
			// Add server ID to each tool for identification
			for _, tool := range server.Tools {
				toolWithServer := tool
				toolWithServer.Parameters = make(map[string]string)
				for k, v := range tool.Parameters {
					toolWithServer.Parameters[k] = v
				}
				toolWithServer.Parameters["serverId"] = server.ID
				toolWithServer.Parameters["serverName"] = server.Name
				allTools = append(allTools, toolWithServer)
			}
		}
	}

	response := map[string]interface{}{
		"tools": allTools,
		"total": len(allTools),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleConfig handles GET/POST /config - get/set global MCP client configuration
func (a *App) handleConfig(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch req.Method {
	case http.MethodGet:
		config := map[string]interface{}{
			"autoDiscovery":     true,
			"connectionTimeout": 30,
			"retryAttempts":     3,
			"enableLogging":     true,
		}
		if err := json.NewEncoder(w).Encode(config); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	case http.MethodPost:
		var config map[string]interface{}
		if err := json.NewDecoder(req.Body).Decode(&config); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// In production, would save to database
		if err := json.NewEncoder(w).Encode(config); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// ToolCallRequest represents the request for calling a tool
type ToolCallRequest struct {
	ToolName  string                 `json:"tool_name"`
	Arguments map[string]interface{} `json:"arguments"`
	ServerID  string                 `json:"server_id,omitempty"`
}

// ToolCallResponse represents the response from calling a tool
type ToolCallResponse struct {
	Success bool   `json:"success"`
	Content string `json:"content"`
	Error   string `json:"error,omitempty"`
}

// handleToolCall handles POST /tools/call - execute a tool on an MCP server
func (a *App) handleToolCall(w http.ResponseWriter, req *http.Request) {
	start := time.Now()
	ctx := req.Context()
	logger := log.DefaultLogger.FromContext(ctx)

	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var toolCallReq ToolCallRequest
	if err := json.NewDecoder(req.Body).Decode(&toolCallReq); err != nil {
		metrics.ErrorsTotal.WithLabelValues(telemetry.ErrTypeValidationError).Inc()
		logger.Error("Invalid tool call request", "error", "invalid JSON")
		http.Error(w, "Invalid JSON request", http.StatusBadRequest)
		return
	}

	logger.Debug("Tool call request received", "tool", toolCallReq.ToolName, "server_id", toolCallReq.ServerID)

	// Find the server that has this tool
	var targetServer MCPServer
	targetServerFound := false
	for _, server := range getServersSnapshot() {
		if !server.Enabled || server.Status != "connected" {
			continue
		}

		// If serverID is specified, use that server
		if toolCallReq.ServerID != "" && server.ID == toolCallReq.ServerID {
			targetServer = server
			targetServerFound = true
			break
		}

		// Otherwise, find the first server that has this tool
		for _, tool := range server.Tools {
			if tool.Name == toolCallReq.ToolName {
				targetServer = server
				targetServerFound = true
				break
			}
		}
		if targetServerFound {
			break
		}
	}

	if !targetServerFound {
		metrics.ErrorsTotal.WithLabelValues(telemetry.ErrTypeServerNotFound).Inc()
		logger.Warn("Server not found for tool call", "tool", toolCallReq.ToolName, "server_id", toolCallReq.ServerID)
		response := ToolCallResponse{
			Success: false,
			Error:   fmt.Sprintf("Tool '%s' not found on any connected server", toolCallReq.ToolName),
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Extract labels for metrics
	serverName := targetServer.Name
	toolName := toolCallReq.ToolName
	recordToolCall := func(status, errType string) {
		duration := time.Since(start).Seconds()
		metrics.ToolCallsTotal.WithLabelValues(serverName, toolName, status).Inc()
		metrics.RequestLatency.WithLabelValues(serverName, toolName).Observe(duration)
		if errType != "" {
			metrics.ErrorsTotal.WithLabelValues(errType).Inc()
		}
	}

	// Create context with timeout using the request context
	callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Get or create a connected MCP client (reuses existing session if available)
	client, err := getOrCreateClient(callCtx, targetServer.URL, targetServer.AuthType, targetServer.AuthToken, targetServer.AuthUser, targetServer.AuthPass)
	if err != nil {
		errType := telemetry.ClassifyError(err)
		recordToolCall("error", errType)
		logger.Error("Failed to get MCP client", "server", serverName, "tool", toolName, "duration_ms", time.Since(start).Milliseconds(), "error_type", errType, "error_code", telemetry.ErrorCode[errType])
		response := ToolCallResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to connect to MCP server: %v", err),
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Fetch tool definitions to get inputSchema for validation
	// The cached MCPTool doesn't include the full inputSchema, so we need to fetch it
	var toolSchema map[string]interface{}
	tools, listErr := client.ListTools(callCtx)
	if listErr == nil {
		for _, tool := range tools {
			if tool.Name == toolName {
				toolSchema = tool.InputSchema
				break
			}
		}
	} else {
		logger.Warn("Failed to fetch tool list for validation, skipping validation", "error", listErr, "tool", toolName)
	}

	// Validate arguments against schema before calling tool
	if err := schemaValidator.Validate(toolName, toolSchema, toolCallReq.Arguments); err != nil {
		recordToolCall("error", telemetry.ErrTypeValidationError)
		logger.Warn("Tool argument validation failed", "tool", toolName, "error", err)
		w.WriteHeader(http.StatusBadRequest)
		response := ToolCallResponse{
			Success: false,
			Error:   fmt.Sprintf("Invalid tool arguments: %v", err),
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Call the tool using the connected client with established session
	result, err := client.CallTool(callCtx, toolName, toolCallReq.Arguments)

	// Handle context cancellation
	if callCtx.Err() == context.Canceled {
		recordToolCall("error", telemetry.ErrTypeTimeout)
		response := ToolCallResponse{Success: false, Error: "Tool call was cancelled"}
		json.NewEncoder(w).Encode(response)
		return
	}
	if callCtx.Err() == context.DeadlineExceeded {
		recordToolCall("error", telemetry.ErrTypeTimeout)
		response := ToolCallResponse{Success: false, Error: "Tool call timed out after 30 seconds"}
		json.NewEncoder(w).Encode(response)
		return
	}

	if err != nil {
		errType := telemetry.ClassifyError(err)
		recordToolCall("error", errType)
		logger.Error("Tool call failed", "server", serverName, "tool", toolName, "duration_ms", time.Since(start).Milliseconds(), "error_type", errType, "error_code", telemetry.ErrorCode[errType])

		// If the error indicates a session issue, invalidate the cached client so next call creates a new one
		errMsg := err.Error()
		if strings.Contains(errMsg, "session") || strings.Contains(errMsg, "Session") {
			logger.Warn("Session error detected, invalidating cached client", "server", targetServer.URL)
			invalidateClientForConfig(
				targetServer.URL,
				targetServer.AuthType,
				targetServer.AuthToken,
				targetServer.AuthUser,
				targetServer.AuthPass,
			)
		}

		response := ToolCallResponse{
			Success: false,
			Error:   fmt.Sprintf("Tool call failed: %v", err),
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Record metrics and log completion
	status := "success"
	errorType := ""
	if result.IsError {
		status = "error"
		errorType = telemetry.ErrTypeToolExecution
	}
	recordToolCall(status, errorType)

	// Successful calls at debug level (per CONTEXT.md), errors at error level
	if status == "success" {
		logger.Debug("Tool call completed", "server", serverName, "tool", toolName, "duration_ms", time.Since(start).Milliseconds(), "status", status)
	} else {
		logger.Error("Tool execution returned error", "server", serverName, "tool", toolName, "duration_ms", time.Since(start).Milliseconds(), "status", status)
	}

	// Extract text content from result
	var content string
	if result.IsError {
		content = "Tool execution error: "
	}

	for _, c := range result.Content {
		if c.Type == "text" {
			content += c.Text
		} else {
			content += fmt.Sprintf("[%s content]", c.Type)
		}
	}

	response := ToolCallResponse{
		Success: !result.IsError,
		Content: content,
		Error:   "",
	}

	if result.IsError {
		response.Error = content
		response.Content = ""
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handlePing is an example HTTP GET resource that returns a {"message": "ok"} JSON response.
func (a *App) handlePing(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	if _, err := w.Write([]byte(`{"message": "ok"}`)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// handleEcho is an example HTTP POST resource that accepts a JSON with a "message" key and
// returns to the client whatever it is sent.
func (a *App) handleEcho(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// TestConnectionRequest represents the request body for direct connection testing
type TestConnectionRequest struct {
	URL       string `json:"url"`
	AuthType  string `json:"authType,omitempty"`
	AuthToken string `json:"authToken,omitempty"`
	AuthUser  string `json:"authUser,omitempty"`
	AuthPass  string `json:"authPass,omitempty"`
	Username  string `json:"username,omitempty"`
	Password  string `json:"password,omitempty"`
}

// handleTestConnection handles POST /test-connection - test connection without saving server
func (a *App) handleTestConnection(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var testReq TestConnectionRequest
	if err := json.NewDecoder(req.Body).Decode(&testReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if testReq.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}
	testReq.AuthType = normalizeAuthType(testReq.AuthType)
	if testReq.AuthUser == "" {
		testReq.AuthUser = testReq.Username
	}
	if testReq.AuthPass == "" {
		testReq.AuthPass = testReq.Password
	}

	// Validate URL format
	if !strings.HasPrefix(testReq.URL, "http://") && !strings.HasPrefix(testReq.URL, "https://") {
		response := ServerStatusResponse{
			Status:  "error",
			Message: "Invalid URL format. URL must start with http:// or https://",
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Test connection directly without creating a server entry
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var status string
	var message string
	var capabilities []string
	var tools []MCPTool

	// Use getOrCreateClient to establish connection with session management
	client, err := getOrCreateClient(ctx, testReq.URL, testReq.AuthType, testReq.AuthToken, testReq.AuthUser, testReq.AuthPass)
	if err != nil {
		log.DefaultLogger.Error("Direct connection test failed", "url", testReq.URL, "error", err)
		status = "error"
		message = fmt.Sprintf("Connection failed: %v", err)
		invalidateClientForConfig(testReq.URL, testReq.AuthType, testReq.AuthToken, testReq.AuthUser, testReq.AuthPass)
	} else {
		// Get tools from server
		serverTools, err := client.ListTools(ctx)
		if err != nil {
			log.DefaultLogger.Warn("Failed to list tools during connection test", "url", testReq.URL, "error", err)
			tools = []MCPTool{}
		} else {
			// Convert ToolDefinition to MCPTool
			tools = make([]MCPTool, len(serverTools))
			for i, tool := range serverTools {
				tools[i] = MCPTool{
					Name:        tool.Name,
					Description: tool.Description,
					Parameters:  make(map[string]string), // Simplified for now
				}
			}
		}

		status = "connected"
		message = "Connection successful! MCP server is responding."
		capabilities = []string{"tools"} // Basic capability
	}

	response := ServerStatusResponse{
		Status:       status,
		Message:      message,
		Capabilities: capabilities,
		Tools:        tools,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// registerRoutes takes a *http.ServeMux and registers some HTTP handlers.
func (a *App) registerRoutes(mux *http.ServeMux) {
	// Initialize configuration from .ini file on startup
	if err := a.initializeConfiguration(); err != nil {
		log.DefaultLogger.Error("Failed to initialize configuration", "error", err)
	}

	// Original example routes
	mux.HandleFunc("/ping", a.handlePing)
	mux.HandleFunc("/echo", a.handleEcho)

	// MCP Client API routes
	mux.HandleFunc("/servers", a.handleServers)
	mux.HandleFunc("/test-connection", a.handleTestConnection)
	mux.HandleFunc("/config/reload", a.handleConfigReload)
	mux.HandleFunc("/config/status", a.handleConfigStatus)
	mux.HandleFunc("/servers/", func(w http.ResponseWriter, req *http.Request) {
		path := strings.TrimPrefix(req.URL.Path, "/servers/")
		parts := strings.Split(path, "/")

		if len(parts) >= 2 {
			switch parts[1] {
			case "status":
				a.handleServerStatus(w, req)
			case "test":
				a.handleServerTest(w, req)
			default:
				a.handleServerByID(w, req)
			}
		} else {
			a.handleServerByID(w, req)
		}
	})
	mux.HandleFunc("/tools", a.handleTools)
	mux.HandleFunc("/tools/call", a.handleToolCall)
	mux.HandleFunc("/config", a.handleConfig)
}

// handleConfigReload reloads server configurations from the .ini file
func (a *App) handleConfigReload(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.DefaultLogger.Info("Configuration reload requested")
	servers := getServersSnapshot()

	// With provisioning-based configuration, changes require Grafana restart
	// Return current configuration status instead of attempting reload
	response := map[string]interface{}{
		"status":        "configuration_managed_by_provisioning",
		"message":       "MCP server configuration is now managed through Grafana provisioning. Restart Grafana to apply changes.",
		"servers_count": len(servers),
		"config_path":   getConfigPath(),
		"timestamp":     time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleConfigStatus returns information about the current configuration
func (a *App) handleConfigStatus(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stateMu.RLock()
	initialized := configInitialized
	stateMu.RUnlock()
	servers := getServersSnapshot()

	response := map[string]interface{}{
		"initialized":   initialized,
		"config_source": "Grafana App Provisioning",
		"config_path":   getConfigPath(),
		"servers_count": len(servers),
		"servers":       getServerSummary(),
		"configuration": "Managed through Grafana provisioning (apps.yaml)",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getServerSummary returns a summary of configured servers
func getServerSummary() []map[string]interface{} {
	servers := getServersSnapshot()
	summary := make([]map[string]interface{}, 0, len(servers))
	for _, server := range servers {
		summary = append(summary, map[string]interface{}{
			"id":      server.ID,
			"name":    server.Name,
			"url":     server.URL,
			"type":    server.Type,
			"enabled": server.Enabled,
			"status":  server.Status,
		})
	}
	return summary
}

// mcpServerProvider implements health.ServerProvider for the App.
type mcpServerProvider struct {
	app *App
}

// GetEnabledServers returns all enabled MCP servers for health checking.
func (sp *mcpServerProvider) GetEnabledServers() []health.ServerInfo {
	var servers []health.ServerInfo
	for _, s := range getServersSnapshot() {
		if s.Enabled {
			servers = append(servers, health.ServerInfo{
				Name:      s.Name,
				URL:       s.URL,
				AuthType:  s.AuthType,
				AuthToken: s.AuthToken,
				AuthUser:  s.AuthUser,
				AuthPass:  s.AuthPass,
			})
		}
	}
	return servers
}

// mcpConnectionTester implements health.ConnectionTester using MCPClient.
type mcpConnectionTester struct{}

// TestConnection tests connectivity to an MCP server.
func (t *mcpConnectionTester) TestConnection(ctx context.Context, server health.ServerInfo) error {
	client := NewMCPClient(server.URL, server.AuthType, server.AuthToken, server.AuthUser, server.AuthPass)
	return client.TestConnection(ctx)
}
