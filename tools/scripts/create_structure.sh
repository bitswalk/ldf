#!/bin/bash

# Create directory structure for Linux Distribution Factory

# cmd directories
mkdir -p src/cmd/ldf
mkdir -p src/cmd/ldf-api
mkdir -p src/cmd/ldf-tui

# internal directories
mkdir -p src/internal/core/builder
mkdir -p src/internal/core/kernel
mkdir -p src/internal/core/components/bootloader
mkdir -p src/internal/core/components/filesystem
mkdir -p src/internal/core/components/init
mkdir -p src/internal/core/components/journal
mkdir -p src/internal/core/components/package
mkdir -p src/internal/core/components/security
mkdir -p src/internal/core/platform/aarch64
mkdir -p src/internal/core/platform/x86_64
mkdir -p src/internal/core/board
mkdir -p src/internal/core/distribution

# API directories
mkdir -p src/internal/api/handlers
mkdir -p src/internal/api/middleware
mkdir -p src/internal/api/models
mkdir -p src/internal/api/routes

# TUI directories
mkdir -p src/internal/tui/models
mkdir -p src/internal/tui/views
mkdir -p src/internal/tui/components

# CLI directories
mkdir -p src/internal/cli/distribution
mkdir -p src/internal/cli/board
mkdir -p src/internal/cli/kernel

# Supporting modules
mkdir -p src/internal/log
mkdir -p src/internal/config

# Public packages
mkdir -p src/pkg/types
mkdir -p src/pkg/utils

# API specifications
mkdir -p src/api/openapi/schemas

# Configuration directories
mkdir -p src/configs/examples

# Docker directory
mkdir -p tools/docker

# Scripts directory
mkdir -p tools/scripts

# Templates directories
mkdir -p src/templates/kernel
mkdir -p src/templates/init/systemd
mkdir -p src/templates/init/openrc
mkdir -p src/templates/bootloader/grub
mkdir -p src/templates/bootloader/systemd-boot

# Data directories
mkdir -p src/data/boards
mkdir -p src/data/patches

# Build directories
mkdir -p build/workspace
mkdir -p build/output

# Documentation directories
mkdir -p docs/guides

# Test directories
mkdir -p src/test/unit
mkdir -p src/test/integration
mkdir -p src/test/fixtures

# GitHub workflows
mkdir -p .github/workflows

echo "Directory structure created successfully!"
