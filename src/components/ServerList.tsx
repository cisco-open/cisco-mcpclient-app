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

import React, { useState, useEffect } from 'react';
import {
  Button,
  Card,
  ConfirmModal,
  Field,
  Icon,
  IconButton,
  LoadingPlaceholder,
  Modal,
  Switch,
  useStyles2,
  Stack,
} from '@grafana/ui';
import { AppEvents, GrafanaTheme2 } from '@grafana/data';
import { getAppEvents } from '@grafana/runtime';
import { css } from '@emotion/css';
import { MCPServerConfig } from '../types/mcp';
import { MCPService } from '../services/MCPService';
import { ServerConfigForm } from './ServerConfigForm';

interface ServerListProps {
  onServerSelect?: (config: MCPServerConfig) => void;
}

export const ServerList: React.FC<ServerListProps> = ({ onServerSelect }) => {
  const styles = useStyles2(getStyles);
  const [servers, setServers] = useState<MCPServerConfig[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showAddModal, setShowAddModal] = useState(false);
  const [editingServer, setEditingServer] = useState<MCPServerConfig | null>(null);
  const [deletingServer, setDeletingServer] = useState<MCPServerConfig | null>(null);

  const showSuccess = (message: string) => {
    getAppEvents().publish({
      type: AppEvents.alertSuccess.name,
      payload: [message],
    });
  };

  const showError = (title: string, error: unknown) => {
    const message = error instanceof Error ? error.message : 'An unexpected error occurred';
    getAppEvents().publish({
      type: AppEvents.alertError.name,
      payload: [title, message],
    });
  };

  useEffect(() => {
    loadServers();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const loadServers = async () => {
    setLoading(true);
    setError(null);
    try {
      const servers = await MCPService.getServerConfigs();
      setServers(servers);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error';
      setError('Failed to load servers: ' + message);
      showError('Failed to load servers', err);
    } finally {
      setLoading(false);
    }
  };

  const handleAddServer = async (config: MCPServerConfig) => {
    try {
      setError(null);
      await MCPService.addServer(config);
      showSuccess(`Server "${config.name}" added successfully`);
      setShowAddModal(false);
      await loadServers();
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error';
      setError('Failed to add server: ' + message);
      showError('Failed to add server', err);
    }
  };

  const handleEditServer = async (config: MCPServerConfig) => {
    try {
      setError(null);
      await MCPService.updateServer(config);
      showSuccess(`Server "${config.name}" updated successfully`);
      setEditingServer(null);
      await loadServers();
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error';
      setError('Failed to update server: ' + message);
      showError('Failed to update server', err);
    }
  };

  const handleDeleteServer = async () => {
    if (!deletingServer) {
      return;
    }

    try {
      setError(null);
      const serverId = deletingServer.id || deletingServer.name;
      await MCPService.removeServer(serverId);
      showSuccess(`Server "${deletingServer.name}" deleted`);
      setDeletingServer(null);
      await loadServers();
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error';
      setError('Failed to delete server: ' + message);
      showError('Failed to delete server', err);
    }
  };

  const handleToggleEnabled = async (server: MCPServerConfig) => {
    try {
      setError(null);
      const updatedServer = { ...server, enabled: !server.enabled };
      await MCPService.updateServer(updatedServer);
      showSuccess(`Server "${server.name}" ${updatedServer.enabled ? 'enabled' : 'disabled'}`);
      await loadServers();
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error';
      setError('Failed to toggle server: ' + message);
      showError('Failed to toggle server', err);
    }
  };

  const handleTestConnection = async (server: MCPServerConfig) => {
    try {
      setError(null);
      const connected = await MCPService.testServerConnection(server.name);
      if (connected) {
        showSuccess(`Connection to "${server.name}" successful`);
        await loadServers();
      } else {
        throw new Error('Connection test failed - server unreachable');
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error';
      setError('Connection test failed: ' + message);
      showError('Connection test failed', err);
      await loadServers();
    }
  };

  const getStatusIcon = (status = 'disconnected') => {
    switch (status) {
      case 'connected':
        return <Icon name="check-circle" className={styles.statusConnected} />;
      case 'error':
        return <Icon name="exclamation-circle" className={styles.statusError} />;
      default:
        return <Icon name="circle" className={styles.statusDisconnected} />;
    }
  };

  if (loading) {
    return <LoadingPlaceholder text="Loading MCP servers..." />;
  }

  return (
    <div className={styles.container}>
      <Stack direction="column" gap={4}>
        <Stack direction="row" justifyContent="space-between">
          <h3>MCP Servers</h3>
          <Button onClick={() => setShowAddModal(true)} icon="plus">
            Add Server
          </Button>
        </Stack>

        {error && (
          <div className={styles.error}>
            <Icon name="exclamation-triangle" />
            {error}
          </div>
        )}

        {servers.length === 0 ? (
          <Card className={styles.emptyState}>
            <Card.Heading>No MCP servers configured</Card.Heading>
            <Card.Description>
              Add your first MCP server to get started. You can connect to local servers
              like the Time MCP Server or remote servers with authentication.
            </Card.Description>
            <Card.Actions>
              <Button onClick={() => setShowAddModal(true)} variant="primary" icon="plus">
                Add Your First Server
              </Button>
            </Card.Actions>
          </Card>
        ) : (
          <div className={styles.serverGrid}>
            {servers.map((server) => (
              <Card key={server.id} className={styles.serverCard}>
                <Card.Heading className={styles.serverHeader}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                      {getStatusIcon(server.status)}
                      <span>{server.name}</span>
                      {server.type === 'local' && (
                        <span className={styles.typeLabel}>Local</span>
                      )}
                    </div>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
                      <Field label="Enabled" className={styles.enabledSwitch}>
                        <Switch
                          value={server.enabled}
                          onChange={() => handleToggleEnabled(server)}
                        />
                      </Field>
                      <IconButton
                        name="cog"
                        tooltip="Configure"
                        onClick={() => setEditingServer(server)}
                      />
                      <IconButton
                        name="trash-alt"
                        tooltip="Delete"
                        onClick={() => setDeletingServer(server)}
                      />
                    </div>
                  </div>
                </Card.Heading>

                <Card.Description>
                  <div>
                    <div><strong>URL:</strong> {server.url}</div>
                    {server.description && (
                      <div><strong>Description:</strong> {server.description}</div>
                    )}
                    {server.capabilities && server.capabilities.length > 0 && (
                      <div>
                        <strong>Capabilities:</strong> {server.capabilities.join(', ')}
                      </div>
                    )}
                    {server.auth && (
                      <div><strong>Authentication:</strong> {server.auth.type}</div>
                    )}
                  </div>
                </Card.Description>

                <Card.Actions>
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={() => handleTestConnection(server)}
                  >
                    Test Connection
                  </Button>
                  {onServerSelect && (
                    <Button
                      variant="primary"
                      size="sm"
                      onClick={() => onServerSelect(server)}
                    >
                      Select
                    </Button>
                  )}
                </Card.Actions>
              </Card>
            ))}
          </div>
        )}
      </Stack>

      {/* Add Server Modal */}
      {showAddModal && (
        <Modal
          title="Add MCP Server"
          isOpen={showAddModal}
          onDismiss={() => setShowAddModal(false)}
          className={styles.modal}
        >
          <ServerConfigForm
            onSave={handleAddServer}
            onCancel={() => setShowAddModal(false)}
          />
        </Modal>
      )}

      {/* Edit Server Modal */}
      {editingServer && (
        <Modal
          title="Edit MCP Server"
          isOpen={!!editingServer}
          onDismiss={() => setEditingServer(null)}
          className={styles.modal}
        >
          <ServerConfigForm
            initialConfig={editingServer}
            onSave={handleEditServer}
            onCancel={() => setEditingServer(null)}
          />
        </Modal>
      )}

      {/* Delete Confirmation Modal */}
      {deletingServer && (
        <ConfirmModal
          isOpen={!!deletingServer}
          title="Delete MCP Server"
          body={`Are you sure you want to delete "${deletingServer.name}"? This action cannot be undone.`}
          confirmText="Delete"
          onConfirm={handleDeleteServer}
          onDismiss={() => setDeletingServer(null)}
        />
      )}
    </div>
  );
};

const getStyles = (theme: GrafanaTheme2) => ({
  container: css({
    padding: theme.spacing(2),
  }),

  error: css({
    color: theme.colors.error.text,
    backgroundColor: theme.colors.error.transparent,
    padding: theme.spacing(1, 2),
    borderRadius: theme.shape.radius.default,
    display: 'flex',
    alignItems: 'center',
    gap: theme.spacing(1),
  }),

  emptyState: css({
    textAlign: 'center',
    padding: theme.spacing(4),
  }),

  serverGrid: css({
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fill, minmax(400px, 1fr))',
    gap: theme.spacing(2),
  }),

  serverCard: css({
    cursor: 'pointer',
    transition: 'transform 0.2s ease-in-out',
    '&:hover': {
      transform: 'translateY(-2px)',
    },
  }),

  serverHeader: css({
    margin: 0,
  }),

  enabledSwitch: css({
    margin: 0,
    '> label': {
      display: 'none',
    },
  }),

  typeLabel: css({
    backgroundColor: theme.colors.primary.transparent,
    color: theme.colors.primary.text,
    padding: theme.spacing(0.25, 0.75),
    borderRadius: theme.shape.radius.pill,
    fontSize: theme.typography.bodySmall.fontSize,
    fontWeight: theme.typography.fontWeightMedium,
  }),

  statusConnected: css({
    color: theme.colors.success.text,
  }),

  statusError: css({
    color: theme.colors.error.text,
  }),

  statusDisconnected: css({
    color: theme.colors.text.secondary,
  }),

  modal: css({
    width: '600px',
    maxWidth: '90vw',
  }),
});
