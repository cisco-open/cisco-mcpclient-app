# Grafana MCP Client App - Development Guide

## Overview

The Grafana MCP Client App is a plugin for configuring and managing Model Context Protocol (MCP) servers within Grafana. This app allows users to connect to both local and remote MCP servers, configure authentication, and manage tool availability for LLM interactions.

## Quick Start

### 1. Prerequisites

- Docker or Podman
- Node.js 18+
- Go 1.21+ (for backend development)
- Git

### 2. Setup Development Environment

```bash
# Clone and setup
git clone <repository-url>
cd cisco-mcpclient-app

# Start development environment
./scripts/dev.sh
```

This will:

- Copy configuration templates if needed
- Start Grafana at <http://localhost:3000> (admin/admin)
- Start local MCP server at <http://localhost:3031>
- Display helpful development commands

### 3. Build and Develop

```bash
# Install dependencies
npm install

# Build the plugin
./scripts/build.sh

# Development with hot reload
npm run dev

# Watch for changes
npm run watch
```

## Architecture

### Frontend (TypeScript/React)

- **App Plugin**: Standalone application accessible via Grafana Apps menu
- **React Components**: Server configuration forms, tool discovery UI
- **Configuration Management**: Support for both UI-based and file-based (.ini) configuration
- **Real-time Updates**: Connection testing and server health monitoring

### Backend (Go)

- **MCP Client**: Real MCP protocol implementation over HTTP
- **Configuration Parser**: .ini file parsing with environment variable expansion
- **API Endpoints**: RESTful endpoints for frontend communication
- **Health Monitoring**: Connection status and tool discovery

## Development Workflow

### 1. Environment Setup

```bash
# Ensure you have a .env file
cp .env.example .env
# Edit .env with your actual values

# Ensure you have MCP server configuration
cp config/mcp-servers.ini.example config/mcp-servers.ini
# Edit config/mcp-servers.ini with your server details
```

### 2. Development Commands

```bash
# Start development environment
./scripts/dev.sh

# Build plugin (both frontend and backend)
./scripts/build.sh

# Frontend development
npm run dev          # Hot reload development
npm run build        # Production build
npm run watch        # Watch mode

# Backend development
go run ./pkg         # Run Go backend locally
go build ./pkg       # Build backend binary
go test ./pkg/...    # Run tests
```

### 3. Testing Your Changes

1. **Local Testing**: Use the development environment to test changes
2. **MCP Server Testing**: The included MCP server provides real tools for testing
3. **Configuration Testing**: Test both UI and file-based configuration approaches
4. **Integration Testing**: Verify MCP client can connect to real MCP servers

## Configuration

### Environment Variables

Create a `.env` file:

```bash
# Grafana API Token (for MCP server access)
GRAFANA_API_KEY=glsa_your_actual_api_key_here

# MCP Server Authentication (if using remote servers)
MCP_SERVER_URL=https://your-mcp-server.com
MCP_API_TOKEN=your_mcp_token_here
```

### MCP Server Configuration

Edit `config/mcp-servers.ini`:

```ini
[local-server]
name=Local Grafana MCP Server
url=http://grafana-mcp-server:8000/mcp
type=local
enabled=true
auth_type=none

[production-server]
name=Production MCP Server
url=${MCP_SERVER_URL}
type=remote
enabled=true
auth_type=bearer
auth_token=${MCP_API_TOKEN}
```

## Plugin Structure

```text
cisco-mcpclient-app/
├── src/                          # Frontend source code
│   ├── components/               # React components
│   ├── services/                 # API services
│   ├── types/                    # TypeScript definitions
│   └── pages/                    # App pages
├── pkg/                          # Backend Go source
│   ├── plugin/                   # Main plugin logic
│   ├── config/                   # Configuration parsing
│   └── mcp/                      # MCP client implementation
├── config/                       # Configuration files
│   └── mcp-servers.ini           # MCP server definitions
├── scripts/                      # Build and development scripts
│   ├── build.sh                  # Build script
│   └── dev.sh                    # Development environment
└── dist/                         # Build output
```

## API Reference

### Backend Endpoints

The Go backend provides these API endpoints:

- `GET /api/plugins/cisco-mcpclient-app/resources/servers` - List configured MCP servers
- `POST /api/plugins/cisco-mcpclient-app/resources/servers` - Add new MCP server
- `GET /api/plugins/cisco-mcpclient-app/resources/servers/{id}` - Get server details
- `PUT /api/plugins/cisco-mcpclient-app/resources/servers/{id}` - Update server configuration
- `DELETE /api/plugins/cisco-mcpclient-app/resources/servers/{id}` - Remove server
- `GET /api/plugins/cisco-mcpclient-app/resources/tools` - List available tools from all servers
- `POST /api/plugins/cisco-mcpclient-app/resources/config/reload` - Reload configuration from files

### Frontend Services

```typescript
// MCP Service for server management
import { MCPService } from './services/MCPService';

const mcpService = new MCPService();

// List servers
const servers = await mcpService.listServers();

// Test connection
const status = await mcpService.testConnection(serverConfig);

// List tools
const tools = await mcpService.listTools();
```

## Troubleshooting

### Common Issues

1. **Plugin not loading**: Check Grafana logs for unsigned plugin errors
2. **MCP connection fails**: Verify MCP server is running and accessible
3. **Authentication errors**: Check GRAFANA_API_KEY is valid
4. **Build fails**: Ensure Node.js and Go versions meet requirements

### Development Tips

1. **Use the development environment**: `./scripts/dev.sh` sets up everything you need
2. **Check container logs**: `docker-compose logs -f` for real-time debugging
3. **Test with real MCP servers**: Don't rely on mock data for development
4. **Environment variables**: Use `.env` file for local development secrets

### Debugging

```bash
# Check Grafana plugin logs
docker-compose logs grafana | grep mcpclient

# Check MCP server status
curl http://localhost:3031

# Test MCP server connection
curl -X POST http://localhost:3031 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "method": "initialize", "id": 1}'

# Check plugin API endpoints
curl http://localhost:3000/api/plugins/cisco-mcpclient-app/resources/servers
```

## Contributing

1. **Follow the established patterns**: Use existing TypeScript and Go patterns
2. **Test thoroughly**: Test with real MCP servers, not mock data
3. **Update documentation**: Keep this guide updated with any changes
4. **Production-ready code**: All features must be production quality

## Deployment

### Plugin Packaging

```bash
# Build for production
./scripts/build.sh

# The dist/ directory contains the complete plugin
# Package for distribution:
cd dist && zip -r cisco-mcpclient-app.zip .
```

### Installation

1. Extract plugin to Grafana plugins directory
2. Configure unsigned plugin in Grafana
3. Restart Grafana
4. Configure MCP servers via the app

For detailed deployment instructions, see the main repository documentation.