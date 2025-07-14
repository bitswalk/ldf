# Linux Distribution Factory - Project Structure

## Overview

This document outlines the proposed directory structure for the Linux Distribution Factory project, a modern alternative to YoctoProject and BuildRoot. The project is built with Go and provides CLI, API, and TUI interfaces for creating custom Linux distributions.

## Directory Tree

```shell
linux-distribution-factory/
├── src/cmd/
│   ├── ldf/                    # Main CLI application
│   │   └── main.go
│   ├── ldf-api/                # API server application
│   │   └── main.go
│   └── ldf-tui/                # TUI application
│       └── main.go
├── src/internal/
│   ├── core/                   # Core business logic
│   │   ├── builder/            # Distribution build orchestration
│   │   │   ├── builder.go
│   │   │   ├── stages.go
│   │   │   └── validation.go
│   │   ├── kernel/             # Kernel compilation logic
│   │   │   ├── compiler.go
│   │   │   ├── config.go
│   │   │   └── upstream.go
│   │   ├── components/         # System components
│   │   │   ├── bootloader/
│   │   │   │   ├── grub2.go
│   │   │   │   ├── systemd_boot.go
│   │   │   │   └── unified_kernel.go
│   │   │   ├── filesystem/
│   │   │   │   ├── ext4.go
│   │   │   │   ├── xfs.go
│   │   │   │   └── btrfs.go
│   │   │   ├── init/
│   │   │   │   ├── systemd.go
│   │   │   │   └── openrc.go
│   │   │   ├── journal/
│   │   │   │   ├── journalctl.go
│   │   │   │   ├── syslog.go
│   │   │   │   └── rsyslog.go
│   │   │   ├── package/
│   │   │   │   ├── dnf.go
│   │   │   │   └── apt.go
│   │   │   └── security/
│   │   │       ├── selinux.go
│   │   │       └── apparmor.go
│   │   ├── platform/           # Platform-specific code
│   │   │   ├── aarch64/
│   │   │   │   └── platform.go
│   │   │   ├── x86_64/
│   │   │   │   └── platform.go
│   │   │   └── common.go
│   │   ├── board/              # Board profiles
│   │   │   ├── profile.go
│   │   │   ├── devicetree.go
│   │   │   └── drivers.go
│   │   └── distribution/       # Distribution management
│   │       ├── distribution.go
│   │       ├── repository.go
│   │       └── metadata.go
│   ├── api/                    # REST API implementation
│   │   ├── handlers/
│   │   │   ├── distribution.go
│   │   │   ├── build.go
│   │   │   ├── kernel.go
│   │   │   ├── board.go
│   │   │   └── health.go
│   │   ├── middleware/
│   │   │   ├── auth.go
│   │   │   ├── cors.go
│   │   │   └── logging.go
│   │   ├── models/
│   │   │   ├── distribution.go
│   │   │   ├── build.go
│   │   │   └── response.go
│   │   ├── routes/
│   │   │   └── routes.go
│   │   └── server.go
│   ├── tui/                    # TUI implementation
│   │   ├── models/
│   │   │   ├── distribution.go
│   │   │   └── navigation.go
│   │   ├── views/
│   │   │   ├── main.go
│   │   │   ├── distribution_list.go
│   │   │   ├── distribution_create.go
│   │   │   ├── build_config.go
│   │   │   └── build_progress.go
│   │   ├── components/
│   │   │   ├── header.go
│   │   │   ├── footer.go
│   │   │   ├── list.go
│   │   │   └── form.go
│   │   └── app.go
│   ├── cli/                    # CLI commands implementation
│   │   ├── distribution/
│   │   │   ├── create.go
│   │   │   ├── list.go
│   │   │   ├── delete.go
│   │   │   └── build.go
│   │   ├── board/
│   │   │   ├── create.go
│   │   │   └── list.go
│   │   ├── kernel/
│   │   │   └── list.go
│   │   └── root.go
│   ├── log/                    # Logging module
│   │   ├── logger.go
│   │   ├── formatter.go
│   │   └── levels.go
│   └── config/                 # Configuration management
│       ├── config.go
│       ├── defaults.go
│       └── validation.go
├── src/pkg/                        # Public packages
│   ├── types/                  # Shared types
│   │   ├── distribution.go
│   │   ├── platform.go
│   │   └── component.go
│   └── utils/                  # Shared utilities
│       ├── download.go
│       ├── checksum.go
│       └── compression.go
├── src/api/                        # OpenAPI specifications
│   └── openapi/
│       ├── openapi.yaml
│       └── schemas/
│           ├── distribution.yaml
│           ├── build.yaml
│           └── board.yaml
├── src/configs/                    # Configuration files
│   ├── default.yaml
│   └── examples/
│       ├── minimal.yaml
│       └── full-featured.yaml
├── scripts/                    # Build and utility scripts
│   ├── build.sh
│   ├── test.sh
│   └── generate-api.sh
├── src/templates/                  # Distribution templates
│   ├── kernel/
│   │   └── config.template
│   ├── init/
│   │   ├── systemd/
│   │   └── openrc/
│   └── bootloader/
│       ├── grub/
│       └── systemd-boot/
├── src/data/                       # Static data
│   ├── boards/                 # Board profiles
│   │   ├── raspberry-pi-4.yaml
│   │   └── generic-x86_64.yaml
│   └── patches/                # Kernel patches
├── build/                      # Build artifacts (gitignored)
│   ├── workspace/              # Temporary build workspace
│   └── output/                 # Final distribution images
├── docs/                       # Documentation
│   ├── README.md
│   ├── ARCHITECTURE.md
│   ├── API.md
│   └── guides/
│       ├── getting-started.md
│       └── custom-board.md
├── src/test/                       # Test files
│   ├── unit/
│   ├── integration/
│   └── fixtures/
├── .github/                    # GitHub specific files
│   └── workflows/
│       ├── ci.yml
│       └── release.yml
├── go.mod
├── go.sum
├── Makefile
├── Dockerfile
├── docker-compose.yml
├── .gitignore
├── .golangci.yml               # Go linter configuration
└── README.md
```
## Directory Descriptions

### Root Level
- **build/** - Build artifacts and workspace (gitignored)
- **docs/** - Project documentation
- **tools/scripts/** - Build and utility scripts
- **src/cmd/** - Entry points for the three main applications (CLI, API, TUI)
- **src/internal/** - Private application code (not importable by external projects)
- **src/pkg/** - Public packages that could be imported by other Go projects
- **src/api/** - OpenAPI 3.1.1 specifications
- **src/configs/** - Configuration files and examples
- **src/templates/** - Templates for various distribution components
- **src/data/** - Static data files (board profiles, patches)
- **src/test/** - Test files and fixtures

### Internal Structure

#### Core (`internal/core/`)
The heart of the application containing:
- **builder/** - Distribution build orchestration logic
- **kernel/** - Kernel compilation and configuration
- **components/** - System components (bootloader, filesystem, init, etc.)
- **platform/** - Platform-specific implementations (AARCH64, X86_64)
- **board/** - Board profile management
- **distribution/** - Distribution lifecycle management

#### API (`internal/api/`)
REST API implementation using Gin framework:
- **handlers/** - HTTP request handlers
- **middleware/** - HTTP middleware (auth, CORS, logging)
- **models/** - API data models
- **routes/** - Route definitions

#### TUI (`internal/tui/`)
Terminal UI implementation using Bubble Tea:
- **models/** - TUI data models
- **views/** - Different screens/views
- **components/** - Reusable UI components

#### CLI (`internal/cli/`)
Command-line interface using Cobra:
- Command implementations organized by domain (distribution, board, kernel)

#### Supporting Modules
- **log/** - Centralized logging using charmbracelet/log
- **config/** - Configuration management using spf13/viper

## Key Design Principles

1. **Separation of Concerns** - Clear boundaries between core logic, presentation layers (API/TUI/CLI), and infrastructure concerns
2. **Modularity** - Each system component is independently implementable and testable
3. **Go Best Practices** - Following standard Go project layout conventions
4. **API-First** - OpenAPI specifications drive API development
5. **Testability** - Structure supports comprehensive unit and integration testing
6. **Extensibility** - Easy to add new platforms, components, or board profiles

## Technologies Used

- **Language**: Go
- **CLI Framework**: spf13/cobra
- **Configuration**: spf13/viper
- **API Framework**: gin-gonic/gin (OpenAPI 3.1.1)
- **TUI Framework**: charmbracelet/bubbletea
- **Logging**: charmbracelet/log
