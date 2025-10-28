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

package telemetry

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestClassifyError uses table-driven tests to verify error classification.
func TestClassifyError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		// Nil error
		{"nil error", nil, ""},

		// SSRF blocked errors (Priority 1)
		{"ssrf blocked - IP blocked", errors.New("IP blocked by SSRF protection"), ErrTypeSSRFBlocked},
		{"ssrf blocked - not allowed", errors.New("host not allowed"), ErrTypeSSRFBlocked},
		{"ssrf blocked - not in allowed ranges", errors.New("IP not in allowed ranges"), ErrTypeSSRFBlocked},

		// Validation errors (Priority 2)
		{"validation failed", errors.New("validation failed: missing required field"), ErrTypeValidationError},
		// Note: ClassifyError is case-sensitive, so "Validation" (uppercase) won't match "validation failed"

		// Server not found (Priority 3)
		{"server not found", errors.New("server not found: test-server"), ErrTypeServerNotFound},
		{"not found generic", errors.New("resource not found"), ErrTypeServerNotFound},

		// Timeout errors (Priority 4)
		{"timeout", errors.New("connection timeout after 30s"), ErrTypeTimeout},
		{"deadline exceeded", errors.New("context deadline exceeded"), ErrTypeTimeout},

		// Auth failures (Priority 5)
		{"auth error", errors.New("authentication failed"), ErrTypeAuthFailure},
		{"401 error", errors.New("HTTP 401 Unauthorized"), ErrTypeAuthFailure},

		// Session errors (Priority 6)
		{"session lowercase", errors.New("session expired"), ErrTypeSessionError},
		{"Session uppercase", errors.New("Session invalid"), ErrTypeSessionError}, // "Session" (capital S) is also matched

		// Connection failed (Priority 7)
		{"connection error", errors.New("connection refused"), ErrTypeConnectionFailed},
		{"dial error", errors.New("dial tcp: no such host"), ErrTypeConnectionFailed},

		// MCP protocol errors (Priority 8)
		{"MCP error", errors.New("MCP protocol error"), ErrTypeMCPProtocol},
		{"JSON-RPC error", errors.New("JSON-RPC error code -32600"), ErrTypeMCPProtocol},

		// Parse errors (Priority 9)
		{"unmarshal error", errors.New("json: cannot unmarshal string"), ErrTypeParseError},
		{"decode error", errors.New("failed to decode response"), ErrTypeParseError},

		// Tool execution errors (Priority 10)
		{"tool execution error", errors.New("Tool execution error: invalid arguments"), ErrTypeToolExecution},

		// Unknown errors (default)
		{"unknown error", errors.New("something unexpected happened"), ErrTypeUnknown},
		{"generic error", errors.New("error occurred"), ErrTypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyError(tt.err)
			assert.Equal(t, tt.expected, result, "ClassifyError(%v) should return %s", tt.err, tt.expected)
		})
	}
}

// TestClassifyError_NilError specifically tests nil error handling.
func TestClassifyError_NilError(t *testing.T) {
	result := ClassifyError(nil)
	assert.Equal(t, "", result, "ClassifyError(nil) should return empty string")
}

// TestClassifyError_PriorityOrder verifies that higher priority classifications take precedence.
func TestClassifyError_PriorityOrder(t *testing.T) {
	// Error message contains both "blocked" (SSRF) and "timeout" keywords
	// SSRF should win because it has higher priority
	err := errors.New("connection blocked due to timeout")
	result := ClassifyError(err)
	assert.Equal(t, ErrTypeSSRFBlocked, result, "SSRF should take priority over timeout")

	// Error message contains both "validation failed" and "not found"
	// Validation should win because it has higher priority
	err = errors.New("validation failed: server not found")
	result = ClassifyError(err)
	assert.Equal(t, ErrTypeValidationError, result, "Validation should take priority over not found")
}

// TestIsValidErrorType verifies all 11 error types are valid.
func TestIsValidErrorType(t *testing.T) {
	// All 11 defined error types should be valid
	validTypes := []string{
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

	for _, errType := range validTypes {
		t.Run(errType, func(t *testing.T) {
			assert.True(t, IsValidErrorType(errType), "IsValidErrorType(%s) should return true", errType)
		})
	}

	// Verify count matches expected (11 types)
	assert.Len(t, ValidErrorTypes, 11, "Should have exactly 11 valid error types")
}

// TestIsValidErrorType_Invalid verifies invalid error types are rejected.
func TestIsValidErrorType_Invalid(t *testing.T) {
	invalidTypes := []string{
		"",
		"invalid_type",
		"random",
		"VALIDATION_ERROR", // wrong case
		"validation-error", // wrong separator
	}

	for _, errType := range invalidTypes {
		t.Run(errType, func(t *testing.T) {
			assert.False(t, IsValidErrorType(errType), "IsValidErrorType(%q) should return false", errType)
		})
	}
}

// TestErrorCode_AllTypes verifies all error types have HTTP-like codes.
func TestErrorCode_AllTypes(t *testing.T) {
	// All valid error types should have a code in the ErrorCode map
	for _, errType := range ValidErrorTypes {
		t.Run(errType, func(t *testing.T) {
			code, exists := ErrorCode[errType]
			assert.True(t, exists, "ErrorCode should have entry for %s", errType)
			assert.Greater(t, code, 0, "Error code for %s should be positive", errType)
		})
	}

	// Verify ErrorCode map has same length as ValidErrorTypes
	assert.Len(t, ErrorCode, 11, "ErrorCode should have 11 entries")
}

// TestErrorCode_HTTPCategories verifies error codes follow HTTP conventions.
func TestErrorCode_HTTPCategories(t *testing.T) {
	// Client errors (4xx)
	assert.Equal(t, 400, ErrorCode[ErrTypeValidationError], "Validation errors should be 400")
	assert.Equal(t, 404, ErrorCode[ErrTypeServerNotFound], "Not found errors should be 404")
	assert.Equal(t, 403, ErrorCode[ErrTypeSSRFBlocked], "SSRF blocked should be 403")
	assert.Equal(t, 401, ErrorCode[ErrTypeAuthFailure], "Auth failures should be 401")

	// Server errors (5xx)
	assert.Equal(t, 503, ErrorCode[ErrTypeConnectionFailed], "Connection failed should be 503")
	assert.Equal(t, 504, ErrorCode[ErrTypeTimeout], "Timeouts should be 504")
	assert.Equal(t, 502, ErrorCode[ErrTypeMCPProtocol], "MCP protocol errors should be 502")
	assert.Equal(t, 500, ErrorCode[ErrTypeSessionError], "Session errors should be 500")
	assert.Equal(t, 500, ErrorCode[ErrTypeToolExecution], "Tool execution should be 500")
	assert.Equal(t, 500, ErrorCode[ErrTypeParseError], "Parse errors should be 500")
	assert.Equal(t, 500, ErrorCode[ErrTypeUnknown], "Unknown errors should be 500")
}

// TestValidErrorTypes_Coverage verifies the ValidErrorTypes slice contains all types.
func TestValidErrorTypes_Coverage(t *testing.T) {
	expectedTypes := map[string]bool{
		ErrTypeValidationError:  false,
		ErrTypeServerNotFound:   false,
		ErrTypeSSRFBlocked:      false,
		ErrTypeTimeout:          false,
		ErrTypeAuthFailure:      false,
		ErrTypeConnectionFailed: false,
		ErrTypeSessionError:     false,
		ErrTypeMCPProtocol:      false,
		ErrTypeToolExecution:    false,
		ErrTypeParseError:       false,
		ErrTypeUnknown:          false,
	}

	for _, errType := range ValidErrorTypes {
		if _, exists := expectedTypes[errType]; exists {
			expectedTypes[errType] = true
		} else {
			t.Errorf("Unexpected error type in ValidErrorTypes: %s", errType)
		}
	}

	for errType, found := range expectedTypes {
		assert.True(t, found, "ValidErrorTypes should contain %s", errType)
	}
}
