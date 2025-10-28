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
import { PluginPage } from '@grafana/runtime';
import { Card, Button, Alert, FileDropzone, useStyles2, CodeEditor, TabsBar, Tab, TabContent } from '@grafana/ui';
import { css } from '@emotion/css';
import { GrafanaTheme2 } from '@grafana/data';

function ImportExportPage() {
  const styles = useStyles2(getStyles);
  const [activeTab, setActiveTab] = useState('export');
  const [importFile, setImportFile] = useState<File | null>(null);
  const [exportData, setExportData] = useState('');
  const [importStatus, setImportStatus] = useState<{ type: 'success' | 'error'; message: string } | null>(null);

  // Mock configuration data
  const mockConfig = `[mcp-server-local]
url=http://localhost:3031
type=local
enabled=true
description=Local Grafana MCP Server

[mcp-server-remote]
url=https://api.example.com/mcp
type=remote
enabled=false
auth_type=bearer
auth_token=your_token_here
description=Remote API MCP Server`;

  const handleExport = () => {
    setExportData(mockConfig);
  };

  const handleDownloadConfig = () => {
    const blob = new Blob([exportData], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'mcp-servers.ini';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  const handleFileSelect = (files: File[]) => {
    if (files.length > 0) {
      setImportFile(files[0]);
      setImportStatus(null);
    }
  };

  const handleImport = async () => {
    if (!importFile) return;

    try {
      const content = await importFile.text();
      // TODO: Parse and validate .ini file content
      console.log('Importing configuration:', content);

      setImportStatus({
        type: 'success',
        message: 'Configuration imported successfully! Found 2 server configurations.'
      });
    } catch (error) {
      setImportStatus({
        type: 'error',
        message: 'Failed to import configuration. Please check the file format.'
      });
    }
  };

  const tabs = [
    { label: 'Export', value: 'export' },
    { label: 'Import', value: 'import' }
  ];

  return (
    <PluginPage>
      <div className={styles.container}>
        <div className={styles.header}>
          <h2>Import/Export Configuration</h2>
          <p>Manage MCP server configurations using .ini files</p>
        </div>

        <Card className={styles.mainCard}>
          <TabsBar>
            {tabs.map((tab) => (
              <Tab
                key={tab.value}
                label={tab.label}
                active={activeTab === tab.value}
                onChangeTab={() => setActiveTab(tab.value)}
              />
            ))}
          </TabsBar>

          <TabContent>
            {activeTab === 'export' && (
              <div className={styles.tabContent}>
                <div className={styles.section}>
                  <h4>Export Current Configuration</h4>
                  <p>Export your current MCP server configurations to a .ini file for backup or sharing.</p>

                  <div className={styles.actions}>
                    <Button variant="primary" onClick={handleExport}>
                      Generate Configuration
                    </Button>
                    {exportData && (
                      <Button variant="secondary" onClick={handleDownloadConfig}>
                        Download .ini File
                      </Button>
                    )}
                  </div>

                  {exportData && (
                    <div className={styles.codeSection}>
                      <h5>Generated Configuration:</h5>
                      <CodeEditor
                        value={exportData}
                        language="ini"
                        showLineNumbers
                        showMiniMap={false}
                        height="300px"
                        readOnly
                      />
                    </div>
                  )}
                </div>
              </div>
            )}

            {activeTab === 'import' && (
              <div className={styles.tabContent}>
                <div className={styles.section}>
                  <h4>Import Configuration</h4>
                  <p>Import MCP server configurations from a .ini file. This will add new servers and update existing ones.</p>

                  <div className={styles.dropzoneContainer}>
                    <FileDropzone
                      onFileRemove={() => setImportFile(null)}
                      options={{
                        accept: {
                          'text/plain': ['.ini', '.txt']
                        },
                        multiple: false,
                        onDrop: handleFileSelect
                      }}
                    />
                  </div>

                  {importFile && (
                    <div className={styles.fileInfo}>
                      <p><strong>Selected file:</strong> {importFile.name}</p>
                      <p><strong>Size:</strong> {(importFile.size / 1024).toFixed(2)} KB</p>

                      <div className={styles.actions}>
                        <Button variant="primary" onClick={handleImport}>
                          Import Configuration
                        </Button>
                        <Button variant="secondary" onClick={() => setImportFile(null)}>
                          Clear Selection
                        </Button>
                      </div>
                    </div>
                  )}

                  {importStatus && (
                    <Alert
                      title={importStatus.type === 'success' ? 'Import Successful' : 'Import Failed'}
                      severity={importStatus.type}
                    >
                      {importStatus.message}
                    </Alert>
                  )}
                </div>

                <div className={styles.section}>
                  <h5>Configuration File Format</h5>
                  <p>The .ini file should follow this format:</p>
                  <CodeEditor
                    value={`[server-name]
url=http://localhost:3031
type=local|remote
enabled=true|false
description=Optional description
auth_type=none|bearer|basic
auth_token=your_token (if bearer)
username=user (if basic)
password=pass (if basic)`}
                    language="ini"
                    showLineNumbers
                    showMiniMap={false}
                    height="200px"
                    readOnly
                  />
                </div>
              </div>
            )}
          </TabContent>
        </Card>
      </div>
    </PluginPage>
  );
}

const getStyles = (theme: GrafanaTheme2) => ({
  container: css`
    padding: ${theme.spacing(2)};
    max-width: 900px;
  `,
  header: css`
    margin-bottom: ${theme.spacing(3)};

    h2 {
      margin-bottom: ${theme.spacing(1)};
    }

    p {
      color: ${theme.colors.text.secondary};
      margin: 0;
    }
  `,
  mainCard: css`
    padding: ${theme.spacing(2)};
  `,
  tabContent: css`
    padding-top: ${theme.spacing(2)};
  `,
  section: css`
    margin-bottom: ${theme.spacing(3)};

    h4, h5 {
      margin-bottom: ${theme.spacing(1)};
      color: ${theme.colors.text.primary};
    }

    p {
      color: ${theme.colors.text.secondary};
      margin-bottom: ${theme.spacing(2)};
    }
  `,
  actions: css`
    display: flex;
    gap: ${theme.spacing(1)};
    margin-bottom: ${theme.spacing(2)};
  `,
  codeSection: css`
    margin-top: ${theme.spacing(2)};

    h5 {
      margin-bottom: ${theme.spacing(1)};
    }
  `,
  dropzoneContainer: css`
    margin-bottom: ${theme.spacing(2)};
    border: 2px dashed ${theme.colors.border.medium};
    border-radius: ${theme.shape.radius.default};
    padding: ${theme.spacing(2)};
  `,
  fileInfo: css`
    background: ${theme.colors.background.secondary};
    padding: ${theme.spacing(2)};
    border-radius: ${theme.shape.radius.default};
    margin-bottom: ${theme.spacing(2)};

    p {
      margin-bottom: ${theme.spacing(1)};
      color: ${theme.colors.text.primary};
    }
  `,
});

export default ImportExportPage;
