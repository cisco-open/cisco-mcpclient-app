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
import { Route, Routes } from 'react-router-dom';
import { AppRootProps } from '@grafana/data';
import { ROUTES } from '../../constants';
import { PermissionGuard } from '../PermissionGuard';

const ServersPage = React.lazy(() => import('../../pages/ServersPage'));
const ServerConfigPage = React.lazy(() => import('../../pages/ServerConfigPage'));
const ToolsPage = React.lazy(() => import('../../pages/ToolsPage'));
const ImportExportPage = React.lazy(() => import('../../pages/ImportExportPage'));

function App(props: AppRootProps) {
  return (
    <PermissionGuard appName="MCP Client">
      <Routes>
        <Route path={ROUTES.Config} element={<ServerConfigPage />} />
        <Route path={ROUTES.Tools} element={<ToolsPage />} />
        <Route path={ROUTES.ImportExport} element={<ImportExportPage />} />

        {/* Default page - MCP Servers */}
        <Route path="*" element={<ServersPage />} />
      </Routes>
    </PermissionGuard>
  );
}

export default App;
