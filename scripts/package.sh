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


set -e

echo "📦 Creating Grafana MCP Client App Distribution Package..."

# Configuration
PLUGIN_NAME="cisco-mcpclient-app"
VERSION=${VERSION:-"1.0.0"}
BUILD_DIR="dist"
PACKAGE_DIR="package"

# Clean previous build and package
echo "🧹 Cleaning previous build..."
rm -rf "$BUILD_DIR"
rm -rf "$PACKAGE_DIR"
mkdir -p "$PACKAGE_DIR"

# Install dependencies if needed
if [ ! -d "node_modules" ]; then
    echo "📦 Installing dependencies..."
    npm install
fi

# Type check
echo "🔍 Type checking..."
npm run typecheck

# Build frontend
echo "🏗️  Building frontend..."
npm run build

# Build Go backend for Linux (container target)
echo "🔧 Building Go backend..."
GOOS=linux GOARCH=amd64 go build -o "$BUILD_DIR/gpx_mcpclient" ./pkg

# Copy built plugin to package directory
echo "📂 Packaging plugin files..."
cp -r "$BUILD_DIR/"* "$PACKAGE_DIR/"

# Create README for distribution
cat > "$PACKAGE_DIR/INSTALLATION.md" << 'EOF'
# Grafana MCP Client App Installation

## Installation

1. Extract this archive to your Grafana plugins directory:
   ```bash
   # For standard Grafana installation
   unzip cisco-mcpclient-app.zip -d /var/lib/grafana/plugins/

   # For Docker/container deployments
   unzip cisco-mcpclient-app.zip -d ./grafana/plugins/
   ```

2. Configure unsigned plugin in Grafana:
   ```ini
   # In grafana.ini
   [plugins]
   allow_loading_unsigned_plugins = cisco-mcpclient-app
   ```

3. Restart Grafana

4. Access the MCP Client App via Apps → MCP Client

## Requirements

- Grafana 10.0+
- LLM App plugin (for MCP tool integration)
- Access to MCP servers (local or remote)

## Features

- MCP server configuration and management
- Real-time server health monitoring
- Tool discovery and listing
- File-based configuration (.ini format)
- Integration with Grafana LLM for AI tool calling

## Configuration

Create `mcp-servers.ini` in your Grafana config directory:

```ini
[local-server]
name=Local MCP Server
url=http://localhost:8000/mcp
type=local
enabled=true

[remote-server]
name=Production MCP Server
url=https://api.example.com/mcp
type=remote
enabled=true
auth_type=bearer
auth_token=${MCP_API_TOKEN}
```

For more information, visit: https://github.com/your-org/cisco-mcpclient-app
EOF

# Create version info
cat > "$PACKAGE_DIR/VERSION" << EOF
Plugin: $PLUGIN_NAME
Version: $VERSION
Build Date: $(date -u +"%Y-%m-%d %H:%M:%S UTC")
Build Platform: $(uname -s)/$(uname -m)
Backend: Linux/amd64 (gpx_mcpclient)
EOF

# Create the distribution zip
echo "🗜️  Creating distribution archive..."
zip -r "${PLUGIN_NAME}-${VERSION}.zip" "$PACKAGE_DIR"

# Clean up package directory
rm -rf "$PACKAGE_DIR"

echo "✅ Distribution package created successfully!"
echo ""
echo "📦 Package: ${PLUGIN_NAME}-${VERSION}.zip"
echo "📏 Size: $(du -h "${PLUGIN_NAME}-${VERSION}.zip" | cut -f1)"
echo ""
echo "🚀 Ready for distribution!"
echo "   - Upload to GitHub releases"
echo "   - Share with Grafana administrators"
echo "   - Install in any Grafana instance"
echo ""
echo "🔧 Backend Binary: Linux/amd64 (gpx_mcpclient)"
echo "   Compatible with Docker/Podman deployments"