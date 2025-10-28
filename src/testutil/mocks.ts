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

import { of, throwError } from 'rxjs';

/**
 * Mock fetch function for @grafana/runtime getBackendSrv().fetch()
 * Can be configured per test to return different responses
 */
export const mockFetch = jest.fn();

/**
 * Mock backend service that uses mockFetch
 */
export const mockBackendSrv = {
  fetch: mockFetch,
};

/**
 * Reset all mocks between tests
 */
export const resetMocks = (): void => {
  mockFetch.mockReset();
};

/**
 * Configure mockFetch to return a successful response
 * @param data - The data to return in the response
 * @param status - HTTP status code (default: 200)
 */
export const mockFetchSuccess = <T>(data: T, status = 200): void => {
  mockFetch.mockReturnValue(of({ status, data }));
};

/**
 * Configure mockFetch to return an error
 * @param message - Error message
 */
export const mockFetchError = (message: string): void => {
  mockFetch.mockReturnValue(throwError(() => new Error(message)));
};

/**
 * Configure mockFetch to return a 201 Created response
 * @param data - The data to return in the response
 */
export const mockFetchCreated = <T>(data: T): void => {
  mockFetch.mockReturnValue(of({ status: 201, data }));
};

/**
 * Configure mockFetch to return a 404 Not Found response
 */
export const mockFetchNotFound = (): void => {
  mockFetch.mockReturnValue(of({ status: 404, data: { error: 'Not found' } }));
};

/**
 * Configure mockFetch to return a 500 Internal Server Error response
 * @param message - Error message
 */
export const mockFetchServerError = (message = 'Internal server error'): void => {
  mockFetch.mockReturnValue(of({ status: 500, data: { error: message } }));
};

/**
 * Mock implementation function that can be used with jest.mock
 * Returns a factory that provides the mock backend service
 */
export const createMockGrafanaRuntime = (): {
  getBackendSrv: () => typeof mockBackendSrv;
} => ({
  getBackendSrv: () => mockBackendSrv,
});
