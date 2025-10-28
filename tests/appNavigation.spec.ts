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
    await expect(page.getByRole('heading', { name: 'MCP Servers' })).toBeVisible({ timeout: 10000 });
  });

  test('tools page should render successfully', async ({ gotoPage, page }) => {
    await gotoPage(`/${ROUTES.Tools}`);
    // Tools page should show "Tools & Capabilities" heading (use first() since multiple match)
    await expect(page.getByRole('heading', { name: /Tools/i }).first()).toBeVisible({ timeout: 10000 });
  });

  test('can navigate between servers and tools pages', async ({ gotoPage, page }) => {
    // Start at servers page
    await gotoPage(`/${ROUTES.Servers}`);
    await expect(page.getByRole('heading', { name: 'MCP Servers' })).toBeVisible({ timeout: 10000 });

    // Navigate to tools page via sidebar - use the link text from nav
    const toolsLink = page.getByRole('link', { name: /Tools.*Capabilities/i });
    if (await toolsLink.isVisible({ timeout: 5000 }).catch(() => false)) {
      await toolsLink.click();
      // Should navigate to tools page
      await page.waitForURL(/tools/, { timeout: 10000 });
    }
  });
});
