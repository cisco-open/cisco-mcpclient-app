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

import React, { useState } from 'react';
import {
  Button,
  Field,
  Input,
  Select,
  Switch,
  Alert,
  useStyles2,
  Stack,
} from '@grafana/ui';
import { GrafanaTheme2, SelectableValue } from '@grafana/data';
import { css } from '@emotion/css';
import { MCPServerConfig, MCPServerAuth } from '../types/mcp';

interface ServerConfigFormProps {
  initialConfig?: MCPServerConfig;
  onSave: (config: MCPServerConfig) => void;
  onCancel: () => void;
}

const serverTypeOptions: SelectableValue[] = [
  { label: 'Local Server', value: 'local' },
  { label: 'Remote Server', value: 'remote' },
];

const authTypeOptions: SelectableValue[] = [
  { label: 'None', value: 'none' },
  { label: 'Bearer Token', value: 'bearer' },
  { label: 'Basic Authentication', value: 'basic' },
  { label: 'API Key', value: 'api-key' },
];

export const ServerConfigForm: React.FC<ServerConfigFormProps> = ({
  initialConfig,
  onSave,
  onCancel,
}) => {
  const styles = useStyles2(getStyles);
  const [config, setConfig] = useState<MCPServerConfig>(() => ({
    name: '',
    url: '',
    type: 'local',
    enabled: true,
    description: '',
    auth: {
      type: 'none'
    },
    ...initialConfig,
  }));
  const [isLoading, setIsLoading] = useState(false);
  const [testResult, setTestResult] = useState<{ success: boolean; message: string } | null>(null);

  const updateConfig = (updates: Partial<MCPServerConfig>) => {
    setConfig(prev => ({ ...prev, ...updates }));
  };

  const updateAuth = (updates: Partial<MCPServerAuth>) => {
    setConfig(prev => ({
      ...prev,
      auth: {
        type: 'none',
        ...prev.auth,
        ...updates
      }
    }));
  };

  const handleSave = async () => {
    if (!config.name.trim() || !config.url.trim()) {
      setTestResult({ success: false, message: 'Name and URL are required' });
      return;
    }

    try {
      setIsLoading(true);
      setTestResult(null);

      // Validate URL format
      try {
        new URL(config.url);
      } catch {
        setTestResult({ success: false, message: 'Invalid URL format' });
        return;
      }

      // Clean up the config before saving
      const cleanConfig = {
        ...config,
        name: config.name.trim(),
        url: config.url.trim(),
        description: config.description?.trim() || '',
      };

      onSave(cleanConfig);
    } catch (error) {
      setTestResult({
        success: false,
        message: error instanceof Error ? error.message : 'Failed to save server configuration'
      });
    } finally {
      setIsLoading(false);
    }
  };

  const handleTestConnection = async () => {
    if (!config.url.trim()) {
      setTestResult({ success: false, message: 'URL is required for testing' });
      return;
    }

    try {
      setIsLoading(true);
      setTestResult(null);

      // TODO: Replace with real MCP connection test
      // const response = await fetch('/api/plugins/cisco-mcpclient-app/resources/test', {
      //   method: 'POST',
      //   headers: { 'Content-Type': 'application/json' },
      //   body: JSON.stringify(config)
      // });
      // const result = await response.json();

      // Mock test for development - REMOVE when implementing real backend
      await new Promise(resolve => setTimeout(resolve, 1000));
      const success = Math.random() > 0.3;

      if (success) {
        setTestResult({ success: true, message: 'Connection successful!' });
      } else {
        setTestResult({ success: false, message: 'Connection failed - server unreachable' });
      }
    } catch (error) {
      setTestResult({
        success: false,
        message: error instanceof Error ? error.message : 'Connection test failed'
      });
    } finally {
      setIsLoading(false);
    }
  };

  const getDefaultUrlForType = (type: string) => {
    switch (type) {
      case 'local':
        return 'http://localhost:3040';
      default:
        return '';
    }
  };

  const handleTypeChange = (value: SelectableValue) => {
    const newType = value.value as 'local' | 'remote';
    updateConfig({
      type: newType,
      url: config.url || getDefaultUrlForType(newType)
    });
  };

  return (
    <div className={styles.container}>
      <Stack direction="column" gap={4}>
        <Stack direction="column" gap={2}>
          <Field label="Server Name" required>
            <Input
              value={config.name}
              onChange={(e) => updateConfig({ name: e.currentTarget.value })}
              placeholder="My MCP Server"
            />
          </Field>

          <Field label="Server Type">
            <Select
              value={serverTypeOptions.find(opt => opt.value === config.type)}
              onChange={handleTypeChange}
              options={serverTypeOptions}
            />
          </Field>

          <Field label="Server URL" required>
            <Input
              value={config.url}
              onChange={(e) => updateConfig({ url: e.currentTarget.value })}
              placeholder="http://localhost:3040"
            />
          </Field>

          <Field label="Description">
            <Input
              value={config.description || ''}
              onChange={(e) => updateConfig({ description: e.currentTarget.value })}
              placeholder="Optional description"
            />
          </Field>

          <Field label="Enabled">
            <Switch
              value={config.enabled}
              onChange={(e) => updateConfig({ enabled: e.currentTarget.checked })}
            />
          </Field>
        </Stack>

        {/* Authentication Section */}
        <div className={styles.authSection}>
          <h4>Authentication</h4>
          <Stack direction="column" gap={2}>
            <Field label="Authentication Type">
              <Select
                value={authTypeOptions.find(opt => opt.value === config.auth?.type)}
                onChange={(value) => updateAuth({ type: value.value as any })}
                options={authTypeOptions}
              />
            </Field>

            {config.auth?.type === 'bearer' && (
              <Field label="Bearer Token">
                <Input
                  type="password"
                  value={config.auth.token || ''}
                  onChange={(e) => updateAuth({ token: e.currentTarget.value })}
                  placeholder="Enter bearer token"
                />
              </Field>
            )}

            {config.auth?.type === 'basic' && (
              <>
                <Field label="Username">
                  <Input
                    value={config.auth.username || ''}
                    onChange={(e) => updateAuth({ username: e.currentTarget.value })}
                    placeholder="Enter username"
                  />
                </Field>
                <Field label="Password">
                  <Input
                    type="password"
                    value={config.auth.password || ''}
                    onChange={(e) => updateAuth({ password: e.currentTarget.value })}
                    placeholder="Enter password"
                  />
                </Field>
              </>
            )}

            {config.auth?.type === 'api-key' && (
              <>
                <Field label="API Key">
                  <Input
                    type="password"
                    value={config.auth.apiKey || ''}
                    onChange={(e) => updateAuth({ apiKey: e.currentTarget.value })}
                    placeholder="Enter API key"
                  />
                </Field>
                <Field label="Header Name">
                  <Input
                    value={config.auth.header || 'X-API-Key'}
                    onChange={(e) => updateAuth({ header: e.currentTarget.value })}
                    placeholder="X-API-Key"
                  />
                </Field>
              </>
            )}
          </Stack>
        </div>

        {/* Test Result */}
        {testResult && (
          <Alert
            severity={testResult.success ? 'success' : 'error'}
            title={testResult.success ? 'Success' : 'Error'}
          >
            {testResult.message}
          </Alert>
        )}

        {/* Actions */}
        <Stack direction="row" justifyContent="space-between" gap={2}>
          <Button
            variant="secondary"
            onClick={handleTestConnection}
            disabled={isLoading || !config.url.trim()}
          >
            Test Connection
          </Button>

          <Stack direction="row" gap={1}>
            <Button variant="secondary" onClick={onCancel} disabled={isLoading}>
              Cancel
            </Button>
            <Button
              variant="primary"
              onClick={handleSave}
              disabled={isLoading || !config.name.trim() || !config.url.trim()}
            >
              {isLoading ? 'Saving...' : initialConfig ? 'Update' : 'Add'} Server
            </Button>
          </Stack>
        </Stack>
      </Stack>
    </div>
  );
};

const getStyles = (theme: GrafanaTheme2) => ({
  container: css({
    padding: theme.spacing(2),
    maxWidth: '600px',
  }),

  authSection: css({
    border: `1px solid ${theme.colors.border.medium}`,
    borderRadius: theme.shape.radius.default,
    padding: theme.spacing(2),
    backgroundColor: theme.colors.background.secondary,

    '& h4': {
      margin: `0 0 ${theme.spacing(1)} 0`,
      color: theme.colors.text.primary,
    },
  }),
});
