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
import { MemoryRouter } from 'react-router-dom';
import { AppRootProps, PluginType } from '@grafana/data';
import { render, waitFor } from '@testing-library/react';
import App from './App';

// Mock services to prevent backend calls
jest.mock('../../services/MCPService', () => ({
  MCPService: {
    getServerConfigs: jest.fn().mockResolvedValue([]),
    isLLMAvailable: jest.fn().mockResolvedValue(false),
  },
}));

describe('Components/App', () => {
  let props: AppRootProps;

  beforeEach(() => {
    jest.resetAllMocks();

    props = {
      basename: 'a/grafana-mcpclient-app',
      meta: {
        id: 'grafana-mcpclient-app',
        name: 'MCP Client',
        type: PluginType.app,
        enabled: true,
        jsonData: {},
      },
      query: {},
      path: '',
      onNavChanged: jest.fn(),
    } as unknown as AppRootProps;
  });

  test('renders without an error', async () => {
    const { container } = render(
      <MemoryRouter>
        <App {...props} />
      </MemoryRouter>
    );

    // Application lazy loads routes - verify the container renders
    await waitFor(
      () => {
        expect(container.firstChild).toBeTruthy();
      },
      { timeout: 2000 }
    );
  });
});
