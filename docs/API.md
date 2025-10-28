# API Reference

Backend REST API for the Grafana MCP Client plugin.

**Base URL:** `/api/plugins/grafana-mcpclient-app/resources`

## Endpoints

### Server Management

#### List Servers

```
GET /servers
```

Returns all configured MCP servers with current status.

**Response:**
```json
{
  "servers": [
    {
      "id": "server-1",
      "name": "Local MCP Server",
      "url": "http://localhost:8000/mcp",
      "type": "local",
      "enabled": true,
      "status": "connected",
      "tools": [{"name": "read_file", "description": "Read file contents"}],
      "capabilities": ["tools", "resources"]
    }
  ],
  "total": 1
}
```

#### Create Server

```
POST /servers
```

**Request:**
```json
{
  "name": "New Server",
  "url": "http://mcp-server:8000/mcp",
  "type": "remote",
  "enabled": true,
  "authType": "bearer"
}
```

#### Get Server

```
GET /servers/{id}
```

#### Update Server

```
PUT /servers/{id}
```

#### Delete Server

```
DELETE /servers/{id}
```

**Response:**
```json
{
  "success": true,
  "message": "Server deleted successfully",
  "serverId": "server-1"
}
```

#### Get Server Status

```
GET /servers/{id}/status
```

**Response:**
```json
{
  "status": "connected",
  "message": "Status check successful",
  "capabilities": ["tools", "resources"],
  "tools": [{"name": "read_file", "description": "Read file contents"}]
}
```

#### Test Server Connection

```
POST /servers/{id}/test
```

Performs real MCP connection test and updates server status.

---

### Tools

#### List Tools

```
GET /tools
```

Returns all tools from enabled, connected servers.

**Response:**
```json
{
  "tools": [
    {
      "name": "read_file",
      "description": "Read file contents",
      "parameters": {
        "path": "File path to read",
        "serverId": "server-1",
        "serverName": "Local MCP Server"
      }
    }
  ],
  "total": 1
}
```

#### Call Tool

```
POST /tools/call
```

Executes a tool on an MCP server.

**Request:**
```json
{
  "tool_name": "read_file",
  "arguments": {"path": "/etc/hosts"},
  "server_id": "server-1"
}
```

**Response:**
```json
{
  "success": true,
  "content": "127.0.0.1 localhost"
}
```

---

### Configuration

#### Get Config

```
GET /config
```

**Response:**
```json
{
  "autoDiscovery": true,
  "connectionTimeout": 30,
  "retryAttempts": 3,
  "enableLogging": true
}
```

#### Update Config

```
POST /config
```

#### Get Config Status

```
GET /config/status
```

**Response:**
```json
{
  "initialized": true,
  "config_source": "Grafana App Provisioning",
  "servers_count": 2,
  "servers": [{"id": "server-1", "name": "Local", "enabled": true, "status": "connected"}]
}
```

#### Reload Config

```
POST /config/reload
```

Configuration changes require Grafana restart when using provisioning.

---

### Connection Testing

#### Test Connection (without saving)

```
POST /test-connection
```

Tests connectivity to an MCP server without creating a server entry.

**Request:**
```json
{
  "url": "http://mcp-server:8000/mcp",
  "authType": "bearer",
  "authToken": "token"
}
```

**Response:**
```json
{
  "status": "connected",
  "message": "Connection successful! MCP server is responding.",
  "capabilities": ["tools"],
  "tools": [{"name": "read_file", "description": "Read file contents"}]
}
```

---

### Metrics & Health

#### Prometheus Metrics

```
GET /metrics
```

Returns Prometheus-format metrics including:
- `plugin_up` - Plugin health gauge
- `mcp_tool_calls_total` - Tool call counter by server/tool/status
- `mcp_request_latency_seconds` - Tool call latency histogram
- `mcp_connection_status` - Server connection gauge
- `mcp_errors_total` - Error counter by type

#### Ping

```
GET /ping
```

**Response:**
```json
{"message": "ok"}
```

---

## Error Handling

All endpoints return standard HTTP status codes with JSON error bodies.

**Error Response Format:**
```json
{
  "success": false,
  "error": "Tool 'unknown_tool' not found on any connected server"
}
```

**Status Codes:**
- `400` - Bad Request (invalid JSON, validation error)
- `404` - Not Found (server or tool not found)
- `405` - Method Not Allowed
- `500` - Internal Server Error

Tool argument validation failures return `400` with details:
```json
{
  "success": false,
  "error": "Invalid tool arguments: property 'path' is required"
}
```
