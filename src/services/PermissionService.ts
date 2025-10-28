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

import { config } from '@grafana/runtime';

export interface UserPermissions {
  isAdmin: boolean;
  isEditor: boolean;
  isViewer: boolean;
  canAccessApps: boolean;
  orgRole: string;
}

export class PermissionService {
  static getUserPermissions(): UserPermissions {
    const user = config.bootData.user;
    const orgRole = user.orgRole?.toLowerCase() || 'viewer';
    
    const isAdmin = orgRole === 'admin';
    const isEditor = orgRole === 'editor' || isAdmin;
    const isViewer = orgRole === 'viewer' || isEditor;
    
    // Only Admin and Editor roles can access apps
    const canAccessApps = isAdmin || orgRole === 'editor';
    
    return {
      isAdmin,
      isEditor,
      isViewer,
      canAccessApps,
      orgRole: user.orgRole || 'Viewer'
    };
  }

  static canAccessMCPClient(): boolean {
    const { canAccessApps } = this.getUserPermissions();
    return canAccessApps;
  }

  static getUserInfo() {
    const user = config.bootData.user;
    return {
      login: user.login,
      name: user.name || user.login,
      email: user.email,
      orgRole: user.orgRole || 'Viewer',
      orgId: user.orgId
    };
  }
}
