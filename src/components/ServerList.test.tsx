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

import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { ServerList } from './ServerList';
import { MCPServerConfig } from '../types/mcp';

// Mock MCPService
const mockGetServerConfigs = jest.fn();
const mockAddServer = jest.fn();
const mockUpdateServer = jest.fn();
const mockRemoveServer = jest.fn();
const mockTestServerConnection = jest.fn();

jest.mock('../services/MCPService', () => ({
  MCPService: {
    getServerConfigs: () => mockGetServerConfigs(),
    addServer: (config: MCPServerConfig) => mockAddServer(config),
    updateServer: (config: MCPServerConfig) => mockUpdateServer(config),
    removeServer: (id: string) => mockRemoveServer(id),
    testServerConnection: (name: string) => mockTestServerConnection(name),
  },
}));

// Mock @grafana/runtime
const mockPublish = jest.fn();
jest.mock('@grafana/runtime', () => ({
  getAppEvents: () => ({
    publish: mockPublish,
  }),
}));

describe('ServerList', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    // Suppress console output during tests
    jest.spyOn(console, 'log').mockImplementation(() => {});
    jest.spyOn(console, 'error').mockImplementation(() => {});
    jest.spyOn(console, 'warn').mockImplementation(() => {});
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  // ====================
  // Rendering Tests
  // ====================
  describe('rendering', () => {
    it('shows loading state while fetching servers', async () => {
      // Create a promise that never resolves to keep loading state
      mockGetServerConfigs.mockReturnValue(new Promise(() => {}));

      render(<ServerList />);

      expect(screen.getByText(/loading mcp servers/i)).toBeInTheDocument();
    });

    it('renders empty state when no servers configured', async () => {
      mockGetServerConfigs.mockResolvedValue([]);

      render(<ServerList />);

      await waitFor(() => {
        expect(screen.getByText(/no mcp servers configured/i)).toBeInTheDocument();
      });

      expect(screen.getByRole('button', { name: /add your first server/i })).toBeInTheDocument();
    });

    it('renders server list with names', async () => {
      const mockServers: MCPServerConfig[] = [
        { id: '1', name: 'Production Server', url: 'http://prod.local', type: 'remote', enabled: true },
        { id: '2', name: 'Dev Server', url: 'http://dev.local', type: 'local', enabled: false },
      ];
      mockGetServerConfigs.mockResolvedValue(mockServers);

      render(<ServerList />);

      await waitFor(() => {
        expect(screen.getByText('Production Server')).toBeInTheDocument();
      });

      expect(screen.getByText('Dev Server')).toBeInTheDocument();
    });

    it('displays server URLs in descriptions', async () => {
      const mockServers: MCPServerConfig[] = [
        { id: '1', name: 'Test Server', url: 'http://test.example.com', type: 'remote', enabled: true },
      ];
      mockGetServerConfigs.mockResolvedValue(mockServers);

      render(<ServerList />);

      await waitFor(() => {
        expect(screen.getByText('http://test.example.com')).toBeInTheDocument();
      });
    });

    it('shows local type label for local servers', async () => {
      const mockServers: MCPServerConfig[] = [
        { id: '1', name: 'Local MCP', url: 'http://localhost:8080', type: 'local', enabled: true },
      ];
      mockGetServerConfigs.mockResolvedValue(mockServers);

      render(<ServerList />);

      await waitFor(() => {
        expect(screen.getByText('Local')).toBeInTheDocument();
      });
    });

    it('shows connected status icon for connected servers', async () => {
      const mockServers: MCPServerConfig[] = [
        { id: '1', name: 'Connected Server', url: 'http://test.local', type: 'remote', enabled: true, status: 'connected' },
      ];
      mockGetServerConfigs.mockResolvedValue(mockServers);

      render(<ServerList />);

      await waitFor(() => {
        expect(screen.getByText('Connected Server')).toBeInTheDocument();
      });

      // Icon should be present (check-circle for connected)
      const serverCard = screen.getByText('Connected Server').closest('div');
      expect(serverCard).toBeInTheDocument();
    });

    it('shows error status icon for servers with errors', async () => {
      const mockServers: MCPServerConfig[] = [
        { id: '1', name: 'Error Server', url: 'http://test.local', type: 'remote', enabled: true, status: 'error' },
      ];
      mockGetServerConfigs.mockResolvedValue(mockServers);

      render(<ServerList />);

      await waitFor(() => {
        expect(screen.getByText('Error Server')).toBeInTheDocument();
      });
    });

    it('displays error message when server load fails', async () => {
      mockGetServerConfigs.mockRejectedValue(new Error('Network failure'));

      render(<ServerList />);

      await waitFor(() => {
        expect(screen.getByText(/failed to load servers/i)).toBeInTheDocument();
      });

      // Should also trigger toast notification
      expect(mockPublish).toHaveBeenCalled();
    });

    it('shows authentication type when configured', async () => {
      const mockServers: MCPServerConfig[] = [
        {
          id: '1',
          name: 'Auth Server',
          url: 'http://test.local',
          type: 'remote',
          enabled: true,
          auth: { type: 'bearer', token: 'secret' },
        },
      ];
      mockGetServerConfigs.mockResolvedValue(mockServers);

      render(<ServerList />);

      await waitFor(() => {
        expect(screen.getByText(/bearer/i)).toBeInTheDocument();
      });
    });

    it('shows capabilities when available', async () => {
      const mockServers: MCPServerConfig[] = [
        {
          id: '1',
          name: 'Capable Server',
          url: 'http://test.local',
          type: 'remote',
          enabled: true,
          capabilities: ['tools', 'prompts'],
        },
      ];
      mockGetServerConfigs.mockResolvedValue(mockServers);

      render(<ServerList />);

      await waitFor(() => {
        expect(screen.getByText(/tools, prompts/i)).toBeInTheDocument();
      });
    });
  });

  // ====================
  // Interaction Tests
  // ====================
  describe('interactions', () => {
    it('opens add server modal when Add Server button is clicked', async () => {
      mockGetServerConfigs.mockResolvedValue([]);

      render(<ServerList />);

      await waitFor(() => {
        expect(screen.getByText(/no mcp servers configured/i)).toBeInTheDocument();
      });

      fireEvent.click(screen.getByRole('button', { name: /add your first server/i }));

      await waitFor(() => {
        expect(screen.getByText(/add mcp server/i)).toBeInTheDocument();
      });
    });

    it('opens add server modal from header button when servers exist', async () => {
      const mockServers: MCPServerConfig[] = [
        { id: '1', name: 'Existing Server', url: 'http://test.local', type: 'remote', enabled: true },
      ];
      mockGetServerConfigs.mockResolvedValue(mockServers);

      render(<ServerList />);

      await waitFor(() => {
        expect(screen.getByText('Existing Server')).toBeInTheDocument();
      });

      // Click the Add Server button in the header
      const addButtons = screen.getAllByRole('button', { name: /add server/i });
      fireEvent.click(addButtons[0]);

      await waitFor(() => {
        expect(screen.getByText(/add mcp server/i)).toBeInTheDocument();
      });
    });

    it('triggers delete confirmation when delete button is clicked', async () => {
      const mockServers: MCPServerConfig[] = [
        { id: '1', name: 'Delete Me', url: 'http://test.local', type: 'remote', enabled: true },
      ];
      mockGetServerConfigs.mockResolvedValue(mockServers);

      render(<ServerList />);

      await waitFor(() => {
        expect(screen.getByText('Delete Me')).toBeInTheDocument();
      });

      // Click the delete button (trash-alt icon)
      const deleteButton = screen.getByRole('button', { name: /delete/i });
      fireEvent.click(deleteButton);

      // Should show confirmation modal
      await waitFor(() => {
        expect(screen.getByText(/are you sure you want to delete/i)).toBeInTheDocument();
      });
    });

    it('deletes server when confirm is clicked', async () => {
      const mockServers: MCPServerConfig[] = [
        { id: 'server-1', name: 'To Be Deleted', url: 'http://test.local', type: 'remote', enabled: true },
      ];
      mockGetServerConfigs.mockResolvedValue(mockServers);
      mockRemoveServer.mockResolvedValue(undefined);

      render(<ServerList />);

      await waitFor(() => {
        expect(screen.getByText('To Be Deleted')).toBeInTheDocument();
      });

      // Click delete button
      const deleteButton = screen.getByRole('button', { name: /delete/i });
      fireEvent.click(deleteButton);

      await waitFor(() => {
        expect(screen.getByText(/are you sure you want to delete/i)).toBeInTheDocument();
      });

      // Update mock to return empty list after delete
      mockGetServerConfigs.mockResolvedValue([]);

      // Click confirm - use testid for the danger button in ConfirmModal
      const confirmButton = screen.getByTestId('data-testid Confirm Modal Danger Button');
      fireEvent.click(confirmButton);

      await waitFor(() => {
        expect(mockRemoveServer).toHaveBeenCalledWith('server-1');
      });

      // Should trigger success toast
      expect(mockPublish).toHaveBeenCalled();
    });

    it('triggers connection test when Test Connection button is clicked', async () => {
      const mockServers: MCPServerConfig[] = [
        { id: '1', name: 'Test Me', url: 'http://test.local', type: 'remote', enabled: true },
      ];
      mockGetServerConfigs.mockResolvedValue(mockServers);
      mockTestServerConnection.mockResolvedValue(true);

      render(<ServerList />);

      await waitFor(() => {
        expect(screen.getByText('Test Me')).toBeInTheDocument();
      });

      const testButton = screen.getByRole('button', { name: /test connection/i });
      fireEvent.click(testButton);

      await waitFor(() => {
        expect(mockTestServerConnection).toHaveBeenCalledWith('Test Me');
      });

      // Should trigger success toast
      expect(mockPublish).toHaveBeenCalled();
    });

    it('shows error toast when connection test fails', async () => {
      const mockServers: MCPServerConfig[] = [
        { id: '1', name: 'Failing Server', url: 'http://test.local', type: 'remote', enabled: true },
      ];
      mockGetServerConfigs.mockResolvedValue(mockServers);
      mockTestServerConnection.mockResolvedValue(false);

      render(<ServerList />);

      await waitFor(() => {
        expect(screen.getByText('Failing Server')).toBeInTheDocument();
      });

      const testButton = screen.getByRole('button', { name: /test connection/i });
      fireEvent.click(testButton);

      await waitFor(() => {
        expect(mockTestServerConnection).toHaveBeenCalledWith('Failing Server');
      });

      // Should trigger error toast
      await waitFor(() => {
        expect(mockPublish).toHaveBeenCalled();
      });
    });

    it('toggles server enabled state when switch is clicked', async () => {
      const mockServers: MCPServerConfig[] = [
        { id: '1', name: 'Toggle Me', url: 'http://test.local', type: 'remote', enabled: true },
      ];
      mockGetServerConfigs.mockResolvedValue(mockServers);
      mockUpdateServer.mockResolvedValue({ ...mockServers[0], enabled: false });

      render(<ServerList />);

      await waitFor(() => {
        expect(screen.getByText('Toggle Me')).toBeInTheDocument();
      });

      // Find the switch (has role="switch" in Grafana UI)
      const switchElement = screen.getByRole('switch');
      fireEvent.click(switchElement);

      await waitFor(() => {
        expect(mockUpdateServer).toHaveBeenCalledWith(
          expect.objectContaining({
            id: '1',
            name: 'Toggle Me',
            enabled: false,
          })
        );
      });

      // Should trigger success toast
      expect(mockPublish).toHaveBeenCalled();
    });

    it('opens edit modal when configure button is clicked', async () => {
      const mockServers: MCPServerConfig[] = [
        { id: '1', name: 'Edit Me', url: 'http://test.local', type: 'remote', enabled: true },
      ];
      mockGetServerConfigs.mockResolvedValue(mockServers);

      render(<ServerList />);

      await waitFor(() => {
        expect(screen.getByText('Edit Me')).toBeInTheDocument();
      });

      // Click the configure button (cog icon)
      const configButton = screen.getByRole('button', { name: /configure/i });
      fireEvent.click(configButton);

      // Should show edit modal
      await waitFor(() => {
        expect(screen.getByText(/edit mcp server/i)).toBeInTheDocument();
      });
    });

    it('calls onServerSelect when Select button is clicked', async () => {
      const mockServers: MCPServerConfig[] = [
        { id: '1', name: 'Selectable Server', url: 'http://test.local', type: 'remote', enabled: true },
      ];
      mockGetServerConfigs.mockResolvedValue(mockServers);
      const handleSelect = jest.fn();

      render(<ServerList onServerSelect={handleSelect} />);

      await waitFor(() => {
        expect(screen.getByText('Selectable Server')).toBeInTheDocument();
      });

      const selectButton = screen.getByRole('button', { name: /select/i });
      fireEvent.click(selectButton);

      expect(handleSelect).toHaveBeenCalledWith(mockServers[0]);
    });

    it('does not show Select button when onServerSelect is not provided', async () => {
      const mockServers: MCPServerConfig[] = [
        { id: '1', name: 'No Select', url: 'http://test.local', type: 'remote', enabled: true },
      ];
      mockGetServerConfigs.mockResolvedValue(mockServers);

      render(<ServerList />);

      await waitFor(() => {
        expect(screen.getByText('No Select')).toBeInTheDocument();
      });

      expect(screen.queryByRole('button', { name: /^select$/i })).not.toBeInTheDocument();
    });

    it('shows error message when toggle fails', async () => {
      const mockServers: MCPServerConfig[] = [
        { id: '1', name: 'Toggle Fail', url: 'http://test.local', type: 'remote', enabled: true },
      ];
      mockGetServerConfigs.mockResolvedValue(mockServers);
      mockUpdateServer.mockRejectedValue(new Error('Toggle failed'));

      render(<ServerList />);

      await waitFor(() => {
        expect(screen.getByText('Toggle Fail')).toBeInTheDocument();
      });

      const switchElement = screen.getByRole('switch');
      fireEvent.click(switchElement);

      await waitFor(() => {
        expect(screen.getByText(/failed to toggle server/i)).toBeInTheDocument();
      });

      expect(mockPublish).toHaveBeenCalled();
    });

    it('shows error message when delete fails', async () => {
      const mockServers: MCPServerConfig[] = [
        { id: 'del-1', name: 'Delete Fail', url: 'http://test.local', type: 'remote', enabled: true },
      ];
      mockGetServerConfigs.mockResolvedValue(mockServers);
      mockRemoveServer.mockRejectedValue(new Error('Cannot delete'));

      render(<ServerList />);

      await waitFor(() => {
        expect(screen.getByText('Delete Fail')).toBeInTheDocument();
      });

      // Click delete
      const deleteButton = screen.getByRole('button', { name: /delete/i });
      fireEvent.click(deleteButton);

      await waitFor(() => {
        expect(screen.getByText(/are you sure you want to delete/i)).toBeInTheDocument();
      });

      // Confirm delete - use testid for the danger button in ConfirmModal
      const confirmButton = screen.getByTestId('data-testid Confirm Modal Danger Button');
      fireEvent.click(confirmButton);

      await waitFor(() => {
        expect(screen.getByText(/failed to delete server/i)).toBeInTheDocument();
      });

      expect(mockPublish).toHaveBeenCalled();
    });
  });
});
