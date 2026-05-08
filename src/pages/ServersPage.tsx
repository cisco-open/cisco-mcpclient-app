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
import { PluginPage } from '@grafana/runtime';
import { Card, Button, Badge, useStyles2, Alert } from '@grafana/ui';
import { css } from '@emotion/css';
import { GrafanaTheme2 } from '@grafana/data';
import { useNavigate } from 'react-router-dom';
import { ROUTES } from '../constants';
import MCPService from '../services/MCPService';
import { MCPServerConfig, MCPServerStatus } from '../types/mcp';

function ServersPage() {
  const styles = useStyles2(getStyles);
  const navigate = useNavigate();
  const [servers, setServers] = useState<MCPServerConfig[]>([]);
  const [statuses, setStatuses] = useState<MCPServerStatus[]>([]);
  const [serviceAvailable, setServiceAvailable] = useState<boolean | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    console.log('loadData() called');
    try {
      setLoading(true);

      // Check service availability
      console.log('Calling MCPService.isServiceAvailable()');
      const available = await MCPService.isServiceAvailable();
      console.log('Service available result:', available);
      setServiceAvailable(available);

      if (available) {
        // Load server configurations and statuses
        console.log('Loading server configs and statuses...');
        const serverConfigs = await MCPService.getServerConfigs();
        const serverStatuses = await MCPService.getServerStatuses();
        console.log('Server configs:', serverConfigs);
        console.log('Server statuses:', serverStatuses);
        setServers(serverConfigs);
        setStatuses(serverStatuses);
      }
    } catch (error) {
      console.error('Failed to load MCP server data:', error);
      setServiceAvailable(false);
    } finally {
      setLoading(false);
    }
  };

  const addDefaultServer = async () => {
    console.log('addDefaultServer() called');
    const defaultServer: MCPServerConfig = {
      id: 'local-grafana-mcp',
      name: 'Local Grafana MCP Server',
      url: 'http://localhost:3031',
      type: 'local',
      enabled: true,
      authType: 'none',
      description: 'Default local MCP server for development'
    };

    try {
      console.log('Calling MCPService.addServer with:', defaultServer);
      await MCPService.addServer(defaultServer);
      console.log('addServer completed, calling loadData...');
      await loadData(); // Refresh data
    } catch (error) {
      console.error('Error adding default server:', error);
    }
  };

  const handleAddServer = () => {
    console.log('Header Add Server button clicked');
    // Navigate to config page for new server
    navigate(`${ROUTES.Config}?new=true`);
  };

  const handleConfigure = (serverId: string) => {
    console.log('Configure button clicked for server:', serverId);
    // Navigate to config page for existing server
    navigate(`${ROUTES.Config}?id=${serverId}`);
  };

  const handleTestConnection = async (serverId: string) => {
    console.log('Testing connection for server:', serverId);
    const server = servers.find(s => s.id === serverId);
    console.log('Found server:', server);
    if (!server) {
      return;
    }

    try {
      const result = await MCPService.testServerConnection(serverId);
      console.log('Test connection result:', result);
      await loadData(); // Refresh statuses
    } catch (error) {
      console.error('Error testing connection:', error);
    }
  };

  const handleRemoveServer = async (serverId: string) => {
    try {
      await MCPService.removeServer(serverId);
      await loadData(); // Refresh data
    } catch (error) {
      console.error('Error removing server:', error);
    }
  };

  const getServerStatus = (serverId: string): MCPServerStatus | undefined => {
    return statuses.find(s => s.id === serverId);
  };

  const getStatusColor = (status?: MCPServerStatus): 'green' | 'orange' | 'red' | 'blue' => {
    if (!status) {
      return 'blue';
    }
    if (status.connected) {
      return 'green';
    }
    if (status.error) {
      return 'red';
    }
    return 'orange';
  };

  const getStatusText = (status?: MCPServerStatus): string => {
    if (!status) {
      return 'unknown';
    }
    if (status.connected) {
      return 'connected';
    }
    if (status.error) {
      return 'error';
    }
    return 'disconnected';
  };

  if (loading) {
    return (
      <PluginPage>
        <div className={styles.container}>
          <div className={styles.header}>
            <h2>MCP Servers</h2>
          </div>
          <div>Loading...</div>
        </div>
      </PluginPage>
    );
  }

  if (serviceAvailable === false) {
    return (
      <PluginPage>
        <div className={styles.container}>
          <Alert
            title="MCP Service Not Available"
            severity="warning"
          >
            The MCP/LLM service is not configured or enabled. Please ensure the LLM app is properly configured
            with an MCP provider in Administration → Plugins → LLM App.
          </Alert>
        </div>
      </PluginPage>
    );
  }

  return (
    <PluginPage>
      <div className={styles.container}>
        <div className={styles.header}>
          <h2>MCP Servers</h2>
          <Button variant="primary" icon="plus" onClick={handleAddServer}>
            Add Server
          </Button>
        </div>

        <div className={styles.serverList}>
          {servers.map((server) => {
            const serverId = server.id || server.name;
            const status = getServerStatus(serverId);
            return (
              <Card key={serverId} className={styles.serverCard}>
                <Card.Heading>
                  <div className={styles.serverHeader}>
                    <span>{server.name}</span>
                    <Badge
                      text={getStatusText(status)}
                      color={getStatusColor(status)}
                    />
                  </div>
                </Card.Heading>
                <Card.Meta className={styles.serverMeta}>
                  <div>URL: {server.url}</div>
                  <div>Type: {server.type}</div>
                  <div className={styles.toolsSection}>
                    <div>Available Tools ({status?.tools?.length || 0})</div>
                  </div>
                  {status?.error && <div className={styles.errorText}>Error: {status.error}</div>}
                  {status?.lastChecked && (
                    <div>Last Checked: {status.lastChecked.toLocaleTimeString()}</div>
                  )}
                </Card.Meta>
                <Card.Actions>
                  <Button variant="secondary" size="sm" onClick={() => handleConfigure(serverId)}>
                    Configure
                  </Button>
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={() => handleTestConnection(serverId)}
                  >
                    Test Connection
                  </Button>
                  <Button
                    variant="destructive"
                    size="sm"
                    fill="text"
                    onClick={() => handleRemoveServer(serverId)}
                  >
                    Remove
                  </Button>
                </Card.Actions>
              </Card>
            );
          })}
        </div>

        {servers.length === 0 && (
          <div className={styles.emptyState}>
            <h3>No MCP servers configured</h3>
            <p>Add your first Model Context Protocol server to get started.</p>
            <Button variant="primary" icon="plus" onClick={addDefaultServer}>
              Add Default Local Server
            </Button>
          </div>
        )}
      </div>
    </PluginPage>
  );
}

const getStyles = (theme: GrafanaTheme2) => ({
  container: css`
    padding: ${theme.spacing(2)};
  `,
  header: css`
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: ${theme.spacing(3)};
  `,
  serverList: css`
    display: grid;
    gap: ${theme.spacing(2)};
    grid-template-columns: repeat(auto-fill, minmax(400px, 1fr));
  `,
  serverCard: css`
    padding: ${theme.spacing(2)};
  `,
  serverHeader: css`
    display: flex;
    justify-content: space-between;
    align-items: center;
    width: 100%;
  `,
  serverMeta: css`
    color: ${theme.colors.text.secondary};
    font-size: ${theme.typography.bodySmall.fontSize};

    > div {
      margin-bottom: ${theme.spacing(0.5)};
    }
  `,
  errorText: css`
    color: ${theme.colors.error.text};
    font-weight: ${theme.typography.fontWeightMedium};
  `,
  toolsSection: css`
    margin: ${theme.spacing(1)} 0;
    padding: ${theme.spacing(1)};
    background: ${theme.colors.background.secondary};
    border-radius: ${theme.shape.borderRadius()};
    border-left: 3px solid ${theme.colors.primary.main};
  `,
  toolsList: css`
    max-height: 150px;
    overflow-y: auto;
    margin-top: ${theme.spacing(1)};
    line-height: 1.4;
  `,
  toolName: css`
    font-weight: ${theme.typography.fontWeightMedium};
    color: ${theme.colors.text.primary};
    font-size: ${theme.typography.bodySmall.fontSize};
  `,
  noTools: css`
    color: ${theme.colors.text.disabled};
    font-style: italic;
    margin-top: ${theme.spacing(0.5)};
  `,
  emptyState: css`
    text-align: center;
    padding: ${theme.spacing(4)};
    color: ${theme.colors.text.secondary};

    h3 {
      margin-bottom: ${theme.spacing(1)};
      color: ${theme.colors.text.primary};
    }

    p {
      margin-bottom: ${theme.spacing(2)};
    }
  `,
});

export default ServersPage;
