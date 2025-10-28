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
import { useStyles2, Alert, Spinner, Select } from '@grafana/ui';
import { css } from '@emotion/css';
import { GrafanaTheme2, SelectableValue } from '@grafana/data';
import MCPService from '../services/MCPService';
import { MCPTool, MCPServerConfig, MCPServerStatus } from '../types/mcp';

interface ToolWithServer extends MCPTool {
  serverName?: string;
  available: boolean;
}

function ToolsPage() {
  const styles = useStyles2(getStyles);
  const [tools, setTools] = useState<ToolWithServer[]>([]);
  const [servers, setServers] = useState<MCPServerConfig[]>([]);
  const [statuses, setStatuses] = useState<MCPServerStatus[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [expandedTools, setExpandedTools] = useState<Set<string>>(new Set());
  const [selectedServer, setSelectedServer] = useState<string>('all');

  useEffect(() => {
    loadToolsData();
  }, []);

  const loadToolsData = async () => {
    console.log('loadToolsData() called');
    try {
      setLoading(true);
      setError(null);

      // Check service availability
      const available = await MCPService.isServiceAvailable();
      if (!available) {
        throw new Error('MCP service is not available');
      }

      // Load servers and their statuses
      const [serverConfigs, serverStatuses] = await Promise.all([
        MCPService.getServerConfigs(),
        MCPService.getServerStatuses()
      ]);

      setServers(serverConfigs);
      setStatuses(serverStatuses);

      // Build tools list with server information
      const toolsWithServerInfo: ToolWithServer[] = [];

      serverStatuses.forEach((status) => {
        const server = serverConfigs.find(s => s.id === status.id);
        const serverName = server?.name || status.id;
        const isAvailable = status.connected && server?.enabled;

        if (status.tools && status.tools.length > 0) {
          status.tools.forEach((tool) => {
            toolsWithServerInfo.push({
              ...tool,
              serverId: status.id,
              serverName,
              available: isAvailable || false
            });
          });
        }
      });

      console.log('Loaded tools with server info:', toolsWithServerInfo);
      setTools(toolsWithServerInfo);
    } catch (error) {
      console.error('Failed to load tools data:', error);
      setError(error instanceof Error ? error.message : 'Failed to load tools data');
    } finally {
      setLoading(false);
    }
  };

  const availableTools = tools.filter(tool => {
    const isAvailable = tool.available;
    const matchesServer = selectedServer === 'all' || tool.serverId === selectedServer;
    return isAvailable && matchesServer;
  });

  const unavailableTools = tools.filter(tool => {
    const isUnavailable = !tool.available;
    const matchesServer = selectedServer === 'all' || tool.serverId === selectedServer;
    return isUnavailable && matchesServer;
  });

  const getServerOptions = (): SelectableValue[] => {
    const options: SelectableValue[] = [
      { label: 'All Servers', value: 'all' }
    ];

    statuses.forEach(status => {
      const server = servers.find(s => s.id === status.id);
      const serverName = server?.name || status.id;
      const toolCount = status.tools?.length || 0;
      options.push({
        label: `${serverName} (${toolCount} tools)`,
        value: status.id
      });
    });

    return options;
  };

  const handleServerChange = (option: SelectableValue) => {
    setSelectedServer(option.value || 'all');
  };

  const toggleToolExpansion = (toolKey: string) => {
    setExpandedTools(prev => {
      const newSet = new Set(prev);
      if (newSet.has(toolKey)) {
        newSet.delete(toolKey);
      } else {
        newSet.add(toolKey);
      }
      return newSet;
    });
  };

  const getToolKey = (tool: ToolWithServer, index: number) =>
    `${tool.serverId}-${tool.name}-${index}`;

  const isToolExpanded = (toolKey: string) => expandedTools.has(toolKey);

  if (loading) {
    return (
      <PluginPage>
        <div className={styles.container}>
          <div className={styles.loadingState}>
            <Spinner size={24} />
            <span>Loading tools...</span>
          </div>
        </div>
      </PluginPage>
    );
  }

  if (error) {
    return (
      <PluginPage>
        <div className={styles.container}>
          <Alert title="Error loading tools" severity="error">
            {error}
          </Alert>
        </div>
      </PluginPage>
    );
  }

  return (
    <PluginPage>
      <div className={styles.container}>
        <div className={styles.header}>
          <h2>Tools & Capabilities</h2>
          <p>Available MCP tools from connected servers ({selectedServer === 'all' ? tools.length : availableTools.length + unavailableTools.length} {selectedServer === 'all' ? 'total' : 'filtered'} tools from {servers.length} servers)</p>

          <div className={styles.filterSection}>
            <div className={styles.filterItem}>
              <label className={styles.filterLabel}>Filter by Server:</label>
              <Select
                value={selectedServer}
                options={getServerOptions()}
                onChange={handleServerChange}
                width={30}
                placeholder="Select server..."
              />
            </div>
          </div>
        </div>

        {availableTools.length > 0 && (
          <div className={styles.section}>
            <h3>Available Tools ({availableTools.length})</h3>
            <div className={styles.toolList}>
              {availableTools.map((tool, index) => {
                const toolKey = getToolKey(tool, index);
                const expanded = isToolExpanded(toolKey);
                return (
                  <div key={toolKey} className={styles.toolItem}>
                    <div
                      className={styles.toolHeader}
                      onClick={() => toggleToolExpansion(toolKey)}
                    >
                      <div className={styles.toolName}>{tool.name}</div>
                      <div className={styles.expandIcon}>
                        {expanded ? '−' : '+'}
                      </div>
                    </div>
                    {expanded && (
                      <div className={styles.toolDetails}>
                        <div className={styles.toolMeta}>
                          <div><strong>Server:</strong> {tool.serverName || tool.serverId}</div>
                          <div><strong>Description:</strong> {tool.description}</div>
                          {tool.parameters && Object.keys(tool.parameters).length > 0 && (
                            <div className={styles.parametersSection}>
                              <strong>Parameters:</strong>
                              <div className={styles.parametersList}>
                                {Object.entries(tool.parameters).map(([key, value]) => (
                                  <div key={key} className={styles.parameterItem}>
                                    <code>{key}</code>: {typeof value === 'string' ? value : JSON.stringify(value)}
                                  </div>
                                ))}
                              </div>
                            </div>
                          )}
                        </div>
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          </div>
        )}

        {unavailableTools.length > 0 && (
          <div className={styles.section}>
            <h3>Unavailable Tools ({unavailableTools.length})</h3>
            <div className={styles.toolList}>
              {unavailableTools.map((tool, index) => {
                const toolKey = getToolKey(tool, index);
                const expanded = isToolExpanded(toolKey);
                return (
                  <div key={toolKey} className={`${styles.toolItem} ${styles.unavailableTool}`}>
                    <div
                      className={styles.toolHeader}
                      onClick={() => toggleToolExpansion(toolKey)}
                    >
                      <div className={styles.toolName}>{tool.name}</div>
                      <div className={styles.expandIcon}>
                        {expanded ? '−' : '+'}
                      </div>
                    </div>
                    {expanded && (
                      <div className={styles.toolDetails}>
                        <div className={styles.toolMeta}>
                          <div><strong>Server:</strong> {tool.serverName || tool.serverId}</div>
                          <div><strong>Description:</strong> {tool.description}</div>
                          {tool.parameters && Object.keys(tool.parameters).length > 0 && (
                            <div className={styles.parametersSection}>
                              <strong>Parameters:</strong>
                              <div className={styles.parametersList}>
                                {Object.entries(tool.parameters).map(([key, value]) => (
                                  <div key={key} className={styles.parameterItem}>
                                    <code>{key}</code>: {typeof value === 'string' ? value : JSON.stringify(value)}
                                  </div>
                                ))}
                              </div>
                            </div>
                          )}
                        </div>
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          </div>
        )}

        {(availableTools.length === 0 && unavailableTools.length === 0) && (
          <div className={styles.emptyState}>
            <h3>No Tools {selectedServer === 'all' ? 'Available' : 'Found'}</h3>
            <p>
              {selectedServer === 'all'
                ? 'No MCP tools found. Make sure servers are connected and enabled.'
                : 'No tools found for the selected server. Try selecting a different server or "All Servers".'}
            </p>
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
    margin-bottom: ${theme.spacing(3)};

    h2 {
      margin-bottom: ${theme.spacing(1)};
    }

    p {
      color: ${theme.colors.text.secondary};
      margin: 0 0 ${theme.spacing(2)} 0;
    }
  `,
  filterSection: css`
    display: flex;
    align-items: center;
    gap: ${theme.spacing(2)};
    padding: ${theme.spacing(1.5)};
    background: ${theme.colors.background.secondary};
    border-radius: ${theme.shape.borderRadius()};
    border: 1px solid ${theme.colors.border.weak};
  `,
  filterItem: css`
    display: flex;
    align-items: center;
    gap: ${theme.spacing(1)};
  `,
  filterLabel: css`
    font-weight: ${theme.typography.fontWeightMedium};
    color: ${theme.colors.text.primary};
    font-size: ${theme.typography.body.fontSize};
    white-space: nowrap;
  `,
  section: css`
    margin-bottom: ${theme.spacing(3)};

    h3 {
      margin-bottom: ${theme.spacing(2)};
      color: ${theme.colors.text.primary};
    }
  `,
  toolList: css`
    display: flex;
    flex-direction: column;
    gap: ${theme.spacing(1)};
  `,
  toolItem: css`
    background: ${theme.colors.background.secondary};
    border-radius: ${theme.shape.borderRadius()};
    border: 1px solid ${theme.colors.border.weak};
    overflow: hidden;
    transition: all 0.2s ease-in-out;

    &:hover {
      border-color: ${theme.colors.border.medium};
      box-shadow: ${theme.shadows.z1};
    }
  `,
  unavailableTool: css`
    opacity: 0.6;
    border-color: ${theme.colors.border.weak};

    &:hover {
      opacity: 0.8;
    }
  `,
  toolHeader: css`
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: ${theme.spacing(1)} ${theme.spacing(1.5)};
    cursor: pointer;
    user-select: none;

    &:hover {
      background: ${theme.colors.emphasize(theme.colors.background.secondary, 0.03)};
    }
  `,
  toolName: css`
    font-weight: ${theme.typography.fontWeightMedium};
    color: ${theme.colors.text.primary};
    font-size: ${theme.typography.body.fontSize};
    word-break: break-word;
  `,
  expandIcon: css`
    width: 20px;
    height: 20px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: ${theme.colors.action.hover};
    border-radius: 50%;
    font-weight: bold;
    font-size: 14px;
    color: ${theme.colors.text.primary};
    transition: all 0.2s ease-in-out;

    &:hover {
      background: ${theme.colors.action.selected};
    }
  `,
  toolDetails: css`
    border-top: 1px solid ${theme.colors.border.weak};
    padding: ${theme.spacing(1.5)};
    background: ${theme.colors.background.canvas};
    animation: slideDown 0.2s ease-out;

    @keyframes slideDown {
      from {
        opacity: 0;
        max-height: 0;
        padding-top: 0;
        padding-bottom: 0;
      }
      to {
        opacity: 1;
        max-height: 500px;
        padding-top: ${theme.spacing(1.5)};
        padding-bottom: ${theme.spacing(1.5)};
      }
    }
  `,
  toolMeta: css`
    color: ${theme.colors.text.secondary};
    font-size: ${theme.typography.bodySmall.fontSize};

    > div {
      margin-bottom: ${theme.spacing(0.5)};
    }

    strong {
      color: ${theme.colors.text.primary};
    }
  `,
  parametersSection: css`
    margin-top: ${theme.spacing(1)};
    padding: ${theme.spacing(1)};
    background: ${theme.colors.background.secondary};
    border-radius: ${theme.shape.borderRadius()};
    border-left: 3px solid ${theme.colors.info.main};
  `,
  parametersList: css`
    margin-top: ${theme.spacing(0.5)};
  `,
  parameterItem: css`
    margin-bottom: ${theme.spacing(0.25)};
    font-size: ${theme.typography.bodySmall.fontSize};

    code {
      background: ${theme.colors.background.canvas};
      padding: 2px 4px;
      border-radius: 2px;
      font-family: ${theme.typography.fontFamilyMonospace};
      font-weight: ${theme.typography.fontWeightMedium};
      color: ${theme.colors.text.primary};
    }
  `,
  loadingState: css`
    display: flex;
    align-items: center;
    justify-content: center;
    gap: ${theme.spacing(1)};
    padding: ${theme.spacing(4)};
    color: ${theme.colors.text.secondary};
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

export default ToolsPage;
