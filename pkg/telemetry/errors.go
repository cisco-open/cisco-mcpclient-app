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

// Package telemetry provides types and utilities for MCP telemetry collection.
package telemetry

import "strings"

// Error type constants for classifying MCP errors.
// These are used as metric labels and in logs for error categorization.
const (
	// ErrTypeValidationError indicates input validation failed (HTTP 400).
	ErrTypeValidationError = "validation_error"

	// ErrTypeServerNotFound indicates the requested MCP server is not configured (HTTP 404).
	ErrTypeServerNotFound = "server_not_found"

	// ErrTypeSSRFBlocked indicates the request was blocked by SSRF protection (HTTP 403).
	ErrTypeSSRFBlocked = "ssrf_blocked"

	// ErrTypeTimeout indicates the request timed out (HTTP 504).
	ErrTypeTimeout = "timeout"

	// ErrTypeAuthFailure indicates an authentication error (HTTP 401).
	ErrTypeAuthFailure = "auth_failure"

	// ErrTypeConnectionFailed indicates a connection to the MCP server failed (HTTP 503).
	ErrTypeConnectionFailed = "connection_failed"

	// ErrTypeSessionError indicates an MCP session error (HTTP 500).
	ErrTypeSessionError = "session_error"

	// ErrTypeMCPProtocol indicates an MCP protocol or JSON-RPC error (HTTP 502).
	ErrTypeMCPProtocol = "mcp_protocol_error"

	// ErrTypeToolExecution indicates a tool execution error (HTTP 500).
	ErrTypeToolExecution = "tool_execution"

	// ErrTypeParseError indicates a response parsing error (HTTP 500).
	ErrTypeParseError = "parse_error"

	// ErrTypeUnknown indicates an unclassified error (HTTP 500).
	ErrTypeUnknown = "unknown"
)

// ErrorCode maps error types to HTTP-like numeric codes.
// These codes are used in logs and API responses for programmatic handling.
var ErrorCode = map[string]int{
	ErrTypeValidationError:  400,
	ErrTypeServerNotFound:   404,
	ErrTypeSSRFBlocked:      403,
	ErrTypeTimeout:          504,
	ErrTypeAuthFailure:      401,
	ErrTypeConnectionFailed: 503,
	ErrTypeSessionError:     500,
	ErrTypeMCPProtocol:      502,
	ErrTypeToolExecution:    500,
	ErrTypeParseError:       500,
	ErrTypeUnknown:          500,
}

// ValidErrorTypes contains all valid error types for validation.
var ValidErrorTypes = []string{
	ErrTypeValidationError,
	ErrTypeServerNotFound,
	ErrTypeSSRFBlocked,
	ErrTypeTimeout,
	ErrTypeAuthFailure,
	ErrTypeConnectionFailed,
	ErrTypeSessionError,
	ErrTypeMCPProtocol,
	ErrTypeToolExecution,
	ErrTypeParseError,
	ErrTypeUnknown,
}

// validErrorTypeMap is an internal map for O(1) validation lookups.
var validErrorTypeMap = func() map[string]bool {
	m := make(map[string]bool, len(ValidErrorTypes))
	for _, t := range ValidErrorTypes {
		m[t] = true
	}
	return m
}()

// IsValidErrorType returns true if the given error type is valid.
func IsValidErrorType(errType string) bool {
	return validErrorTypeMap[errType]
}

// ClassifyError categorizes an error based on its message content.
// Returns the appropriate error type constant, or empty string for nil errors.
// Classification is done by keyword matching in priority order.
func ClassifyError(err error) string {
	if err == nil {
		return ""
	}

	msg := err.Error()

	// Priority 1: SSRF protection errors
	if strings.Contains(msg, "blocked") ||
		strings.Contains(msg, "not allowed") ||
		strings.Contains(msg, "not in allowed ranges") {
		return ErrTypeSSRFBlocked
	}

	// Priority 2: Validation errors
	if strings.Contains(msg, "validation failed") {
		return ErrTypeValidationError
	}

	// Priority 3: Server not found
	if strings.Contains(msg, "not found") {
		return ErrTypeServerNotFound
	}

	// Priority 4: Timeout errors
	if strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline exceeded") {
		return ErrTypeTimeout
	}

	// Priority 5: Authentication errors
	if strings.Contains(msg, "auth") || strings.Contains(msg, "401") {
		return ErrTypeAuthFailure
	}

	// Priority 6: Session errors
	if strings.Contains(msg, "session") || strings.Contains(msg, "Session") {
		return ErrTypeSessionError
	}

	// Priority 7: Connection errors
	if strings.Contains(msg, "connection") || strings.Contains(msg, "dial") {
		return ErrTypeConnectionFailed
	}

	// Priority 8: MCP protocol errors
	if strings.Contains(msg, "MCP") || strings.Contains(msg, "JSON-RPC") {
		return ErrTypeMCPProtocol
	}

	// Priority 9: Parse errors
	if strings.Contains(msg, "unmarshal") || strings.Contains(msg, "decode") {
		return ErrTypeParseError
	}

	// Priority 10: Tool execution errors
	if strings.Contains(msg, "Tool execution error") {
		return ErrTypeToolExecution
	}

	// Default: Unknown error
	return ErrTypeUnknown
}
