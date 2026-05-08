# Changelog

## 1.0.0 (Unreleased)

### Features

- MCP Client plugin for Grafana to configure and manage Model Context Protocol servers
- Server management UI with add, edit, delete, and health monitoring
- Tools & Capabilities page showing available MCP tools across servers
- Import/Export functionality for server configurations
- Permission-based access control (Admin role required)

### CI/CD

- CI pipeline: frontend lint, typecheck, unit tests, build + backend vet, test, build
- Multi-platform cross-compilation (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64)
- E2E tests with Playwright against Grafana 12.1.0
- Release workflow with multi-platform builds and placeholder plugin signing

### Maintenance

- Standardized plugin org to `grafana` across plugin.json and provisioning
- Replaced `lma`/`collab` references with `Cisco Systems, Inc.`
