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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/mcpclient/pkg/plugin/ssrf"
)

// securityConfig holds the SSRF protection configuration loaded once at startup.
var securityConfig *ssrf.SecurityConfig

func init() {
	var err error
	securityConfig, err = ssrf.LoadSecurityConfig()
	if err != nil {
		log.DefaultLogger.Error("Failed to load security config, using defaults", "error", err)
		securityConfig = ssrf.DefaultConfig()
	}
}

// MCPClient handles real connections to MCP servers
type MCPClient struct {
	baseURL    string
	httpClient *http.Client
	serverInfo *ServerInfo
	sessionID  string // Session ID for streamable-http transport
	authType   string // Authentication type: "none", "bearer", "basic"
	authToken  string // Bearer token for authentication
	authUser   string // Username for basic authentication
	authPass   string // Password for basic authentication
}

// MCPRequest represents a JSON-RPC request to MCP server
type MCPRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	ID      int         `json:"id"`
}

// MCPResponse represents a JSON-RPC response from MCP server
type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

// MCPError represents an error from MCP server
type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// InitializeParams represents MCP initialize parameters
type InitializeParams struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ClientInfo      ClientInfo             `json:"clientInfo"`
}

// ClientInfo represents MCP client information
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// InitializeResult represents MCP initialize response
type InitializeResult struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ServerInfo      ServerInfo             `json:"serverInfo"`
}

// ServerInfo represents MCP server information
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ToolsListResult represents tools/list response
type ToolsListResult struct {
	Tools []ToolDefinition `json:"tools"`
}

// ToolDefinition represents an MCP tool definition
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// NewMCPClient creates a new MCP client for the given server URL and optional auth token.
// Uses SSRF-safe HTTP transport for all MCP server requests.
func NewMCPClient(serverURL string, authType string, authToken string, authUser string, authPass string) *MCPClient {
	return &MCPClient{
		baseURL:   serverURL,
		authType:  authType,
		authToken: authToken,
		authUser:  authUser,
		authPass:  authPass,
		// Use SSRF-safe client for all MCP server connections.
		// This applies IP validation at connection time and disables redirects.
		httpClient: ssrf.NewSafeClient(securityConfig),
	}
}

// Connect establishes connection to MCP server and initializes session
func (c *MCPClient) Connect(ctx context.Context) error {
	log.DefaultLogger.Debug("Connecting to MCP server", "url", c.baseURL, "existingSessionID", c.sessionID)

	// If we already have a session ID (from TestConnection), we're already connected
	if c.sessionID != "" {
		log.DefaultLogger.Debug("Already have session, skipping re-initialization")
		return nil
	}

	// Initialize MCP protocol (this also establishes the session)
	if err := c.initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize MCP protocol: %w", err)
	}

	log.DefaultLogger.Debug("Successfully connected to MCP server", "sessionID", c.sessionID)
	return nil
}

// GetServerInfo retrieves server information (after connection)
func (c *MCPClient) GetServerInfo(ctx context.Context) (*ServerInfo, error) {
	if c.serverInfo == nil {
		return nil, fmt.Errorf("server info not available - not connected or initialization failed")
	}
	return c.serverInfo, nil
}

// Close closes the MCP client connection
func (c *MCPClient) Close() error {
	// For Streamable HTTP mode, no persistent connection to close
	return nil
}

// initialize sends the MCP initialize request
func (c *MCPClient) initialize(ctx context.Context) error {
	params := InitializeParams{
		ProtocolVersion: "2024-11-05",
		Capabilities: map[string]interface{}{
			"roots": map[string]interface{}{
				"listChanged": true,
			},
		},
		ClientInfo: ClientInfo{
			Name:    "cisco-mcpclient-app",
			Version: "1.0.0",
		},
	}

	request := MCPRequest{
		JSONRPC: "2.0",
		Method:  "initialize",
		Params:  params,
		ID:      1,
	}

	var response MCPResponse
	if err := c.sendRequest(ctx, request, &response); err != nil {
		return err
	}

	if response.Error != nil {
		return fmt.Errorf("MCP initialize error: %s", response.Error.Message)
	}

	// Parse the initialize result to extract server info
	if response.Result != nil {
		resultBytes, err := json.Marshal(response.Result)
		if err == nil {
			var initResult InitializeResult
			if err := json.Unmarshal(resultBytes, &initResult); err == nil {
				c.serverInfo = &initResult.ServerInfo
			}
		}
	}

	log.DefaultLogger.Debug("MCP protocol initialized successfully")
	return nil
}

// ListTools retrieves available tools from the MCP server
func (c *MCPClient) ListTools(ctx context.Context) ([]ToolDefinition, error) {
	request := MCPRequest{
		JSONRPC: "2.0",
		Method:  "tools/list",
		ID:      2,
	}

	var response MCPResponse
	if err := c.sendRequest(ctx, request, &response); err != nil {
		return nil, err
	}

	if response.Error != nil {
		return nil, fmt.Errorf("MCP tools/list error: %s", response.Error.Message)
	}

	// Parse the result
	resultBytes, err := json.Marshal(response.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tools result: %w", err)
	}

	var toolsResult ToolsListResult
	if err := json.Unmarshal(resultBytes, &toolsResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tools result: %w", err)
	}

	log.DefaultLogger.Debug("Retrieved tools from MCP server", "count", len(toolsResult.Tools))
	return toolsResult.Tools, nil
}

// sendRequest sends a JSON-RPC request to the MCP server
func (c *MCPClient) sendRequest(ctx context.Context, request MCPRequest, response *MCPResponse) error {
	// Use the baseURL directly (it should already include the /mcp endpoint)
	messageURL := c.baseURL

	// Marshal request
	requestBody, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	log.DefaultLogger.Debug("Sending MCP request", "method", request.Method, "url", messageURL, "sessionID", c.sessionID)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", messageURL, bytes.NewReader(requestBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	// Required for streamable HTTP transport
	req.Header.Set("Accept", "application/json, text/event-stream")

	// Include session ID if we have one (required for streamable-http after initialize)
	if c.sessionID != "" {
		req.Header.Set("Mcp-Session-Id", c.sessionID)
	}

	// Add authentication based on auth type
	switch c.authType {
	case "bearer":
		if c.authToken != "" {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.authToken))
		}
	case "basic":
		if c.authUser != "" && c.authPass != "" {
			req.SetBasicAuth(c.authUser, c.authPass)
		}
	}

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Check if this is an SSRF block (from Dialer.Control hook)
		if strings.Contains(err.Error(), "blocked") || strings.Contains(err.Error(), "not allowed") || strings.Contains(err.Error(), "not in allowed ranges") {
			log.DefaultLogger.Warn("SSRF blocked", "url", messageURL, "error", err)
			return fmt.Errorf("connection blocked: %w", err)
		}
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Capture session ID from response header (server sends this after initialize)
	if sessionID := resp.Header.Get("Mcp-Session-Id"); sessionID != "" {
		c.sessionID = sessionID
		log.DefaultLogger.Debug("Captured MCP session ID", "sessionID", sessionID)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP status %d: %s", resp.StatusCode, string(body))
	}

	// Read response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle SSE format for streamable HTTP transport
	responseText := string(responseBody)
	if len(responseText) > 0 && responseText[0] == 'e' && bytes.Contains(responseBody, []byte("event: message")) {
		// Parse SSE format
		lines := bytes.Split(responseBody, []byte("\n"))
		var jsonData []byte
		for _, line := range lines {
			if bytes.HasPrefix(line, []byte("data: ")) {
				jsonData = line[6:] // Remove "data: " prefix
				break
			}
		}
		if len(jsonData) == 0 {
			return fmt.Errorf("no data found in SSE response")
		}

		// Parse JSON from SSE data
		if err := json.Unmarshal(jsonData, response); err != nil {
			return fmt.Errorf("failed to decode SSE JSON response: %w", err)
		}
	} else {
		// Handle regular JSON response
		if err := json.Unmarshal(responseBody, response); err != nil {
			return fmt.Errorf("failed to decode JSON response: %w", err)
		}
	}

	return nil
}

// TestConnection tests if the MCP server is reachable and responsive
func (c *MCPClient) TestConnection(ctx context.Context) error {
	// Use initialize method to test connection - this is the standard MCP handshake
	// that all MCP servers must support (streamable-http servers require this first)
	// Note: Each call creates a new session. Session caching would require refactoring
	// to persist MCPClient instances across requests.
	params := InitializeParams{
		ProtocolVersion: "2024-11-05",
		Capabilities:    map[string]interface{}{},
		ClientInfo: ClientInfo{
			Name:    "cisco-mcpclient-app",
			Version: "1.0.0",
		},
	}

	request := MCPRequest{
		JSONRPC: "2.0",
		Method:  "initialize",
		Params:  params,
		ID:      999,
	}

	var response MCPResponse
	if err := c.sendRequest(ctx, request, &response); err != nil {
		return fmt.Errorf("failed to connect to MCP server: %w", err)
	}

	// Check if it's a valid JSON-RPC response
	if response.JSONRPC != "2.0" {
		return fmt.Errorf("server does not support JSON-RPC 2.0 protocol")
	}

	// Check for errors
	if response.Error != nil {
		return fmt.Errorf("MCP server error: %s", response.Error.Message)
	}

	return nil
}

// CallToolParams represents parameters for calling a tool
type CallToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// ToolCallResult represents the result of calling a tool
type ToolCallResult struct {
	Content []ToolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// ToolContent represents content from a tool call
type ToolContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// CallTool calls a tool on the MCP server
func (c *MCPClient) CallTool(ctx context.Context, toolName string, arguments map[string]interface{}) (*ToolCallResult, error) {
	log.DefaultLogger.Debug("MCPClient.CallTool called", "tool", toolName, "argumentCount", len(arguments))

	params := CallToolParams{
		Name:      toolName,
		Arguments: arguments,
	}

	request := MCPRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  params,
		ID:      200, // Use a distinct ID for tool calls
	}

	var response MCPResponse
	if err := c.sendRequest(ctx, request, &response); err != nil {
		return nil, fmt.Errorf("failed to call tool %s: %w", toolName, err)
	}

	if response.Error != nil {
		return &ToolCallResult{
			Content: []ToolContent{{
				Type: "text",
				Text: fmt.Sprintf("Tool call error: %s", response.Error.Message),
			}},
			IsError: true,
		}, nil
	}

	// Parse the result
	resultBytes, err := json.Marshal(response.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tool call result: %w", err)
	}

	var result ToolCallResult
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		// If unmarshaling fails, try to extract text content directly
		log.DefaultLogger.Warn("Failed to unmarshal tool result, trying direct extraction", "error", err)

		// Try to extract text from various possible response formats
		if resultMap, ok := response.Result.(map[string]interface{}); ok {
			if content, exists := resultMap["content"]; exists {
				result = ToolCallResult{
					Content: []ToolContent{{
						Type: "text",
						Text: fmt.Sprintf("%v", content),
					}},
					IsError: false,
				}
			} else {
				// Fallback: use the entire result as text
				result = ToolCallResult{
					Content: []ToolContent{{
						Type: "text",
						Text: fmt.Sprintf("%v", response.Result),
					}},
					IsError: false,
				}
			}
		} else {
			return nil, fmt.Errorf("failed to unmarshal tool call result: %w", err)
		}
	}

	log.DefaultLogger.Debug("Tool call completed successfully", "tool", toolName, "contentCount", len(result.Content), "isError", result.IsError)
	return &result, nil
}
