# Multi-Platform Build Guide

## Overview

The MCP Client App now supports multi-platform builds, creating binaries for all major Grafana deployment targets.

## Supported Platforms

✅ **macOS**
- Intel (x86_64): `gpx_mcpclient_darwin_amd64`
- Apple Silicon (ARM64): `gpx_mcpclient_darwin_arm64`

✅ **Linux**
- AMD64/x86_64: `gpx_mcpclient_linux_amd64`
- ARM 32-bit: `gpx_mcpclient_linux_arm`
- ARM 64-bit: `gpx_mcpclient_linux_arm64`

✅ **Windows**
- AMD64/x86_64: `gpx_mcpclient_windows_amd64.exe`

## Building

### Multi-Platform Build (Recommended)
```bash
npm run package:multi
```

### Single Platform Build (Development)
```bash
npm run build
GOOS=linux GOARCH=amd64 go build -o dist/gpx_mcpclient ./pkg
```

## Build Outputs

- **`package/`** - Complete plugin package with all platform binaries
- **`cisco-mcpclient-app-1.0.0.zip`** - Distribution ZIP file
- **`go_plugin_build_manifest`** - Build metadata and file hashes

## Script Features

🚀 **Automated Multi-Platform Compilation**
- Cross-compiles for 6 different platforms
- Handles platform-specific naming conventions
- Includes Windows `.exe` extension

🧹 **Build Environment Hygiene**
- Clears webpack/TypeScript caches before build
- Uses `npm ci` for reproducible dependency installation
- Strips debug symbols with `-ldflags="-w -s"`

📦 **Distribution Package Creation**
- Creates complete plugin package structure
- Includes installation instructions
- Generates ZIP file for easy distribution
- Adds version file and checksums

🔍 **Build Verification**
- Lists all created binaries with file sizes
- Generates manifest with file hashes
- Provides build summary and next steps

## Deployment

The plugin automatically selects the correct binary based on the target platform where Grafana is running.

## Example Usage

```bash
# Build for all platforms
cd cisco-mcpclient-app
npm run package:multi

# Deploy to different environments
# Linux server
scp cisco-mcpclient-app-1.0.0.zip server:/var/lib/grafana/plugins/
ssh server "cd /var/lib/grafana/plugins && unzip -o cisco-mcpclient-app-1.0.0.zip"

# Windows server
# Extract ZIP to C:\Program Files\GrafanaLabs\grafana\data\plugins\
```

## Build Script Location

- **Script**: `scripts/build-multiplatform.sh`
- **NPM Command**: `npm run package:multi`
- **Legacy single-platform**: `npm run package`

The multi-platform build script reverse-engineered from the working distribution ZIP provides production-ready binaries for all Grafana deployment scenarios.