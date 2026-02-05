# Linux Distribution Factory

Linux Distribution Factory (LDF) is a modern, modular platform for creating and managing custom Linux distributions. It provides a REST API server, a web interface, and a CLI client to accommodate different workflows.

## Components

| Component | Description | Status |
|-----------|-------------|--------|
| **ldfd** | Core REST API server (Go, Gin) | Production-ready |
| **WebUI** | Web interface (SolidJS, TailwindCSS) | Functional |
| **ldfctl** | CLI client (Go, Cobra) | Planned |

## Key Features

- **Distribution management** -- Create, configure, and track custom Linux distributions with full component selection
- **Component catalog** -- Kernel versions, init systems, filesystems, bootloaders, security frameworks, and more
- **Source tracking** -- Automatic upstream version discovery from GitHub, GitLab, Gitea, Codeberg, Forgejo, and HTTP directories
- **Artifact storage** -- Local filesystem or S3-compatible backends (AWS, MinIO, GarageHQ)
- **Download management** -- Worker pool with retry, verification, and progress tracking
- **Authentication** -- JWT-based auth with role-based access control
- **API documentation** -- Interactive Swagger UI at `/swagger/index.html`

## Quick Links

- [Getting Started](getting-started.md) -- Build and run LDF from source
- [Architecture](architecture.md) -- System design and component overview
- [Configuration](configuration.md) -- All server settings and config file reference
- [Deployment](deployment.md) -- Docker, systemd, and bare metal deployment
- [Sources](sources.md) -- Source management and forge integration
