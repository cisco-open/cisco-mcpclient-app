#!/bin/bash
# Copyright 2025 Cisco Systems, Inc. and its affiliates
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# SPDX-License-Identifier: Apache-2.0


# Build script for Grafana MCP Client App
set -e

echo "🔧 Building Grafana MCP Client App..."

# Check if we're in the right directory
if [ ! -f "package.json" ]; then
    echo "❌ Error: package.json not found. Please run this script from the plugin root directory."
    exit 1
fi

# Install dependencies if node_modules doesn't exist
if [ ! -d "node_modules" ]; then
    echo "📦 Installing npm dependencies..."
    npm install
fi

echo "🏗️  Building frontend..."
npm run build

# Check if Go backend exists
if [ -f "go.mod" ]; then
    echo "🔧 Building Go backend..."

    # Build for Linux (container target)
    echo "  - Building for Linux (container deployment)..."
    GOOS=linux GOARCH=amd64 go build -o dist/gpx_mcpclient ./pkg

    # Build for current platform (development)
    echo "  - Building for local development..."
    go build -o dist/gpx_mcpclient_local ./pkg

    echo "✅ Go backend built successfully"
fi

echo "✅ Build completed successfully!"
echo ""
echo "📁 Build artifacts:"
echo "   - Frontend: dist/ directory"
if [ -f "go.mod" ]; then
    echo "   - Backend (Linux): dist/gpx_mcpclient"
    echo "   - Backend (Local): dist/gpx_mcpclient_local"
fi
echo ""
echo "🚀 Ready for deployment!"