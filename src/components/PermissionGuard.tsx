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
import { Alert, Container } from '@grafana/ui';
import { PermissionService } from '../services/PermissionService';

interface PermissionGuardProps {
  children: React.ReactNode;
  appName: string;
}

export const PermissionGuard: React.FC<PermissionGuardProps> = ({ children, appName }) => {
  const permissions = PermissionService.getUserPermissions();
  const userInfo = PermissionService.getUserInfo();

  if (!permissions.canAccessApps) {
    return (
      <Container>
        <Alert severity="error" title="Access Denied">
          <div>
            <p>
              You don't have permission to access {appName}.
            </p>
            <p>
              <strong>Current Role:</strong> {userInfo.orgRole}<br/>
              <strong>User:</strong> {userInfo.name} ({userInfo.login})
            </p>
            <p>
              <strong>Required Role:</strong> Admin or Editor
            </p>
            <p>
              Please contact your Grafana administrator to request appropriate permissions.
            </p>
          </div>
        </Alert>
      </Container>
    );
  }

  return <>{children}</>;
};
