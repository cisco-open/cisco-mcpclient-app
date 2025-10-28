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
import { PluginPage, locationService } from '@grafana/runtime';
import {
  Card,
  Button,
  Field,
  Input,
  Select,
  Switch,
  Alert,
  useStyles2,
  TextArea
} from '@grafana/ui';
import { css } from '@emotion/css';
import { GrafanaTheme2, SelectableValue } from '@grafana/data';
import { useLocation } from 'react-router-dom';
import MCPService from '../services/MCPService';
import { MCPServerConfig } from '../types/mcp';

const serverTypeOptions: SelectableValue[] = [
  { label: 'Local Server', value: 'local' },
  { label: 'Remote Server', value: 'remote' },
];

const authTypeOptions: SelectableValue[] = [
  { label: 'No Authentication', value: 'none' },
  { label: 'Bearer Token', value: 'bearer' },
  { label: 'Basic Authentication', value: 'basic' },
];

function ServerConfigPage() {
  const styles = useStyles2(getStyles);
  const location = useLocation();

  // Extract URL parameters
  const searchParams = new URLSearchParams(location.search);
  const isNew = searchParams.get('new') === 'true';
  const serverId = searchParams.get('id');

  const [config, setConfig] = useState<MCPServerConfig>({
    id: '',
    name: '',
    url: '',
    type: 'local',
    enabled: true,
    authType: 'none',
    description: '',
  });

  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [isTesting, setIsTesting] = useState(false);
  const [testResult, setTestResult] = useState<{ success: boolean; message: string } | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Load existing server configuration if editing
  useEffect(() => {
    if (serverId && !isNew) {
      loadServerConfig(serverId);
    }
  }, [serverId, isNew]);

  const loadServerConfig = async (id: string) => {
    setLoading(true);
    setError(null);
    try {
      const serverConfig = await MCPService.getServer(id);
      if (serverConfig) {
        setConfig(serverConfig);
      } else {
        setError(`Server with ID ${id} not found`);
      }
    } catch (error) {
      console.error('Error loading server config:', error);
      setError('Failed to load server configuration');
    } finally {
      setLoading(false);
    }
  };

  const handleInputChange = (field: keyof MCPServerConfig) => (event: React.ChangeEvent<HTMLInputElement>) => {
    setConfig(prev => ({
      ...prev,
      [field]: event.target.value
    }));
  };

  const handleTextAreaChange = (field: keyof MCPServerConfig) => (event: React.ChangeEvent<HTMLTextAreaElement>) => {
    setConfig(prev => ({
      ...prev,
      [field]: event.target.value
    }));
  };

  const handleSelectChange = (field: keyof MCPServerConfig) => (value: SelectableValue) => {
    setConfig(prev => ({
      ...prev,
      [field]: value.value
    }));
  };

  const handleSwitchChange = (field: keyof MCPServerConfig) => (event: React.ChangeEvent<HTMLInputElement>) => {
    setConfig(prev => ({
      ...prev,
      [field]: event.target.checked
    }));
  };

  const handleTestConnection = async () => {
    if (!config.url.trim()) {
      setTestResult({
        success: false,
        message: 'Please enter a server URL before testing connection.'
      });
      return;
    }

    setIsTesting(true);
    setTestResult(null);

    try {
      // Use direct connection test - no need to create/delete temporary servers
      const authPayload = {
        authToken: config.authType === 'bearer' ? config.authToken : undefined,
        authUser: config.authType === 'basic' ? (config.authUser ?? config.username) : undefined,
        authPass: config.authType === 'basic' ? (config.authPass ?? config.password) : undefined,
      };
      const isConnected = await MCPService.testDirectConnection(
        config.url,
        config.authType || 'none',
        authPayload
      );

      setTestResult({
        success: isConnected,
        message: isConnected
          ? 'Connection successful! MCP server is responding.'
          : 'Connection failed. Please check the URL and authentication settings.'
      });
    } catch (error) {
      console.error('Error testing connection:', error);
      setTestResult({
        success: false,
        message: 'Connection test failed: ' + (error instanceof Error ? error.message : 'Unknown error')
      });
    } finally {
      setIsTesting(false);
    }
  };

  const handleSave = async () => {
    if (!config.name.trim() || !config.url.trim()) {
      setTestResult({
        success: false,
        message: 'Please fill in required fields (Name and URL).'
      });
      return;
    }

    setSaving(true);
    setError(null);
    setTestResult(null);

    try {
      if (isNew) {
        // Generate ID for new server if not set
        const newConfig = {
          ...config,
          id: config.id || `server-${Date.now()}`
        };
        await MCPService.addServer(newConfig);
        setTestResult({
          success: true,
          message: 'Server configuration saved successfully!'
        });
      } else {
        // Update existing server
        await MCPService.updateServer(config);
        setTestResult({
          success: true,
          message: 'Server configuration updated successfully!'
        });
      }

      // Navigate back to servers page after a short delay
      setTimeout(() => {
        locationService.push('/a/grafana-mcpclient-app');
      }, 1500);
    } catch (error) {
      console.error('Error saving server config:', error);
      setTestResult({
        success: false,
        message: 'Failed to save server configuration: ' + (error instanceof Error ? error.message : 'Unknown error')
      });
    } finally {
      setSaving(false);
    }
  };

  const handleCancel = () => {
    locationService.push('/a/grafana-mcpclient-app');
  };

  if (loading) {
    return (
      <PluginPage>
        <div className={styles.container}>
          <div className={styles.header}>
            <h2>{isNew ? 'Add New Server' : 'Edit Server Configuration'}</h2>
          </div>
          <div>Loading server configuration...</div>
        </div>
      </PluginPage>
    );
  }

  return (
    <PluginPage>
      <div className={styles.container}>
        <div className={styles.header}>
          <h2>{isNew ? 'Add New Server' : 'Edit Server Configuration'}</h2>
          <div className={styles.headerActions}>
            <Button variant="secondary" onClick={handleCancel}>
              Cancel
            </Button>
            <Button
              variant="primary"
              onClick={handleSave}
              disabled={saving || !config.name.trim() || !config.url.trim()}
            >
              {saving ? 'Saving...' : isNew ? 'Add Server' : 'Update Server'}
            </Button>
          </div>
        </div>

        {error && (
          <Alert title="Error" severity="error">
            {error}
          </Alert>
        )}

        {testResult && (
          <Alert
            title={testResult.success ? 'Success' : 'Error'}
            severity={testResult.success ? 'success' : 'error'}
          >
            {testResult.message}
          </Alert>
        )}

        <Card className={styles.configCard}>
          <Card.Heading>MCP Server Details</Card.Heading>

          <div className={styles.formGrid}>
            <Field label="Server Name" required>
              <Input
                value={config.name}
                onChange={handleInputChange('name')}
                placeholder="My MCP Server"
                disabled={saving}
              />
            </Field>

            <Field label="Server Type">
              <Select
                options={serverTypeOptions}
                value={config.type}
                onChange={handleSelectChange('type')}
                disabled={saving}
              />
            </Field>

            <Field label="Server URL" required className={styles.fullWidth}>
              <Input
                value={config.url}
                onChange={handleInputChange('url')}
                placeholder="http://localhost:3031"
                disabled={saving}
              />
            </Field>

            <Field label="Description" className={styles.fullWidth}>
              <TextArea
                value={config.description}
                onChange={handleTextAreaChange('description')}
                placeholder="Optional description of this MCP server..."
                rows={3}
                disabled={saving}
              />
            </Field>

            <Field label="Enabled" className={styles.switchField}>
              <Switch
                value={config.enabled}
                onChange={handleSwitchChange('enabled')}
                disabled={saving}
              />
            </Field>
          </div>
        </Card>

        <Card className={styles.configCard}>
          <Card.Heading>Authentication</Card.Heading>

          <div className={styles.formGrid}>
            <Field label="Authentication Type" className={styles.fullWidth}>
              <Select
                options={authTypeOptions}
                value={config.authType}
                onChange={handleSelectChange('authType')}
                disabled={saving}
              />
            </Field>

            {config.authType === 'bearer' && (
              <Field label="Bearer Token" className={styles.fullWidth}>
                <Input
                  type="password"
                  value={config.authToken || ''}
                  onChange={handleInputChange('authToken')}
                  placeholder="Enter bearer token..."
                  disabled={saving}
                />
              </Field>
            )}

            {config.authType === 'basic' && (
              <>
                <Field label="Username">
                  <Input
                    value={config.username || ''}
                    onChange={handleInputChange('username')}
                    placeholder="Enter username..."
                    disabled={saving}
                  />
                </Field>
                <Field label="Password">
                  <Input
                    type="password"
                    value={config.password || ''}
                    onChange={handleInputChange('password')}
                    placeholder="Enter password..."
                    disabled={saving}
                  />
                </Field>
              </>
            )}
          </div>
        </Card>

        <Card className={styles.configCard}>
          <Card.Heading>Connection Test</Card.Heading>
          <div className={styles.testSection}>
            <p>Test the connection to verify your MCP server configuration.</p>
            <Button
              variant="secondary"
              onClick={handleTestConnection}
              disabled={isTesting || saving || !config.url.trim()}
            >
              {isTesting ? 'Testing...' : 'Test Connection'}
            </Button>
          </div>
        </Card>
      </div>
    </PluginPage>
  );
}

const getStyles = (theme: GrafanaTheme2) => ({
  container: css`
    padding: ${theme.spacing(2)};
    max-width: 800px;
  `,
  header: css`
    margin-bottom: ${theme.spacing(3)};
    display: flex;
    justify-content: space-between;
    align-items: center;
  `,
  headerActions: css`
    display: flex;
    gap: ${theme.spacing(1)};
  `,
  configCard: css`
    margin-bottom: ${theme.spacing(2)};
    padding: ${theme.spacing(2)};
  `,
  formGrid: css`
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: ${theme.spacing(2)};
    align-items: start;
  `,
  fullWidth: css`
    grid-column: 1 / -1;
  `,
  switchField: css`
    display: flex;
    align-items: center;

    > label {
      margin-right: ${theme.spacing(1)};
      margin-bottom: 0;
    }
  `,
  testSection: css`
    display: flex;
    flex-direction: column;
    gap: ${theme.spacing(2)};
  `,
});

export default ServerConfigPage;
