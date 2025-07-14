# Linux Distribution Factory Client

A modern, modular, client for the LDF (Linux Distribution Factory) project.

## Overview

Linux Distribution Factory Client (ldfctl) is a Go-based tool that provide operators with a nice and sleak human ready interfaces options.
A Linux Distribution Factory (ldf) operator can choose to use one of the three available ldfctl command interface mode (GUI/TUI/CLI).

## Build

Building the ldf client from source is only recommended for code contributors or operators willing to test latest edge version or specific a specific patch level.
If you don't want to bother with building binaries, we provide packaged binaries for any OS and platforms.

### Requirements:
The ldf project leverage a monorepo structure, consequently we recommend you to use sapling scm client (https://sapling-scm.com/), altough you still can use a classic git client.

The ldf project leverage a taskfile (https://taskfile.dev) based build system approach, consequently having taskfile installed is a mandatory requirement.

### Build from sources:
Building the ldf client is easy, once you've installed your taskfile client, just type:
`task build client`
The build system will then create the appropriate ldfctl binary for your platform within the `ldf/build/bin/` folder.

If you want to build a binary for a specific platform and architecture juste type:
`task build client <arch> <platform>` replace `<arch>` and `<platform>` with the appropriate supported values.
The build system will then create the appropriate ldfctl binary for the targeted platform within the `ldf/build/bin/` folder.

## Usage
In order for an operator to use the Linux Distribution Factory graphical interface, it needs the ldfctl binary and launch it with one of the user interface option available:
- `ldfctl` without arguments use the pure CLI (Command Line Interface) mode.
- `ldfctl --tui` start the client binary using the TUI (Terminal User Interface) mode.
- `ldfctl --gui` start the client binary using the GUI (Graphical User Interface) mode.

The ldfctl command will try its best to detect any available Linux Distribution Factory Server (ldfd) locally, however, it can't do magic and it needs at least a endpoint url if you deployed it on a remote infrastructure.

To that purpose, as ldfctl leverage golang Viper configuration management library and golang Cobra command line management library you can either put that information on the client configuration file or as an argument.

By default, the ldfctl client will look for the following configuration file location:
- `/etc/ldf/config.yaml`
- `/opt/ldf/config.yaml`
- `~/.config/ldf/config.yaml`

By default, the ldfctl client when not able to found any configuration file will fallback on its internal sane defaults and override them with any user provided flags.

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

## Architecture

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
