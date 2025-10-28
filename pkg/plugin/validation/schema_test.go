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

package validation

import (
	"strings"
	"testing"
)

func TestValidate_NilSchema(t *testing.T) {
	v := NewSchemaValidator()

	// Nil inputSchema allows any arguments (per CONTEXT.md)
	err := v.Validate("test-tool", nil, map[string]interface{}{
		"query": "test",
		"count": 42,
	})

	if err != nil {
		t.Errorf("Nil schema should allow any arguments, got error: %v", err)
	}

	// Empty arguments with nil schema should also pass
	err = v.Validate("test-tool", nil, map[string]interface{}{})
	if err != nil {
		t.Errorf("Nil schema should allow empty arguments, got error: %v", err)
	}

	// Nil arguments with nil schema should also pass
	err = v.Validate("test-tool", nil, nil)
	if err != nil {
		t.Errorf("Nil schema should allow nil arguments, got error: %v", err)
	}
}

func TestValidate_ValidArguments(t *testing.T) {
	v := NewSchemaValidator()

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type": "string",
			},
		},
		"required": []interface{}{"query"},
	}

	args := map[string]interface{}{
		"query": "test search",
	}

	err := v.Validate("search-tool", schema, args)
	if err != nil {
		t.Errorf("Valid arguments should pass, got error: %v", err)
	}
}

func TestValidate_MissingRequired(t *testing.T) {
	v := NewSchemaValidator()

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type": "string",
			},
			"limit": map[string]interface{}{
				"type": "integer",
			},
		},
		"required": []interface{}{"query"},
	}

	// Empty arguments - missing required "query"
	args := map[string]interface{}{}

	err := v.Validate("search-tool", schema, args)
	if err == nil {
		t.Fatal("Expected error for missing required field, got nil")
	}

	errStr := err.Error()
	// Error should mention the tool name and missing field
	if !strings.Contains(errStr, "search-tool") {
		t.Errorf("Error should mention tool name, got: %v", err)
	}
	// Error message should indicate missing property
	if !strings.Contains(errStr, "missing") || !strings.Contains(errStr, "query") {
		t.Errorf("Error should mention 'missing' and field name, got: %v", err)
	}
}

func TestValidate_WrongType(t *testing.T) {
	v := NewSchemaValidator()

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"count": map[string]interface{}{
				"type": "integer",
			},
		},
		"required": []interface{}{"count"},
	}

	// "count" should be integer, not string
	args := map[string]interface{}{
		"count": "not a number",
	}

	err := v.Validate("count-tool", schema, args)
	if err == nil {
		t.Fatal("Expected error for wrong type, got nil")
	}

	errStr := err.Error()
	// Error should mention the tool and the type issue
	if !strings.Contains(errStr, "count-tool") {
		t.Errorf("Error should mention tool name, got: %v", err)
	}
	if !strings.Contains(errStr, "count") {
		t.Errorf("Error should mention field name, got: %v", err)
	}
}

func TestValidate_ExtraProperties(t *testing.T) {
	v := NewSchemaValidator()

	// Schema only defines "query" property
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type": "string",
			},
		},
		"required": []interface{}{"query"},
		// Note: additionalProperties is NOT set to false
		// Per CONTEXT.md: permissive validation, allow extra properties
	}

	// Arguments include "extra" which is not in schema
	args := map[string]interface{}{
		"query": "test",
		"extra": "should be allowed",
	}

	err := v.Validate("permissive-tool", schema, args)
	if err != nil {
		t.Errorf("Extra properties should be allowed (permissive validation), got error: %v", err)
	}
}

func TestValidate_Caching(t *testing.T) {
	v := NewSchemaValidator()

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type": "string",
			},
		},
	}

	args := map[string]interface{}{
		"name": "test",
	}

	// First call - compiles and caches schema
	err := v.Validate("cached-tool", schema, args)
	if err != nil {
		t.Fatalf("First validation failed: %v", err)
	}

	// Second call - should use cached schema
	err = v.Validate("cached-tool", schema, args)
	if err != nil {
		t.Fatalf("Second validation (cached) failed: %v", err)
	}

	// Verify cache has the schema
	v.mu.RLock()
	_, exists := v.cache["cached-tool"]
	v.mu.RUnlock()
	if !exists {
		t.Error("Schema should be cached after validation")
	}

	// Invalidate cache for this tool
	v.InvalidateCache("cached-tool")

	v.mu.RLock()
	_, exists = v.cache["cached-tool"]
	v.mu.RUnlock()
	if exists {
		t.Error("Schema should be removed from cache after InvalidateCache")
	}

	// Third call - should recompile schema
	err = v.Validate("cached-tool", schema, args)
	if err != nil {
		t.Fatalf("Third validation (after invalidation) failed: %v", err)
	}

	// Verify it's cached again
	v.mu.RLock()
	_, exists = v.cache["cached-tool"]
	v.mu.RUnlock()
	if !exists {
		t.Error("Schema should be re-cached after validation")
	}
}

func TestValidate_ClearCache(t *testing.T) {
	v := NewSchemaValidator()

	schema := map[string]interface{}{
		"type": "object",
	}

	// Cache multiple tools
	v.Validate("tool-1", schema, nil)
	v.Validate("tool-2", schema, nil)
	v.Validate("tool-3", schema, nil)

	v.mu.RLock()
	count := len(v.cache)
	v.mu.RUnlock()
	if count != 3 {
		t.Errorf("Expected 3 cached schemas, got %d", count)
	}

	// Clear all
	v.ClearCache()

	v.mu.RLock()
	count = len(v.cache)
	v.mu.RUnlock()
	if count != 0 {
		t.Errorf("Expected 0 cached schemas after ClearCache, got %d", count)
	}
}

func TestValidate_ComplexSchema(t *testing.T) {
	v := NewSchemaValidator()

	// Complex schema with nested objects and arrays
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type": "string",
			},
			"config": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"timeout": map[string]interface{}{
						"type": "integer",
					},
					"retries": map[string]interface{}{
						"type": "integer",
					},
				},
				"required": []interface{}{"timeout"},
			},
			"tags": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "string",
				},
			},
		},
		"required": []interface{}{"name", "config"},
	}

	// Valid complex arguments
	validArgs := map[string]interface{}{
		"name": "my-service",
		"config": map[string]interface{}{
			"timeout": 30,
			"retries": 3,
		},
		"tags": []interface{}{"prod", "critical"},
	}

	err := v.Validate("complex-tool", schema, validArgs)
	if err != nil {
		t.Errorf("Valid complex arguments should pass, got error: %v", err)
	}

	// Invalid - missing nested required field
	invalidArgs := map[string]interface{}{
		"name": "my-service",
		"config": map[string]interface{}{
			// Missing "timeout" which is required
			"retries": 3,
		},
	}

	err = v.Validate("complex-tool-2", schema, invalidArgs)
	if err == nil {
		t.Error("Missing nested required field should fail validation")
	}
}

func TestValidate_MultipleRequiredFields(t *testing.T) {
	v := NewSchemaValidator()

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"from": map[string]interface{}{
				"type": "string",
			},
			"to": map[string]interface{}{
				"type": "string",
			},
			"amount": map[string]interface{}{
				"type": "number",
			},
		},
		"required": []interface{}{"from", "to", "amount"},
	}

	// All required fields present
	validArgs := map[string]interface{}{
		"from":   "alice",
		"to":     "bob",
		"amount": 100.5,
	}

	err := v.Validate("transfer-tool", schema, validArgs)
	if err != nil {
		t.Errorf("All required fields present should pass, got error: %v", err)
	}

	// Missing one required field
	missingOne := map[string]interface{}{
		"from":   "alice",
		"amount": 100.5,
		// Missing "to"
	}

	err = v.Validate("transfer-tool-2", schema, missingOne)
	if err == nil {
		t.Error("Missing required field 'to' should fail validation")
	}
}

func TestValidate_ArrayValidation(t *testing.T) {
	v := NewSchemaValidator()

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"ids": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "integer",
				},
			},
		},
		"required": []interface{}{"ids"},
	}

	// Valid array of integers
	validArgs := map[string]interface{}{
		"ids": []interface{}{1, 2, 3},
	}

	err := v.Validate("array-tool", schema, validArgs)
	if err != nil {
		t.Errorf("Valid array should pass, got error: %v", err)
	}

	// Invalid array (contains string instead of integer)
	invalidArgs := map[string]interface{}{
		"ids": []interface{}{1, "two", 3},
	}

	err = v.Validate("array-tool-2", schema, invalidArgs)
	if err == nil {
		t.Error("Array with wrong item type should fail validation")
	}
}

func TestValidate_BooleanType(t *testing.T) {
	v := NewSchemaValidator()

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"enabled": map[string]interface{}{
				"type": "boolean",
			},
		},
		"required": []interface{}{"enabled"},
	}

	// Valid boolean
	validArgs := map[string]interface{}{
		"enabled": true,
	}

	err := v.Validate("bool-tool", schema, validArgs)
	if err != nil {
		t.Errorf("Valid boolean should pass, got error: %v", err)
	}

	// Invalid - string instead of boolean
	invalidArgs := map[string]interface{}{
		"enabled": "true",
	}

	err = v.Validate("bool-tool-2", schema, invalidArgs)
	if err == nil {
		t.Error("String instead of boolean should fail validation")
	}
}

func TestValidate_NullValue(t *testing.T) {
	v := NewSchemaValidator()

	// Schema that allows null for a property
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type": []interface{}{"string", "null"},
			},
		},
		"required": []interface{}{"name"},
	}

	// Null value when null is allowed
	validArgs := map[string]interface{}{
		"name": nil,
	}

	err := v.Validate("nullable-tool", schema, validArgs)
	if err != nil {
		t.Errorf("Null value should be allowed when type includes null, got error: %v", err)
	}

	// String value also valid
	stringArgs := map[string]interface{}{
		"name": "test",
	}

	err = v.Validate("nullable-tool", schema, stringArgs)
	if err != nil {
		t.Errorf("String value should also be allowed, got error: %v", err)
	}
}

func TestValidate_MinMaxConstraints(t *testing.T) {
	v := NewSchemaValidator()

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"count": map[string]interface{}{
				"type":    "integer",
				"minimum": 1,
				"maximum": 100,
			},
		},
		"required": []interface{}{"count"},
	}

	// Valid value within range
	validArgs := map[string]interface{}{
		"count": 50,
	}

	err := v.Validate("range-tool", schema, validArgs)
	if err != nil {
		t.Errorf("Value within range should pass, got error: %v", err)
	}

	// Value below minimum
	belowMin := map[string]interface{}{
		"count": 0,
	}

	err = v.Validate("range-tool-2", schema, belowMin)
	if err == nil {
		t.Error("Value below minimum should fail validation")
	}

	// Value above maximum
	aboveMax := map[string]interface{}{
		"count": 101,
	}

	err = v.Validate("range-tool-3", schema, aboveMax)
	if err == nil {
		t.Error("Value above maximum should fail validation")
	}
}

func TestValidate_EnumConstraint(t *testing.T) {
	v := NewSchemaValidator()

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"status": map[string]interface{}{
				"type": "string",
				"enum": []interface{}{"pending", "active", "completed"},
			},
		},
		"required": []interface{}{"status"},
	}

	// Valid enum value
	validArgs := map[string]interface{}{
		"status": "active",
	}

	err := v.Validate("enum-tool", schema, validArgs)
	if err != nil {
		t.Errorf("Valid enum value should pass, got error: %v", err)
	}

	// Invalid enum value
	invalidArgs := map[string]interface{}{
		"status": "unknown",
	}

	err = v.Validate("enum-tool-2", schema, invalidArgs)
	if err == nil {
		t.Error("Invalid enum value should fail validation")
	}
}

func TestNewSchemaValidator(t *testing.T) {
	v := NewSchemaValidator()

	if v == nil {
		t.Fatal("NewSchemaValidator returned nil")
	}

	if v.cache == nil {
		t.Error("SchemaValidator should have initialized cache")
	}

	if len(v.cache) != 0 {
		t.Errorf("New validator should have empty cache, got %d entries", len(v.cache))
	}
}

func TestValidate_ErrorMessageFormat(t *testing.T) {
	v := NewSchemaValidator()

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"email": map[string]interface{}{
				"type": "string",
			},
		},
		"required": []interface{}{"email"},
	}

	args := map[string]interface{}{
		// Missing required "email"
	}

	err := v.Validate("email-tool", schema, args)
	if err == nil {
		t.Fatal("Expected validation error")
	}

	errStr := err.Error()

	// Error should be structured for LLM self-correction
	// Should include: tool name, property name, reason
	if !strings.Contains(errStr, "email-tool") {
		t.Errorf("Error should contain tool name for context, got: %s", errStr)
	}
	if !strings.Contains(errStr, "validation failed") {
		t.Errorf("Error should indicate validation failure, got: %s", errStr)
	}
}
