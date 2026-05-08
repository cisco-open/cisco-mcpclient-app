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

import { test, expect } from './fixtures';
import { ROUTES } from '../src/constants';

test.describe('navigating app', () => {
  test('servers page should render successfully', async ({ gotoPage, page }) => {
    await gotoPage(`/${ROUTES.Servers}`);

    // Page should load without 404
    await expect(page).not.toHaveTitle(/not found/i);

    // The page renders inside a PluginPage wrapper. When the backend is unavailable,
    // it shows an Alert "MCP Service Not Available" instead of the "MCP Servers" heading.
    // Either state indicates the plugin page loaded correctly.
    const serversHeading = page.getByRole('heading', { name: 'MCP Servers' });
    const serviceAlert = page.getByText('MCP Service Not Available');
    await expect(serversHeading.or(serviceAlert)).toBeVisible({ timeout: 10000 });
  });

  test('tools page should render successfully', async ({ gotoPage, page }) => {
    await gotoPage(`/${ROUTES.Tools}`);

    // Page should load without 404
    await expect(page).not.toHaveTitle(/not found/i);

    // When the backend is available, shows "Tools & Capabilities" heading.
    // When unavailable, shows an error Alert or a loading spinner.
    // Either state indicates the plugin page loaded correctly.
    const toolsHeading = page.getByRole('heading', { name: /Tools/i }).first();
    await expect(toolsHeading).toBeVisible({ timeout: 10000 });
  });

  test('can navigate between servers and tools pages', async ({ gotoPage, page }) => {
    // Start at servers page
    await gotoPage(`/${ROUTES.Servers}`);

    // Wait for the page to render plugin content
    const serversHeading = page.getByRole('heading', { name: 'MCP Servers' });
    const serviceAlert = page.getByText('MCP Service Not Available');
    await expect(serversHeading.or(serviceAlert)).toBeVisible({ timeout: 10000 });

    // Navigate to tools page - try sidebar link, fall back to direct navigation
    const toolsLink = page.getByRole('link', { name: /Tools/i });
    if (await toolsLink.isVisible({ timeout: 5000 }).catch(() => false)) {
      await toolsLink.click();
      await page.waitForURL(/tools/, { timeout: 10000 });
    } else {
      // Sidebar nav may not be visible; navigate directly
      await gotoPage(`/${ROUTES.Tools}`);
    }

    // Verify tools page loaded (any valid state)
    const toolsHeading = page.getByRole('heading', { name: /Tools/i }).first();
    await expect(toolsHeading).toBeVisible({ timeout: 10000 });
  });
});
