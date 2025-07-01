# pkg/

This directory contains the main package modules for the Bitbucket CLI.

## Structure

- **api/** - Bitbucket API client and HTTP request handling
- **auth/** - Authentication management (app passwords, OAuth, access tokens)
- **cmd/** - Command implementations organized by command group
- **config/** - Configuration management using Koanf
- **output/** - Output formatting (table, JSON, YAML)
- **git/** - Git repository operations and context detection
- **utils/** - Shared utilities and helper functions
- **version/** - Version information and build metadata

## Design Principles

- **Real API Integration**: All modules are designed for real Bitbucket API v2 integration from day one
- **No Mock Data**: Modules should work with actual API responses, not mock data
- **Testable**: Each module should be easily testable with real API integration tests
- **Modular**: Modules should be loosely coupled and independently testable

## Development Guidelines

1. Follow Go package conventions
2. Include comprehensive tests for each module
3. Document public APIs with Go doc comments
4. Use interfaces for dependencies to enable testing
5. Handle errors gracefully with context information