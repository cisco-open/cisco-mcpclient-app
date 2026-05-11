# Contributing to Grafana MCP Client App

Thank you for your interest in contributing to the Grafana MCP Client App! This guide will help you get started with development and contributions.

## Table of Contents

- [Overview](#overview)
- [Development Environment Setup](#development-environment-setup)
- [Project Structure](#project-structure)
- [Development Workflow](#development-workflow)
- [Testing](#testing)
- [Code Standards](#code-standards)
- [Submission Guidelines](#submission-guidelines)
- [Release Process](#release-process)

## Overview

The Grafana MCP Client App is a Grafana application plugin that enables configuration and management of Model Context Protocol (MCP) servers. It provides both UI-based configuration and provisioning-based configuration for production deployments.

### Key Features

- MCP server configuration and management
- Real-time connection testing and health monitoring
- Tool discovery and capability reporting
- Integration with Grafana's provisioning system
- Support for local and remote MCP servers

## Development Environment Setup

### Prerequisites

- Node.js 18+ and npm
- Go 1.21+ (for backend development)
- Grafana 10.0+ development environment
- Docker or Podman (for testing)

### Initial Setup

1. **Clone the repository**

   ```bash
   git clone <repository-url>
   cd cisco-mcpclient-app
   ```

2. **Install dependencies**

   ```bash
   npm install
   ```

3. **Set up development environment**

   ```bash
   # Start local Grafana instance with plugin development setup
   # Option 1: Using Docker Compose (recommended)
   docker-compose up -d

   # Option 2: Using existing Grafana installation
   # Copy plugin to Grafana plugins directory after building
   ```

4. **Build and install plugin**

   ```bash
   # Development build
   npm run dev

   # Production build
   npm run build

   # Build Go backend for container deployment
   ./scripts/build.sh
   ```

### Development Environment Details

For development and testing, you can use:

**Option 1: Docker Compose (Recommended)**
The project includes a `docker-compose.yml` with:

- **Grafana** (port 3000): Development platform with plugin pre-installed
- **Test MCP Server** (optional): For testing MCP functionality

**Option 2: Local Grafana Installation**

- Install Grafana locally
- Copy built plugin to Grafana's plugins directory
- Configure Grafana to allow unsigned plugins

Access Grafana at `http://localhost:3000` (admin/admin) after setup.

## Project Structure

```
cisco-mcpclient-app/
├── src/                          # Frontend TypeScript/React code
│   ├── components/               # React components
│   │   ├── ServerList.tsx        # MCP server listing UI
│   │   ├── ServerForm.tsx        # Server configuration forms
│   │   └── PermissionGuard.tsx   # Access control wrapper
│   ├── pages/                    # Application pages
│   │   ├── ConfigPage.tsx        # Main configuration page
│   │   └── ToolsPage.tsx         # Tools discovery page
│   ├── services/                 # Frontend services
│   │   ├── BackendService.ts     # Backend API communication
│   │   └── ConfigService.ts      # Configuration management
│   └── types/                    # TypeScript type definitions
├── pkg/                          # Go backend code
│   ├── config/                   # Configuration management
│   │   └── provisioning_config.go # Provisioning system integration
│   ├── plugin/                   # Plugin implementation
│   │   ├── app.go                # Main plugin app
│   │   ├── mcpclient.go          # MCP client implementation
│   │   └── resources.go          # HTTP API endpoints
│   └── main.go                   # Plugin entry point
├── provisioning/                 # Grafana provisioning configuration
│   └── plugins/apps.yaml         # App provisioning example
├── config/                       # Configuration files
│   └── mcp-servers.ini.example   # Example server configuration
├── scripts/                      # Build and development scripts
├── tests/                        # Test files and fixtures
└── docs/                         # Documentation
```

### Key Files

- **`src/plugin.json`**: Plugin metadata and configuration
- **`pkg/plugin/resources.go`**: Backend API endpoint definitions
- **`pkg/config/provisioning_config.go`**: Grafana provisioning integration
- **`provisioning/plugins/apps.yaml`**: Example app provisioning configuration

## Development Workflow

### Backend Development (Go)

1. **Make changes** to Go code in `pkg/` directory
2. **Build backend** for your development platform:

   ```bash
   # Local development (macOS/Linux)
   go build -o dist/gpx_mcpclient ./pkg

   # Container deployment (Linux)
   GOOS=linux GOARCH=amd64 go build -o dist/gpx_mcpclient ./pkg
   ```

3. **Test locally** before containerization
4. **Deploy to container** for integration testing

### Frontend Development (TypeScript/React)

1. **Make changes** to TypeScript/React code in `src/` directory
2. **Build frontend**:

   ```bash
   npm run dev    # Development build with watch
   npm run build  # Production build
   ```

3. **Test in browser** at `http://localhost:3000`
4. **Check console** for errors and warnings

### Full Development Cycle

1. **Start development environment**:

   ```bash
   # Using Docker Compose
   docker-compose up -d

   # OR using local Grafana
   # Start your local Grafana instance
   ```

2. **Make your changes** to backend and/or frontend code

3. **Build and deploy**:

   ```bash
   ./scripts/build.sh                    # Build both frontend and backend

   # For Docker Compose
   docker-compose restart grafana

   # For local Grafana
   # Copy dist/ to your Grafana plugins directory
   ```

4. **Test in Grafana**:
   - Navigate to Apps → MCP Client
   - Test your changes
   - Check browser console and Grafana logs

5. **Iterate** until satisfied

### Configuration Testing

Test both configuration methods:

1. **UI Configuration**: Use the web interface to add/edit servers
2. **Provisioning Configuration**: Test with `provisioning/plugins/apps.yaml`

## Testing

### Manual Testing

1. **Basic Functionality**:
   - [ ] Plugin loads without errors
   - [ ] Server configuration form works
   - [ ] Connection testing functions properly
   - [ ] Tool discovery displays correctly

2. **MCP Integration**:
   - [ ] Connects to local MCP server (port 3031)
   - [ ] Discovers available tools
   - [ ] Reports server capabilities
   - [ ] Handles connection errors gracefully

3. **Configuration Persistence**:
   - [ ] UI configurations save and reload
   - [ ] Provisioning configurations load properly
   - [ ] Settings survive Grafana restarts

### Automated Testing

```bash
# Run frontend tests
npm test

# Run Go tests
go test ./pkg/...

# Build verification
npm run build && echo "Frontend build successful"
go build -o dist/gpx_mcpclient ./pkg && echo "Backend build successful"
```

### Integration Testing

Test with the complete environment:

```bash
# Start test environment
docker-compose up -d

# Verify Grafana is running
curl http://localhost:3000/api/health

# Test plugin installation
./scripts/build.sh && docker-compose restart grafana

# Optional: Test with external MCP server
# curl http://your-mcp-server:port  # Should connect to your MCP server
```

## Code Standards

### TypeScript/React Guidelines

- **Use TypeScript** for all frontend code
- **Follow React hooks patterns** for state management
- **Use Grafana UI components** (`@grafana/ui`) when possible
- **Implement proper error handling** with user-friendly messages
- **Add TypeScript types** for all data structures

### Go Guidelines

- **Follow Go conventions** (gofmt, golint)
- **Use proper error handling** with meaningful messages
- **Implement comprehensive logging** for debugging
- **Write testable code** with dependency injection
- **Document exported functions** with GoDoc comments

### General Standards

- **Meaningful commit messages** following conventional commits
- **Code documentation** for complex functions
- **Error handling** with user-friendly messages
- **Logging** for debugging and monitoring
- **Configuration validation** for all user inputs

### Example Code Patterns

#### Frontend Service Call

```typescript
// Good: Proper error handling and typing
async function fetchServers(): Promise<MCPServer[]> {
  try {
    const response = await getBackendSrv().get('/api/plugins/cisco-mcpclient-app/resources/servers');
    return response.servers || [];
  } catch (error) {
    console.error('Failed to fetch servers:', error);
    throw new Error('Unable to load MCP servers. Please check your configuration.');
  }
}
```

#### Backend API Endpoint

```go
// Good: Proper validation and error responses
func (a *App) handleServers(w http.ResponseWriter, r *http.Request) {
    if r.Method == http.MethodGet {
        servers, err := a.config.GetServers()
        if err != nil {
            http.Error(w, "Failed to load servers", http.StatusInternalServerError)
            return
        }

        response := map[string]interface{}{
            "servers": servers,
            "status":  "success",
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(response)
    }
    // ... handle other methods
}
```

## Submission Guidelines

### Before Submitting

1. **Test thoroughly** with the development environment
2. **Verify build process** works correctly
3. **Check for TypeScript/Go compilation errors**
4. **Test both UI and provisioning configuration methods**
5. **Update documentation** if needed

### Pull Request Process

1. **Create feature branch** from main
2. **Make focused commits** with clear messages
3. **Test changes** in development environment
4. **Update CHANGELOG.md** with your changes
5. **Submit pull request** with description of changes

### Pull Request Template

```markdown
## Description
Brief description of changes made.

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Documentation update
- [ ] Refactoring

## Testing
- [ ] Tested in development environment
- [ ] Verified build process works
- [ ] Tested both UI and provisioning configuration
- [ ] No TypeScript/Go compilation errors

## Checklist
- [ ] Code follows project style guidelines
- [ ] Added/updated tests as needed
- [ ] Updated documentation as needed
- [ ] CHANGELOG.md updated
```

## Release Process

### Version Management

The project uses semantic versioning (semver):

- **MAJOR**: Breaking changes to configuration or API
- **MINOR**: New features, backward compatible
- **PATCH**: Bug fixes, backward compatible

### Release Steps

1. **Update version** in `package.json` and `src/plugin.json`
2. **Update CHANGELOG.md** with release notes
3. **Create release build**:

   ```bash
   ./scripts/package.sh  # Creates distribution package
   ```

4. **Test release package** in clean environment
5. **Create git tag** and push to repository
6. **Create GitHub release** with package attachment

### Distribution Package

The release process creates a ZIP package with:

- Frontend build artifacts
- Multi-platform Go binaries (6 platforms)
- Plugin metadata and documentation
- Installation instructions

## Getting Help

### Resources

- **Project Documentation**: See `docs/` directory
- **Grafana Plugin Documentation**: https://grafana.com/docs/grafana/latest/developers/plugins/
- **MCP Specification**: https://spec.modelcontextprotocol.io/
- **Go Documentation**: https://golang.org/doc/
- **TypeScript/React**: https://react.dev/ and https://www.typescriptlang.org/

### Community

- **Issues**: Report bugs and request features via GitHub issues
- **Discussions**: Use GitHub discussions for questions and ideas
- **Development**: Join development discussions in pull requests

### Development Environment Troubleshooting

**Plugin not loading**:

- Check Grafana logs for error messages
- Verify plugin is in correct directory structure
- Ensure unsigned plugin configuration is correct

**Build failures**:

- Verify Node.js and Go versions meet requirements
- Check for TypeScript compilation errors
- Ensure all dependencies are installed

**MCP connection issues**:

- Verify MCP server is running on specified port
- Check network connectivity between containers
- Review MCP server logs for errors

Thank you for contributing to the Grafana MCP Client App!
