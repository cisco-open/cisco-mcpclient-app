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

import { getBackendSrv } from '@grafana/runtime';
import { lastValueFrom } from 'rxjs';
import { MCPServerConfig, MCPServerStatus, MCPTool } from '../types/mcp';

interface DirectConnectionAuth {
  authToken?: string;
  authUser?: string;
  authPass?: string;
  username?: string;
  password?: string;
}

export class MCPService {
  private static toServerPayload(config: MCPServerConfig): Record<string, unknown> {
    const authType = config.authType ?? config.auth?.type;
    const authToken = config.authToken ?? config.auth?.token;
    const authUser = config.authUser ?? config.username ?? config.auth?.username;
    const authPass = config.authPass ?? config.password ?? config.auth?.password;

    const payload: Record<string, unknown> = {};
    if (config.id !== undefined) {
      payload.id = config.id;
    }
    if (config.name !== undefined) {
      payload.name = config.name;
    }
    if (config.url !== undefined) {
      payload.url = config.url;
    }
    if (config.type !== undefined) {
      payload.type = config.type;
    }
    if (config.enabled !== undefined) {
      payload.enabled = config.enabled;
    }
    if (config.description !== undefined) {
      payload.description = config.description;
    }
    if (authType !== undefined) {
      payload.authType = authType;
    }

    if (authToken !== undefined) {
      payload.authToken = authToken;
    }
    if (authUser !== undefined) {
      payload.authUser = authUser;
    }
    if (authPass !== undefined) {
      payload.authPass = authPass;
    }

    return payload;
  }

  private static toDirectAuthPayload(auth: DirectConnectionAuth): Record<string, unknown> {
    const payload: Record<string, unknown> = {};
    const authUser = auth.authUser ?? auth.username;
    const authPass = auth.authPass ?? auth.password;

    if (auth.authToken !== undefined) {
      payload.authToken = auth.authToken;
    }
    if (authUser !== undefined) {
      payload.authUser = authUser;
    }
    if (authPass !== undefined) {
      payload.authPass = authPass;
    }

    return payload;
  }

  /**
   * Check if MCP service backend is available
   */
  static async isServiceAvailable(): Promise<boolean> {
    console.log('MCPService.isServiceAvailable() called');
    try {
      const response = await lastValueFrom(getBackendSrv().fetch({
        url: '/api/plugins/cisco-mcpclient-app/resources/ping',
        method: 'GET',
      }));
      console.log('isServiceAvailable response:', response);
      return response.status === 200;
    } catch (error) {
      console.error('Error checking service availability:', error);
      return false;
    }
  }

  /**
   * Get all server configurations from backend
   */
  static async getServerConfigs(): Promise<MCPServerConfig[]> {
    try {
      const response = await lastValueFrom(getBackendSrv().fetch({
        url: '/api/plugins/cisco-mcpclient-app/resources/servers',
        method: 'GET',
      }));
      return (response.data as any)?.servers || [];
    } catch (error) {
      console.error('Error fetching server configs:', error);
      return [];
    }
  }

  /**
   * Get server statuses from backend (simplified)
   */
  static async getServerStatuses(): Promise<MCPServerStatus[]> {
    try {
      const servers = await MCPService.getServerConfigs();
      return servers.map((server: any) => ({
        id: server.id,
        connected: server.status === 'connected',
        lastChecked: new Date(),
        error: server.status === 'connected' ? undefined : `Server status: ${server.status || 'unknown'}`,
        tools: server.tools || [],
        capabilities: server.capabilities
      }));
    } catch (error) {
      console.error('Error fetching server statuses:', error);
      return [];
    }
  }

  /**
   * Test connection to a specific server and get detailed status including tools
   */
  static async getServerDetailedStatus(serverId: string): Promise<{
    connected: boolean;
    tools: MCPTool[];
    capabilities: string[];
    message?: string;
  }> {
    console.log('MCPService.getServerDetailedStatus() called with serverId:', serverId);
    try {
      const response = await lastValueFrom(getBackendSrv().fetch({
        url: `/api/plugins/cisco-mcpclient-app/resources/servers/${serverId}/test`,
        method: 'POST',
      }));
      console.log('getServerDetailedStatus response:', response);

      const data = response.data as any;
      return {
        connected: data?.status === 'connected',
        tools: data?.tools || [],
        capabilities: data?.capabilities || [],
        message: data?.message
      };
    } catch (error) {
      console.error('Error getting server detailed status:', error);
      return {
        connected: false,
        tools: [],
        capabilities: [],
        message: error instanceof Error ? error.message : 'Connection failed'
      };
    }
  }

  /**
   * Test connection to a specific server (via backend)
   */
  static async testServerConnection(serverId: string): Promise<boolean> {
    console.log('MCPService.testServerConnection() called with serverId:', serverId);
    try {
      const response = await lastValueFrom(getBackendSrv().fetch({
        url: `/api/plugins/cisco-mcpclient-app/resources/servers/${serverId}/test`,
        method: 'POST',
      }));
      console.log('testServerConnection response:', response);
      return (response.data as any)?.status === 'connected';
    } catch (error) {
      console.error('Error testing server connection:', error);
      return false;
    }
  }

  /**
   * Test connection directly without creating a server (for configuration page)
   */
  static async testDirectConnection(
    url: string,
    authType = 'none',
    auth: DirectConnectionAuth = {}
  ): Promise<boolean> {
    console.log('MCPService.testDirectConnection() called with url:', url);
    try {
      const response = await lastValueFrom(getBackendSrv().fetch({
        url: `/api/plugins/cisco-mcpclient-app/resources/test-connection`,
        method: 'POST',
        data: {
          url,
          authType,
          ...MCPService.toDirectAuthPayload(auth),
        },
      }));
      console.log('testDirectConnection response:', response);
      return (response.data as any)?.status === 'connected';
    } catch (error) {
      console.error('Error testing direct connection:', error);
      return false;
    }
  }

  /**
   * Add a new server configuration
   */
  static async addServer(config: MCPServerConfig): Promise<MCPServerConfig> {
    console.log('MCPService.addServer() called with config:', config);
    try {
      const response = await lastValueFrom(getBackendSrv().fetch({
        url: '/api/plugins/cisco-mcpclient-app/resources/servers',
        method: 'POST',
        data: MCPService.toServerPayload(config),
      }));
      console.log('addServer response:', response);
      return response.data as MCPServerConfig;
    } catch (error) {
      console.error('Error adding server:', error);
      throw error;
    }
  }

  /**
   * Update an existing server configuration
   */
  static async updateServer(config: MCPServerConfig): Promise<MCPServerConfig> {
    console.log('MCPService.updateServer() called with config:', config);
    try {
      const response = await lastValueFrom(getBackendSrv().fetch({
        url: `/api/plugins/cisco-mcpclient-app/resources/servers/${config.id}`,
        method: 'PUT',
        data: MCPService.toServerPayload(config),
      }));
      console.log('updateServer response:', response);
      return response.data as MCPServerConfig;
    } catch (error) {
      console.error('Error updating server:', error);
      throw error;
    }
  }

  /**
   * Get a single server configuration by ID
   */
  static async getServer(serverId: string): Promise<MCPServerConfig | null> {
    console.log('MCPService.getServer() called with serverId:', serverId);
    try {
      const response = await lastValueFrom(getBackendSrv().fetch({
        url: `/api/plugins/cisco-mcpclient-app/resources/servers/${serverId}`,
        method: 'GET',
      }));
      console.log('getServer response:', response);
      return response.data as MCPServerConfig;
    } catch (error) {
      console.error('Error getting server:', error);
      return null;
    }
  }

  /**
   * Remove a server configuration
   */
  static async removeServer(serverId: string): Promise<void> {
    console.log('MCPService.removeServer() called with serverId:', serverId);
    try {
      await lastValueFrom(getBackendSrv().fetch({
        url: `/api/plugins/cisco-mcpclient-app/resources/servers/${serverId}`,
        method: 'DELETE',
      }));
      console.log('removeServer completed successfully');
    } catch (error) {
      console.error('Error removing server:', error);
      throw error;
    }
  }

  /**
   * Get tools available from a specific server
   */
  static async getServerTools(serverId: string): Promise<MCPTool[]> {
    console.log('MCPService.getServerTools() called with serverId:', serverId);
    try {
      const servers = await MCPService.getServerConfigs();
      const server = servers.find((s: any) => s.id === serverId);

      if (!server) {
        console.warn(`Server with ID ${serverId} not found`);
        return [];
      }

      return server.tools || [];
    } catch (error) {
      console.error('Error fetching server tools:', error);
      return [];
    }
  }

  /**
   * Get tools with server information included
   */
  static async getToolsWithServerInfo(): Promise<MCPTool[]> {
    console.log('MCPService.getToolsWithServerInfo() called');
    try {
      const servers = await MCPService.getServerConfigs();
      const allTools: MCPTool[] = [];

      servers.forEach((server: MCPServerConfig) => {
        if (server.enabled && server.tools) {
          server.tools.forEach((tool: MCPTool) => {
            allTools.push({
              ...tool,
              serverId: server.id
            });
          });
        }
      });

      console.log('getToolsWithServerInfo result:', allTools);
      return allTools;
    } catch (error) {
      console.error('Error fetching tools with server info:', error);
      return [];
    }
  }

  static async getAllTools(): Promise<MCPTool[]> {
    console.log('MCPService.getAllTools() called');
    try {
      const response = await lastValueFrom(getBackendSrv().fetch({
        url: '/api/plugins/cisco-mcpclient-app/resources/tools',
        method: 'GET',
      }));
      console.log('getAllTools response:', response);

      const data = response.data as any;
      return data?.tools || [];
    } catch (error) {
      console.error('Error fetching all tools:', error);
      return [];
    }
  }

  /**
   * Call a tool on an MCP server
   */
  static async callTool(
    toolName: string,
    args: Record<string, unknown>,
    serverId?: string
  ): Promise<{ success: boolean; content: string; error?: string }> {
    console.log('MCPService.callTool() called with:', { toolName, args, serverId });
    try {
      const response = await lastValueFrom(getBackendSrv().fetch({
        url: '/api/plugins/cisco-mcpclient-app/resources/tools/call',
        method: 'POST',
        data: {
          tool_name: toolName,
          arguments: args,
          server_id: serverId,
        },
      }));
      console.log('callTool response:', response);
      const data = response.data as { success: boolean; content: string; error?: string };
      return {
        success: data.success ?? false,
        content: data.content ?? '',
        error: data.error,
      };
    } catch (error) {
      console.error('Error calling tool:', error);
      return {
        success: false,
        content: '',
        error: error instanceof Error ? error.message : 'Tool call failed',
      };
    }
  }

  /**
   * Get global MCP client configuration
   */
  static async getConfig(): Promise<Record<string, unknown>> {
    console.log('MCPService.getConfig() called');
    try {
      const response = await lastValueFrom(getBackendSrv().fetch({
        url: '/api/plugins/cisco-mcpclient-app/resources/config',
        method: 'GET',
      }));
      console.log('getConfig response:', response);
      return (response.data as Record<string, unknown>) ?? {};
    } catch (error) {
      console.error('Error getting config:', error);
      return {};
    }
  }

  /**
   * Set global MCP client configuration
   */
  static async setConfig(config: Record<string, unknown>): Promise<Record<string, unknown>> {
    console.log('MCPService.setConfig() called with:', config);
    try {
      const response = await lastValueFrom(getBackendSrv().fetch({
        url: '/api/plugins/cisco-mcpclient-app/resources/config',
        method: 'POST',
        data: config,
      }));
      console.log('setConfig response:', response);
      return (response.data as Record<string, unknown>) ?? {};
    } catch (error) {
      console.error('Error setting config:', error);
      throw error;
    }
  }
}

// Default export
export default MCPService;
