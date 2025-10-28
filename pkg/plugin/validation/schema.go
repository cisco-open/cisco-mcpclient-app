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

// Package validation provides JSON Schema validation for MCP tool arguments.
package validation

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// SchemaValidator validates MCP tool arguments against JSON Schema definitions.
// It caches compiled schemas for performance.
type SchemaValidator struct {
	cache map[string]*jsonschema.Schema
	mu    sync.RWMutex
}

// NewSchemaValidator creates a new SchemaValidator with an empty cache.
func NewSchemaValidator() *SchemaValidator {
	return &SchemaValidator{
		cache: make(map[string]*jsonschema.Schema),
	}
}

// Validate validates tool arguments against the tool's inputSchema.
// If inputSchema is nil, any arguments are allowed (trusts the MCP server).
// Returns nil on success, or an error with details about which field failed and why.
func (v *SchemaValidator) Validate(toolName string, inputSchema map[string]interface{}, arguments map[string]interface{}) error {
	// Per CONTEXT.md: If tool has no inputSchema, allow any arguments
	if inputSchema == nil {
		return nil
	}

	// Check cache with read lock
	v.mu.RLock()
	schema, exists := v.cache[toolName]
	v.mu.RUnlock()

	if !exists {
		// Compile schema
		compiledSchema, err := v.compileSchema(toolName, inputSchema)
		if err != nil {
			return err
		}

		// Cache with write lock
		v.mu.Lock()
		// Double-check in case another goroutine compiled while we were waiting
		if existingSchema, ok := v.cache[toolName]; ok {
			schema = existingSchema
		} else {
			v.cache[toolName] = compiledSchema
			schema = compiledSchema
		}
		v.mu.Unlock()
	}

	// Validate arguments
	if err := schema.Validate(arguments); err != nil {
		return v.formatValidationError(toolName, err)
	}

	return nil
}

// compileSchema compiles an inputSchema map into a jsonschema.Schema.
func (v *SchemaValidator) compileSchema(toolName string, inputSchema map[string]interface{}) (*jsonschema.Schema, error) {
	// Marshal schema to JSON
	schemaJSON, err := json.Marshal(inputSchema)
	if err != nil {
		return nil, fmt.Errorf("validation failed for tool %q: invalid schema: %w", toolName, err)
	}

	// Create compiler
	c := jsonschema.NewCompiler()

	// Parse schema from JSON
	schemaDoc, err := jsonschema.UnmarshalJSON(strings.NewReader(string(schemaJSON)))
	if err != nil {
		return nil, fmt.Errorf("validation failed for tool %q: invalid JSON schema: %w", toolName, err)
	}

	// Add schema as resource
	resourceName := fmt.Sprintf("%s-schema.json", toolName)
	if err := c.AddResource(resourceName, schemaDoc); err != nil {
		return nil, fmt.Errorf("validation failed for tool %q: failed to add schema resource: %w", toolName, err)
	}

	// Compile the schema
	schema, err := c.Compile(resourceName)
	if err != nil {
		return nil, fmt.Errorf("validation failed for tool %q: failed to compile schema: %w", toolName, err)
	}

	return schema, nil
}

// formatValidationError formats a validation error with detailed information.
func (v *SchemaValidator) formatValidationError(toolName string, err error) error {
	// Try to cast to ValidationError for detailed output
	if ve, ok := err.(*jsonschema.ValidationError); ok {
		// Get the detailed error message from the validation error
		details := extractValidationDetails(ve)
		if details != "" {
			return fmt.Errorf("validation failed for tool %q: %s", toolName, details)
		}
	}

	return fmt.Errorf("validation failed for tool %q: %v", toolName, err)
}

// defaultPrinter is used for localizing validation error messages.
var defaultPrinter = message.NewPrinter(language.English)

// extractValidationDetails extracts human-readable details from a ValidationError.
func extractValidationDetails(ve *jsonschema.ValidationError) string {
	// If there are nested causes, use the first one for a more specific message
	if len(ve.Causes) > 0 {
		return extractValidationDetails(ve.Causes[0])
	}

	// Build the error message from the validation error
	var parts []string

	// Include the instance location (which field) - InstanceLocation is []string
	if len(ve.InstanceLocation) > 0 {
		location := strings.Join(ve.InstanceLocation, "/")
		parts = append(parts, fmt.Sprintf("property %q", location))
	}

	// Include the error message (what went wrong)
	if ve.ErrorKind != nil {
		parts = append(parts, ve.ErrorKind.LocalizedString(defaultPrinter))
	}

	if len(parts) == 0 {
		return ve.Error()
	}

	return strings.Join(parts, ": ")
}

// InvalidateCache removes a single tool's schema from the cache.
// Use this when a tool's schema changes (e.g., tools/list_changed notification).
func (v *SchemaValidator) InvalidateCache(toolName string) {
	v.mu.Lock()
	delete(v.cache, toolName)
	v.mu.Unlock()
}

// ClearCache removes all schemas from the cache.
// Use this when reconnecting to an MCP server.
func (v *SchemaValidator) ClearCache() {
	v.mu.Lock()
	v.cache = make(map[string]*jsonschema.Schema)
	v.mu.Unlock()
}
