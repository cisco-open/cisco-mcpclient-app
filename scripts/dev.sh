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


# Development script for Grafana MCP Client App
set -e

echo "🚀 Starting Grafana MCP Client App development environment..."

# Check if .env file exists
if [ ! -f ".env" ]; then
    echo "⚠️  Warning: .env file not found. Copying from .env.example..."
    if [ -f ".env.example" ]; then
        cp .env.example .env
        echo "📝 Please edit .env file with your actual values"
    else
        echo "❌ Error: .env.example not found. Please create .env file manually."
        exit 1
    fi
fi

# Check if config/mcp-servers.ini exists
if [ ! -f "config/mcp-servers.ini" ]; then
    echo "⚠️  Warning: config/mcp-servers.ini not found. Copying from example..."
    if [ -f "config/mcp-servers.ini.example" ]; then
        cp config/mcp-servers.ini.example config/mcp-servers.ini
        echo "📝 Please edit config/mcp-servers.ini with your actual MCP server configurations"
    else
        echo "❌ Error: config/mcp-servers.ini.example not found."
        exit 1
    fi
fi

# Start development environment
echo "🐳 Starting containers..."
docker-compose up -d

echo "⏳ Waiting for services to start..."
sleep 10

# Check service health
echo "🔍 Checking service status..."

if curl -s http://localhost:3000/api/health > /dev/null; then
    echo "✅ Grafana is running at http://localhost:3000"
    echo "   - Username: admin"
    echo "   - Password: admin"
else
    echo "⚠️  Grafana may still be starting up..."
fi

if curl -s http://localhost:3031 > /dev/null 2>&1; then
    echo "✅ MCP Server is running at http://localhost:3031"
else
    echo "⚠️  MCP Server may still be starting up..."
fi

echo ""
echo "🎯 Development Environment Ready!"
echo ""
echo "📚 Quick Commands:"
echo "   - Build plugin:     npm run build"
echo "   - Watch mode:       npm run dev"
echo "   - View logs:        docker-compose logs -f"
echo "   - Stop environment: docker-compose down"
echo ""
echo "🔗 Useful URLs:"
echo "   - Grafana:     http://localhost:3000"
echo "   - MCP Server:  http://localhost:3031"
echo ""
echo "📂 Edit your plugin source code and run 'npm run dev' for hot reload!"