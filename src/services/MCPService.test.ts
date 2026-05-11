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

import { of, throwError } from 'rxjs';
import { MCPService } from './MCPService';
import { MCPServerConfig, MCPTool } from '../types/mcp';

// Mock @grafana/runtime
const mockFetch = jest.fn();
jest.mock('@grafana/runtime', () => ({
  getBackendSrv: () => ({ fetch: mockFetch }),
}));

describe('MCPService', () => {
  beforeEach(() => {
    mockFetch.mockReset();
    // Suppress console.log and console.error during tests
    jest.spyOn(console, 'log').mockImplementation(() => {});
    jest.spyOn(console, 'error').mockImplementation(() => {});
    jest.spyOn(console, 'warn').mockImplementation(() => {});
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  // ====================
  // isServiceAvailable
  // ====================
  describe('isServiceAvailable', () => {
    it('returns true when backend responds with 200', async () => {
      mockFetch.mockReturnValue(of({ status: 200, data: {} }));

      const result = await MCPService.isServiceAvailable();

      expect(result).toBe(true);
      expect(mockFetch).toHaveBeenCalledWith(
        expect.objectContaining({
          url: '/api/plugins/cisco-mcpclient-app/resources/ping',
          method: 'GET',
        })
      );
    });

    it('returns false when backend responds with non-200', async () => {
      mockFetch.mockReturnValue(of({ status: 503, data: {} }));

      const result = await MCPService.isServiceAvailable();

      expect(result).toBe(false);
    });

    it('returns false on network error', async () => {
      mockFetch.mockReturnValue(throwError(() => new Error('Network error')));

      const result = await MCPService.isServiceAvailable();

      expect(result).toBe(false);
    });
  });

  // ====================
  // getServerConfigs
  // ====================
  describe('getServerConfigs', () => {
    it('returns servers from backend', async () => {
      const mockServers: MCPServerConfig[] = [
        { id: '1', name: 'Server 1', url: 'http://server1.local', type: 'remote', enabled: true },
        { id: '2', name: 'Server 2', url: 'http://server2.local', type: 'remote', enabled: false },
      ];
      mockFetch.mockReturnValue(of({ status: 200, data: { servers: mockServers, total: 2 } }));

      const result = await MCPService.getServerConfigs();

      expect(result).toHaveLength(2);
      expect(result[0].name).toBe('Server 1');
      expect(result[1].name).toBe('Server 2');
      expect(mockFetch).toHaveBeenCalledWith(
        expect.objectContaining({
          url: '/api/plugins/cisco-mcpclient-app/resources/servers',
          method: 'GET',
        })
      );
    });

    it('returns empty array when no servers exist', async () => {
      mockFetch.mockReturnValue(of({ status: 200, data: { servers: [], total: 0 } }));

      const result = await MCPService.getServerConfigs();

      expect(result).toEqual([]);
    });

    it('returns empty array on network error', async () => {
      mockFetch.mockReturnValue(throwError(() => new Error('Network error')));

      const result = await MCPService.getServerConfigs();

      expect(result).toEqual([]);
    });

    it('returns empty array when data is undefined', async () => {
      mockFetch.mockReturnValue(of({ status: 200, data: undefined }));

      const result = await MCPService.getServerConfigs();

      expect(result).toEqual([]);
    });
  });

  // ====================
  // getServer
  // ====================
  describe('getServer', () => {
    it('returns server by ID', async () => {
      const mockServer: MCPServerConfig = {
        id: 'server-123',
        name: 'Test Server',
        url: 'http://test.local',
        type: 'remote',
        enabled: true,
      };
      mockFetch.mockReturnValue(of({ status: 200, data: mockServer }));

      const result = await MCPService.getServer('server-123');

      expect(result).toEqual(mockServer);
      expect(mockFetch).toHaveBeenCalledWith(
        expect.objectContaining({
          url: '/api/plugins/cisco-mcpclient-app/resources/servers/server-123',
          method: 'GET',
        })
      );
    });

    it('returns null when server not found', async () => {
      mockFetch.mockReturnValue(throwError(() => new Error('Not found')));

      const result = await MCPService.getServer('nonexistent');

      expect(result).toBeNull();
    });
  });

  // ====================
  // addServer
  // ====================
  describe('addServer', () => {
    it('posts server data to backend and returns created server', async () => {
      const newServer: MCPServerConfig = {
        name: 'New Server',
        url: 'http://new.local',
        type: 'remote',
        enabled: true,
      };
      const createdServer: MCPServerConfig = { ...newServer, id: 'new-123' };
      mockFetch.mockReturnValue(of({ status: 201, data: createdServer }));

      const result = await MCPService.addServer(newServer);

      expect(result).toEqual(createdServer);
      expect(result.id).toBe('new-123');
      expect(mockFetch).toHaveBeenCalledWith(
        expect.objectContaining({
          url: '/api/plugins/cisco-mcpclient-app/resources/servers',
          method: 'POST',
          data: newServer,
        })
      );
    });

    it('throws error on backend failure', async () => {
      const newServer: MCPServerConfig = {
        name: 'New Server',
        url: 'http://new.local',
        type: 'remote',
        enabled: true,
      };
      mockFetch.mockReturnValue(throwError(() => new Error('Server error')));

      await expect(MCPService.addServer(newServer)).rejects.toThrow('Server error');
    });
  });

  // ====================
  // updateServer
  // ====================
  describe('updateServer', () => {
    it('puts updated server data to backend', async () => {
      const updatedServer: MCPServerConfig = {
        id: 'server-123',
        name: 'Updated Server',
        url: 'http://updated.local',
        type: 'remote',
        enabled: false,
      };
      mockFetch.mockReturnValue(of({ status: 200, data: updatedServer }));

      const result = await MCPService.updateServer(updatedServer);

      expect(result).toEqual(updatedServer);
      expect(mockFetch).toHaveBeenCalledWith(
        expect.objectContaining({
          url: '/api/plugins/cisco-mcpclient-app/resources/servers/server-123',
          method: 'PUT',
          data: updatedServer,
        })
      );
    });

    it('throws error on backend failure', async () => {
      const updatedServer: MCPServerConfig = {
        id: 'server-123',
        name: 'Updated Server',
        url: 'http://updated.local',
        type: 'remote',
        enabled: true,
      };
      mockFetch.mockReturnValue(throwError(() => new Error('Validation error')));

      await expect(MCPService.updateServer(updatedServer)).rejects.toThrow('Validation error');
    });
  });

  // ====================
  // removeServer
  // ====================
  describe('removeServer', () => {
    it('deletes server by ID', async () => {
      mockFetch.mockReturnValue(of({ status: 200, data: { success: true } }));

      await expect(MCPService.removeServer('server-123')).resolves.not.toThrow();

      expect(mockFetch).toHaveBeenCalledWith(
        expect.objectContaining({
          url: '/api/plugins/cisco-mcpclient-app/resources/servers/server-123',
          method: 'DELETE',
        })
      );
    });

    it('throws error on backend failure', async () => {
      mockFetch.mockReturnValue(throwError(() => new Error('Delete failed')));

      await expect(MCPService.removeServer('server-123')).rejects.toThrow('Delete failed');
    });
  });

  // ====================
  // testServerConnection
  // ====================
  describe('testServerConnection', () => {
    it('returns true when server is connected', async () => {
      mockFetch.mockReturnValue(
        of({ status: 200, data: { status: 'connected', tools: [], capabilities: [] } })
      );

      const result = await MCPService.testServerConnection('server-123');

      expect(result).toBe(true);
      expect(mockFetch).toHaveBeenCalledWith(
        expect.objectContaining({
          url: '/api/plugins/cisco-mcpclient-app/resources/servers/server-123/test',
          method: 'POST',
        })
      );
    });

    it('returns false when server is disconnected', async () => {
      mockFetch.mockReturnValue(
        of({ status: 200, data: { status: 'disconnected', message: 'Connection refused' } })
      );

      const result = await MCPService.testServerConnection('server-123');

      expect(result).toBe(false);
    });

    it('returns false on network error', async () => {
      mockFetch.mockReturnValue(throwError(() => new Error('Network error')));

      const result = await MCPService.testServerConnection('server-123');

      expect(result).toBe(false);
    });
  });

  // ====================
  // getServerDetailedStatus
  // ====================
  describe('getServerDetailedStatus', () => {
    it('returns detailed status with tools and capabilities', async () => {
      const mockTools: MCPTool[] = [{ name: 'tool1', description: 'Test tool' }];
      mockFetch.mockReturnValue(
        of({
          status: 200,
          data: { status: 'connected', tools: mockTools, capabilities: ['tools', 'prompts'] },
        })
      );

      const result = await MCPService.getServerDetailedStatus('server-123');

      expect(result.connected).toBe(true);
      expect(result.tools).toHaveLength(1);
      expect(result.tools[0].name).toBe('tool1');
      expect(result.capabilities).toContain('tools');
    });

    it('returns disconnected status with error message on failure', async () => {
      mockFetch.mockReturnValue(throwError(() => new Error('Connection timeout')));

      const result = await MCPService.getServerDetailedStatus('server-123');

      expect(result.connected).toBe(false);
      expect(result.tools).toEqual([]);
      expect(result.capabilities).toEqual([]);
      expect(result.message).toBe('Connection timeout');
    });
  });

  // ====================
  // testDirectConnection
  // ====================
  describe('testDirectConnection', () => {
    it('tests connection without creating server', async () => {
      mockFetch.mockReturnValue(of({ status: 200, data: { status: 'connected' } }));

      const result = await MCPService.testDirectConnection('http://test.local', 'bearer', {
        authToken: 'token123',
      });

      expect(result).toBe(true);
      expect(mockFetch).toHaveBeenCalledWith(
        expect.objectContaining({
          url: '/api/plugins/cisco-mcpclient-app/resources/test-connection',
          method: 'POST',
          data: { url: 'http://test.local', authType: 'bearer', authToken: 'token123' },
        })
      );
    });

    it('maps basic auth credentials to authUser/authPass', async () => {
      mockFetch.mockReturnValue(of({ status: 200, data: { status: 'connected' } }));

      const result = await MCPService.testDirectConnection('http://test.local', 'basic', {
        username: 'alice',
        password: 'secret',
      });

      expect(result).toBe(true);
      expect(mockFetch).toHaveBeenCalledWith(
        expect.objectContaining({
          data: {
            url: 'http://test.local',
            authType: 'basic',
            authUser: 'alice',
            authPass: 'secret',
          },
        })
      );
    });

    it('returns false on connection failure', async () => {
      mockFetch.mockReturnValue(of({ status: 200, data: { status: 'disconnected' } }));

      const result = await MCPService.testDirectConnection('http://test.local');

      expect(result).toBe(false);
    });

    it('returns false on network error', async () => {
      mockFetch.mockReturnValue(throwError(() => new Error('Network error')));

      const result = await MCPService.testDirectConnection('http://test.local');

      expect(result).toBe(false);
    });
  });

  // ====================
  // getServerStatuses
  // ====================
  describe('getServerStatuses', () => {
    it('returns statuses for all servers', async () => {
      const mockServers: MCPServerConfig[] = [
        { id: '1', name: 'Server 1', url: 'http://s1.local', type: 'remote', enabled: true, status: 'connected', tools: [{ name: 'tool1' }] },
        { id: '2', name: 'Server 2', url: 'http://s2.local', type: 'remote', enabled: true, status: 'disconnected' },
      ];
      mockFetch.mockReturnValue(of({ status: 200, data: { servers: mockServers } }));

      const result = await MCPService.getServerStatuses();

      expect(result).toHaveLength(2);
      expect(result[0].id).toBe('1');
      expect(result[0].connected).toBe(true);
      expect(result[0].tools).toHaveLength(1);
      expect(result[1].connected).toBe(false);
      expect(result[1].error).toContain('disconnected');
    });

    it('returns empty array on error', async () => {
      mockFetch.mockReturnValue(throwError(() => new Error('Network error')));

      const result = await MCPService.getServerStatuses();

      expect(result).toEqual([]);
    });
  });

  // ====================
  // getServerTools
  // ====================
  describe('getServerTools', () => {
    it('returns tools for specific server', async () => {
      const mockTools: MCPTool[] = [
        { name: 'tool1', description: 'First tool' },
        { name: 'tool2', description: 'Second tool' },
      ];
      const mockServers: MCPServerConfig[] = [
        { id: 'server-1', name: 'Server 1', url: 'http://s1.local', type: 'remote', enabled: true, tools: mockTools },
      ];
      mockFetch.mockReturnValue(of({ status: 200, data: { servers: mockServers } }));

      const result = await MCPService.getServerTools('server-1');

      expect(result).toHaveLength(2);
      expect(result[0].name).toBe('tool1');
      expect(result[1].name).toBe('tool2');
    });

    it('returns empty array when server not found', async () => {
      mockFetch.mockReturnValue(of({ status: 200, data: { servers: [] } }));

      const result = await MCPService.getServerTools('nonexistent');

      expect(result).toEqual([]);
    });

    it('returns empty array on error', async () => {
      mockFetch.mockReturnValue(throwError(() => new Error('Network error')));

      const result = await MCPService.getServerTools('server-1');

      expect(result).toEqual([]);
    });
  });

  // ====================
  // getToolsWithServerInfo
  // ====================
  describe('getToolsWithServerInfo', () => {
    it('returns tools with serverId attached', async () => {
      const mockTools: MCPTool[] = [{ name: 'tool1' }, { name: 'tool2' }];
      const mockServers: MCPServerConfig[] = [
        { id: 'server-1', name: 'Server 1', url: 'http://s1.local', type: 'remote', enabled: true, tools: mockTools },
        { id: 'server-2', name: 'Server 2', url: 'http://s2.local', type: 'remote', enabled: false, tools: [{ name: 'tool3' }] },
      ];
      mockFetch.mockReturnValue(of({ status: 200, data: { servers: mockServers } }));

      const result = await MCPService.getToolsWithServerInfo();

      // Only returns tools from enabled servers
      expect(result).toHaveLength(2);
      expect(result[0].serverId).toBe('server-1');
      expect(result[1].serverId).toBe('server-1');
    });

    it('returns empty array when no enabled servers', async () => {
      const mockServers: MCPServerConfig[] = [
        { id: 'server-1', name: 'Server 1', url: 'http://s1.local', type: 'remote', enabled: false, tools: [{ name: 'tool1' }] },
      ];
      mockFetch.mockReturnValue(of({ status: 200, data: { servers: mockServers } }));

      const result = await MCPService.getToolsWithServerInfo();

      expect(result).toEqual([]);
    });

    it('returns empty array on error', async () => {
      mockFetch.mockReturnValue(throwError(() => new Error('Network error')));

      const result = await MCPService.getToolsWithServerInfo();

      expect(result).toEqual([]);
    });
  });

  // ====================
  // getAllTools
  // ====================
  describe('getAllTools', () => {
    it('returns all tools from backend', async () => {
      const mockTools: MCPTool[] = [
        { name: 'tool1', description: 'Tool 1' },
        { name: 'tool2', description: 'Tool 2' },
      ];
      mockFetch.mockReturnValue(of({ status: 200, data: { tools: mockTools } }));

      const result = await MCPService.getAllTools();

      expect(result).toHaveLength(2);
      expect(mockFetch).toHaveBeenCalledWith(
        expect.objectContaining({
          url: '/api/plugins/cisco-mcpclient-app/resources/tools',
          method: 'GET',
        })
      );
    });

    it('returns empty array when no tools', async () => {
      mockFetch.mockReturnValue(of({ status: 200, data: { tools: [] } }));

      const result = await MCPService.getAllTools();

      expect(result).toEqual([]);
    });

    it('returns empty array on error', async () => {
      mockFetch.mockReturnValue(throwError(() => new Error('Network error')));

      const result = await MCPService.getAllTools();

      expect(result).toEqual([]);
    });
  });

  // ====================
  // callTool
  // ====================
  describe('callTool', () => {
    it('calls tool successfully', async () => {
      mockFetch.mockReturnValue(of({
        status: 200,
        data: { success: true, content: 'Tool executed successfully' },
      }));

      const result = await MCPService.callTool('testTool', { param: 'value' });

      expect(result).toEqual({
        success: true,
        content: 'Tool executed successfully',
        error: undefined,
      });
      expect(mockFetch).toHaveBeenCalledWith(expect.objectContaining({
        url: '/api/plugins/cisco-mcpclient-app/resources/tools/call',
        method: 'POST',
        data: {
          tool_name: 'testTool',
          arguments: { param: 'value' },
          server_id: undefined,
        },
      }));
    });

    it('calls tool with server ID', async () => {
      mockFetch.mockReturnValue(of({
        status: 200,
        data: { success: true, content: 'Result from specific server' },
      }));

      const result = await MCPService.callTool('testTool', { param: 'value' }, 'server-123');

      expect(result.success).toBe(true);
      expect(mockFetch).toHaveBeenCalledWith(expect.objectContaining({
        data: expect.objectContaining({
          server_id: 'server-123',
        }),
      }));
    });

    it('returns error on tool failure', async () => {
      mockFetch.mockReturnValue(of({
        status: 200,
        data: { success: false, content: '', error: 'Tool not found' },
      }));

      const result = await MCPService.callTool('unknownTool', {});

      expect(result.success).toBe(false);
      expect(result.error).toBe('Tool not found');
    });

    it('handles network error gracefully', async () => {
      mockFetch.mockReturnValue(throwError(() => new Error('Network error')));

      const result = await MCPService.callTool('testTool', {});

      expect(result.success).toBe(false);
      expect(result.error).toBe('Network error');
    });
  });

  // ====================
  // getConfig
  // ====================
  describe('getConfig', () => {
    it('returns config from backend', async () => {
      mockFetch.mockReturnValue(of({
        status: 200,
        data: {
          autoDiscovery: true,
          connectionTimeout: 30,
          retryAttempts: 3,
          enableLogging: true,
        },
      }));

      const result = await MCPService.getConfig();

      expect(result).toEqual({
        autoDiscovery: true,
        connectionTimeout: 30,
        retryAttempts: 3,
        enableLogging: true,
      });
      expect(mockFetch).toHaveBeenCalledWith(expect.objectContaining({
        url: '/api/plugins/cisco-mcpclient-app/resources/config',
        method: 'GET',
      }));
    });

    it('returns empty object on error', async () => {
      mockFetch.mockReturnValue(throwError(() => new Error('Network error')));

      const result = await MCPService.getConfig();

      expect(result).toEqual({});
    });
  });

  // ====================
  // setConfig
  // ====================
  describe('setConfig', () => {
    it('saves config to backend', async () => {
      const newConfig = { autoDiscovery: false, connectionTimeout: 60 };
      mockFetch.mockReturnValue(of({ status: 200, data: newConfig }));

      const result = await MCPService.setConfig(newConfig);

      expect(result).toEqual(newConfig);
      expect(mockFetch).toHaveBeenCalledWith(expect.objectContaining({
        url: '/api/plugins/cisco-mcpclient-app/resources/config',
        method: 'POST',
        data: newConfig,
      }));
    });

    it('throws on error', async () => {
      mockFetch.mockReturnValue(throwError(() => new Error('Save failed')));

      await expect(MCPService.setConfig({ setting: 'value' })).rejects.toThrow('Save failed');
    });
  });
});
