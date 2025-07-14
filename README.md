# Linux Distribution Factory

A modern, modular, and user-friendly alternative to YoctoProject and BuildRoot for creating custom Linux distributions.

## Overview

Linux Distribution Factory (LDF) is a collection of Go-based tool that simplifies the process of creating custom Linux distributions. It provides two interfaces - CLI (ldfctl), API (ldfd) - to accommodate different workflows and use cases.

The API interface is a pure restful API core server, it only provide a programmatic interface which only output json payload.
In order for a user to use the Linux Distribution Factory graphical interface, it needs to use the ldfctl command with one of the user interface option available:
- `ldfctl` without arguments use the pure CLI (Command Line Interface) mode.
- `ldfctl --tui` start the client binary using the TUI (Terminal User Interface) mode.
- `ldfctl --gui` start the client binary using the GUI (Graphical User Interface) mode.

## Features

- **Multiple Interfaces**: CLI, REST API, and Terminal UI
- **Modular Design**: Choose your components:
  - Kernel version (any upstream kernel.org version)
  - Init system (systemd or OpenRC)
  - Filesystem (ext4, XFS, or Btrfs)
  - Resource management (cgroups or none)
  - Logging (journalctl, syslog, rsyslog, or none)
  - Package manager (DNF, APT, or none)
  - Security framework (SELinux, AppArmor, or none)
  - Bootloader (GRUB2, systemd-boot, or Unified Kernel Image)
- **Multi-Architecture Support**: x86_64 and AARCH64
- **Board Profiles**: Custom hardware configurations with device tree support
- **Multiple Distributions**: Manage and build multiple distribution configurations

## Quick Start

```bash
# Build the project
make build

# Run the CLI
./bin/ldf --help

# Start the API server
./bin/ldf-api

# Launch the TUI
./bin/ldf-tui
```

## Installation

### Prerequisites

- Go 1.21 or higher
- Git
- Build essentials (gcc, make, etc.)

### Building from Source

```bash
git clone https://github.com/yourusername/linux-distribution-factory.git
cd linux-distribution-factory
make install
```

## Usage

### CLI Usage

```bash
# Create a new distribution
ldfctl create distribution --name my-distro --platform x86_64

# Create a new distribution release
ldfctl create release --distribution my-distro --version 0.0.1

# Configure components
ldfctl configure distribution --name my-distro --release 0.0.1 --description my-awesome-distribution

# Configure release
ldfctl configure release --distribution my-distro --version 0.0.1 --channel alpha --author my-name --kernel 6.7.1 --init systemd --filesystem ext4

# Build the distribution
ldfctl build distribution --name my-distro --release 0.0.1
```

### API Usage

The REST API runs on port 8080 by default:

```bash
# Start the API server
ldf-api

# Create a distribution via API
curl -X POST http://localhost:8080/api/v1/distributions \
  -H "Content-Type: application/json" \
  -d '{"name": "my-distro", "platform": "x86_64"}'
```

### TUI Usage

The Terminal UI provides an interactive interface:

```bash
# Launch the TUI
ldf-tui
```

Navigate using arrow keys, select options with Enter, and follow the on-screen instructions.

## Configuration

Configuration files are stored in `~/.config/ldf/` by default. You can override settings using:

- Environment variables
- Configuration files
- Command-line flags

Example configuration:

```yaml
# ~/.config/ldf/config.yaml
build:
  workspace: /tmp/ldf-build
  output: ~/ldf-distributions
  parallel_jobs: 4

api:
  port: 8080
  cors_enabled: true

logging:
  level: info
  format: json
```

## Architecture

The project follows a modular architecture:

- **Core**: Business logic for building distributions
- **API**: RESTful API using Gin framework
- **TUI**: Terminal UI using Bubble Tea
- **CLI**: Command-line interface using Cobra

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
