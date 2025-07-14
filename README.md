# Linux Distribution Factory
A modern, modular, and user-friendly alternative to YoctoProject and BuildRoot for creating custom Linux distributions.

## Overview
Linux Distribution Factory (LDF) is a collection of Go-based tool that simplifies the process of creating custom Linux distributions.
It provides two interfaces - CLI (ldfctl), API (ldfd) - to accommodate different workflows and use cases.

The API interface is a pure restful API core server, it only provide a programmatic interface which only output json payload.
In order for a user to use the Linux Distribution Factory graphical interface, it needs to use the ldfctl command with one of the user interface option available:
- `ldfctl` without arguments use the pure CLI (Command Line Interface) mode.
- `ldfctl --tui` start the client binary using the TUI (Terminal User Interface) mode.
- `ldfctl --wui` start the client binary using the WebUI (Web User Interface) mode.

## Features

- **Multiple Interfaces**:
  - REST API
  - CLI
  - WebUI
  - Terminal UI
- **Modular Design (Choose your component)**:
  - Kernel version (any upstream kernel.org version)
  - Init system (systemd or OpenRC)
  - Filesystem (ext4, XFS, or Btrfs)
  - Resource management (cgroups or none)
  - Logging (journalctl, syslog, rsyslog, or none)
  - Package manager (DNF, APT, or none)
  - Security framework (SELinux, AppArmor, or none)
  - Bootloader (GRUB2, systemd-boot, or Unified Kernel Image)
- **Multi-Architecture Support**:
  - x86_64
  - AARCH64
- **Board Profiles**: Custom hardware configurations with device tree support
- **Multiple Distributions**: Manage and build multiple distribution configurations

## Quick Start

```bash
# Build the project
task build all

# Start the API server (core)
./bin/ldfd

# Run a client request
./bin/ldfctl create distribution --name "my-distro"

# Start the client using its terminal interface mode
./bin/ldfctl --tui

# Start the client using its web interface mode
./bin/ldfctl --wui
```

## Build

## Requirements:
The ldf project leverage a monorepo structure, consequently we recommend you to use sapling scm client (https://sapling-scm.com/), altough you still can use a classic git client.

The ldf project leverage a taskfile (https://taskfile.dev) based build system approach, consequently having taskfile installed is a mandatory requirement.

TLDR;
- Go 1.24 or higher
- Sapling SCM or Git SCM
- Taskfile (go-task)
- Build essentials (gcc, make, etc.)

## Build from sources:
Building the ldf client from source is only recommended for code contributors or operators willing to test latest edge version or specific a specific patch level.
If you don't want to bother with building binaries, we provide packaged binaries for any OS and platforms.

## Usage

## API server usage:

The REST API runs on port 8443 by default:

```bash
# Start the API server (core)
ldfd

# Create a distribution via API
curl -X POST http://localhost:8443/api/v1/distributions \
  -H "Content-Type: application/json" \
  -d '{"name": "my-distro", "platform": "x86_64"}'
```

### CLI usage
```bash
# Create a new distribution
ldfctl create distribution --name my-distro

# Configure an already existing distribution
ldfctl configure distribution --name my-distro --description my-awesome-distribution

# Create a new distribution release
ldfctl create release --distribution my-distro --version 0.0.1 --platform x86_64

# Configure a specific distribution release components
ldfctl configure release --distribution my-distro --version 0.0.1 --channel alpha --author my-name --kernel 6.7.1 --init systemd --filesystem ext4

# Build a distribution specific release
ldfctl build distribution --name my-distro --release 0.0.1
```

### TUI Usage
The Terminal UI provides an interactive interface:
```bash
# Launch the TUI
ldfctl --tui
```
Navigate using arrow keys, select options with Enter, and follow the on-screen instructions.

## Configuration

API server configuration files are stored in one of the following default location:
- `/etc/ldf/ldfd.yaml`
- `/opt/ldf/ldfd.yaml`
- `~/.config/ldf/ldfd.yaml`

You can override settings using:
- Environment variables
- Configuration files
- Command-line flags

Example configuration:

```yaml
# ~/.config/ldf/ldfd.yaml
api:
  url: localhost
  port: 8443
  cors_enabled: true

logging:
  level: info
  format: json

build:
  workspace: /tmp/ldf-build
  output: /srv/www/ldf
  parallel_jobs: 4
```

Client configuration files are stored in one of the following default location:
- `/etc/ldf/ldfctl.yaml`
- `/opt/ldf/ldfctl.yaml`
- `~/.config/ldf/ldfctl.yaml`

You can override settings using:
- Environment variables
- Configuration files
- Command-line flags

Example configuration:

```yaml
# ~/.config/ldf/ldfctl.yaml
api:
  url: localhost
  port: 8443
  cors_enabled: true

logging:
  level: info
  format: json
```

## Architecture

The project follows a modular architecture:

- **Core**: Business logic for building distributions
- **API**: RESTful API using Gin framework
- **CLI**: Command-line interface using Cobra
- **TUI**: Terminal UI using Bubble Tea
- **WUI**: Web UI using Gin framework

## Contributing
Contributions are welcome! Please read our [Contributing Guide](CONTRIBUTING.md) for details on our code of conduct and the process for submitting pull requests.

## License
This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments
- Inspired by YoctoProject and BuildRoot
- Built with amazing Go libraries: Cobra, Gin, Bubble Tea, and more

## Documentation
For detailed documentation, visit our [docs](docs/) directory:

- [Architecture Overview](docs/ARCHITECTURE.md)
- [API Reference](docs/API.md)
- [Getting Started Guide](docs/guides/getting-started.md)
- [Custom Board Profiles](docs/guides/custom-board.md)
