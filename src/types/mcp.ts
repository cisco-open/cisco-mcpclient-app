/**
 * Copyright 2025 Cisco Systems, Inc. and its affiliates
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

// MCP (Model Context Protocol) types and interfaces

export interface MCPServerAuth {
  type: 'none' | 'bearer' | 'basic' | 'api-key';
  token?: string;
  username?: string;
  password?: string;
  apiKey?: string;
  header?: string;
}

export interface MCPServerConfig {
  id?: string;
  name: string;
  url: string;
  type: 'local' | 'remote';
  enabled: boolean;
  description?: string;
  auth?: MCPServerAuth;
  // Flat auth fields for backward compatibility with existing pages
  authType?: 'none' | 'bearer' | 'basic' | 'api-key';
  authToken?: string;
  authUser?: string;
  authPass?: string;
  username?: string;
  password?: string;
  capabilities?: string[];
  tools?: MCPTool[];
  status?: 'connected' | 'disconnected' | 'error';
  lastConnected?: Date;
  metadata?: Record<string, any>;
}

export interface MCPTool {
  name: string;
  description?: string;
  inputSchema?: object;
  serverName?: string;
  serverId?: string;
  parameters?: Record<string, any>;
}

export interface MCPCapabilities {
  tools?: {
    listChanged?: boolean;
  };
  prompts?: {
    listChanged?: boolean;
  };
  resources?: {
    subscribe?: boolean;
    listChanged?: boolean;
  };
  logging?: Record<string, any>;
}

export interface MCPServerStatus {
  id?: string;
  connected: boolean;
  lastSeen?: Date;
  lastChecked?: Date;
  capabilities?: MCPCapabilities;
  tools?: MCPTool[];
  error?: string;
}

export interface MCPConnection {
  id: string;
  server: MCPServerConfig;
  status: MCPServerStatus;
  client?: any; // MCP client instance
}

// Configuration file formats
export interface MCPConfigFile {
  version: string;
  servers: MCPServerConfig[];
  settings?: {
    autoConnect?: boolean;
    retryAttempts?: number;
    timeout?: number;
  };
}

// API response types
export interface MCPApiResponse<T = any> {
  success: boolean;
  data?: T;
  error?: string;
  message?: string;
}

export interface MCPServersResponse extends MCPApiResponse {
  data: {
    servers: MCPServerConfig[];
    total: number;
  };
}

export interface MCPToolsResponse extends MCPApiResponse {
  data: {
    tools: MCPTool[];
    servers: string[];
  };
}

// Event types for real-time updates
export interface MCPServerEvent {
  type: 'connected' | 'disconnected' | 'error' | 'tools_updated';
  serverId: string;
  data?: any;
  timestamp: Date;
}

// Validation schemas
export interface MCPValidationError {
  field: string;
  message: string;
  code: string;
}

export interface MCPValidationResult {
  valid: boolean;
  errors: MCPValidationError[];
}
