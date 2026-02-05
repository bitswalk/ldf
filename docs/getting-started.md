# Getting Started

This guide covers building LDF from source and running it locally.

## Prerequisites

- **Go 1.24+** -- [golang.org/dl](https://go.dev/dl/)
- **Taskfile** -- [taskfile.dev/installation](https://taskfile.dev/installation/)
- **Bun** -- [bun.sh](https://bun.sh/) (required for building the WebUI)
- **Git** or **Sapling SCM**

## Building from Source

Clone the repository and build all components:

```bash
git clone https://github.com/bitswalk/ldf.git
cd ldf

# Build everything (server, CLI, WebUI)
task build

# Or build individual components
task build:srv      # Server only
task build:webui    # WebUI only
```

For development builds with debug symbols:

```bash
task build:dev
```

Build artifacts are placed in `build/bin/`.

## Running the Server

Start the server in development mode:

```bash
task run:srv:dev
```

This starts ldfd with stdout logging and debug-level output. The server listens on port **8443** by default.

To run the production binary directly:

```bash
./build/bin/ldfd
```

You can specify a configuration file:

```bash
./build/bin/ldfd --config /path/to/ldfd.yml
```

See [Configuration](configuration.md) for all available options.

## Accessing the WebUI

Once ldfd is running, open your browser at:

```
http://localhost:8443
```

The WebUI is a SolidJS single-page application served by ldfd.

## API Documentation

Interactive API documentation is available at:

```
http://localhost:8443/swagger/index.html
```

The Swagger UI lists all 74 API operations with request/response schemas, parameters, and authentication requirements.

## API Discovery

The root endpoint provides API discovery:

```bash
curl http://localhost:8443/
```

```json
{
  "name": "ldfd",
  "description": "LDF Platform API Server",
  "version": "...",
  "api_versions": ["v1"],
  "endpoints": {
    "health": "/v1/health",
    "version": "/v1/version",
    "api_v1": "/v1/",
    "docs": "/swagger/index.html",
    "auth": {
      "create": "/auth/create",
      "login": "/auth/login",
      "logout": "/auth/logout",
      "refresh": "/auth/refresh",
      "validate": "/auth/validate"
    }
  }
}
```

## First Steps

### 1. Create an Account

```bash
curl -X POST http://localhost:8443/auth/create \
  -H "Content-Type: application/json" \
  -d '{
    "auth": {
      "identity": {
        "methods": ["password"],
        "password": {
          "user": {
            "name": "admin",
            "password": "your-password"
          }
        }
      }
    }
  }'
```

### 2. Log In

```bash
curl -X POST http://localhost:8443/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "auth": {
      "identity": {
        "methods": ["password"],
        "password": {
          "user": {
            "name": "admin",
            "password": "your-password"
          }
        }
      }
    }
  }'
```

The response includes a JWT token in the `X-Subject-Token` header. Use it in subsequent requests:

```bash
export TOKEN="your-jwt-token"
```

### 3. Create a Distribution

```bash
curl -X POST http://localhost:8443/v1/distributions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "my-distro",
    "description": "My custom Linux distribution"
  }'
```

### 4. Browse Components

```bash
curl http://localhost:8443/v1/components
```

### 5. Check Health

```bash
curl http://localhost:8443/v1/health
```

## Next Steps

- [Configuration](configuration.md) -- Customize server settings
- [Deployment](deployment.md) -- Deploy with Docker or systemd
- [Sources](sources.md) -- Set up upstream version tracking
- [Architecture](architecture.md) -- Understand the system design
