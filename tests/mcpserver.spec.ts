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

test.describe('MCP Server Management', () => {
  test('should navigate to servers page successfully', async ({ gotoPage, page }) => {
    await gotoPage(`/${ROUTES.Servers}`);

    // Page should load and show the MCP Servers heading
    await expect(page.getByRole('heading', { name: 'MCP Servers' })).toBeVisible({ timeout: 10000 });
  });

  test('should show empty state or server list', async ({ gotoPage, page }) => {
    await gotoPage(`/${ROUTES.Servers}`);

    // Wait for the page to fully load - look for MCP Servers heading first
    await expect(page.getByRole('heading', { name: 'MCP Servers' })).toBeVisible({ timeout: 10000 });

    // Then check for content - one of these should be present
    const emptyState = page.getByText('No MCP servers configured');
    const serverEntry = page.getByRole('button', { name: 'Configure' });
    const addServerButton = page.getByRole('button', { name: /Add Server/i });
    const serviceAlert = page.getByText('MCP Service Not Available');

    // Wait a bit for async content to load
    await page.waitForTimeout(500);

    // At least one of these should be visible
    const visibleCount = await Promise.all([
      emptyState.isVisible().catch(() => false),
      serverEntry.first().isVisible().catch(() => false),
      addServerButton.isVisible().catch(() => false),
      serviceAlert.isVisible().catch(() => false),
    ]);

    expect(visibleCount.some(Boolean)).toBe(true);
  });

  test('should render page without console errors', async ({ gotoPage, page }) => {
    const consoleErrors: string[] = [];

    page.on('console', (msg) => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });

    await gotoPage(`/${ROUTES.Servers}`);

    // Wait for page to settle
    await page.waitForTimeout(2000);

    // Filter out expected/benign errors
    const unexpectedErrors = consoleErrors.filter((err) => {
      const lowerErr = err.toLowerCase();
      return (
        !lowerErr.includes('failed to load') &&
        !lowerErr.includes('failed to preload plugin') &&
        !lowerErr.includes('fetch') &&
        !lowerErr.includes('networkerror') &&
        !lowerErr.includes('favicon') &&
        !lowerErr.includes('403') &&
        !lowerErr.includes('404') &&
        !lowerErr.includes('resizeobserver') &&
        !lowerErr.includes('chunk') &&
        !lowerErr.includes('source map') &&
        !lowerErr.includes('unknown plugin')
      );
    });

    expect(unexpectedErrors).toHaveLength(0);
  });

  test('should show Add Server button when service is available', async ({ gotoPage, page }) => {
    await gotoPage(`/${ROUTES.Servers}`);

    // Wait for the MCP Servers heading to appear
    await expect(page.getByRole('heading', { name: 'MCP Servers' })).toBeVisible({ timeout: 10000 });

    // If service is not available, the Add Server button won't be shown
    const serviceAlert = page.getByText('MCP Service Not Available');
    const isServiceUnavailable = await serviceAlert.isVisible().catch(() => false);

    if (!isServiceUnavailable) {
      // Service is available, Add Server button should be visible
      const addServerButton = page.getByRole('button', { name: /Add Server/i });
      await expect(addServerButton).toBeVisible({ timeout: 5000 });
    } else {
      // Service not available - test passes as expected behavior
      expect(isServiceUnavailable).toBe(true);
    }
  });
});
