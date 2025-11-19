# Development

## Dependency Management

SettingsSentry uses Go modules for dependency management. The project dependencies are defined in the `go.mod` file and are vendored for reproducible builds.

To update dependencies to their latest versions:

```bash
go get -u ./...
go mod tidy
go mod vendor
```

To build using vendored dependencies:

```bash
go build -mod=vendor
```

### Local Development

A Makefile is provided for common development tasks:

```bash
# Build the application
make build

# Run tests
make test

# Run linter
make lint

# Clean build artifacts
make clean

# Create a release (requires GoReleaser)
make release
```

## Building SettingsSentry

This document provides instructions for building, testing, and developing SettingsSentry.

## Prerequisites

- macOS (10.15 Catalina or newer)
- Go 1.16 or newer
- Homebrew (for installing dependencies)

## Setup Development Environment

1. Install Go:

   ```bash
   brew install go
   ```

2. Install development tools:

   ```bash
   brew install golangci-lint goreleaser
   ```

## Building

### Basic Build

To build the application:

```bash
make build
```

Or manually:

```bash
go build -o settingssentry
```

### Creating a Release Package

To create a ZIP archive with the executable and configuration files:

```bash
make zip
```

Or manually:

```bash
./build.sh
```

### Creating a DMG Installer

To create a macOS DMG installer:

```bash
make dmg
```

## Testing

SettingsSentry includes a comprehensive test suite. Here's how to run the tests:

### Running Unit Tests

To run all unit tests:

```bash
make test
```

Or manually:

```bash
go test ./...
```

### Running Specific Tests

To run tests in a specific package:

```bash
go test ./cron
```

To run a specific test:

```bash
go test -run TestParseConfig
```

### Running Integration Tests

Integration tests are tagged and can be run with:

```bash
go test -tags=integration
```

### Test Coverage

To generate a test coverage report:

```bash
make coverage
```

Or manually:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

#### Current Coverage Statistics

**Overall Coverage: 46.3%** (after refactoring for testability)

Coverage by package:
- `logger`: 93.8% ✅
- `cron`: 89.9% ✅
- `interfaces`: 84.6% ✅
- `command`: 83.9% ✅
- `printer`: 80.0% ✅
- `config`: 73.7% 🟡
- `util`: 55.6% 🟡
- `backup`: 49.9% 🟡
- `main`: 0.0% 🔴 (refactored for testability, tests to be added)

**Note**: The main package was recently refactored to improve testability by:
- Extracting CLI logic into `main_cli.go` with testable methods
- Converting `main()` to use `run()` function that returns errors
- Creating `BackupContext` in `pkg/backup/backup_operations.go` for dependency injection

These refactorings enable comprehensive testing and are targeted to achieve 80%+ coverage in the next phase.

## Linting

To run the linter:

```bash
make lint
```

Or manually:

```bash
golangci-lint run
```

### CI/CD Pipeline

SettingsSentry uses GitHub Actions for continuous integration and deployment, specifically designed for macOS:

- **CI Workflow**: Runs on every push to main and pull requests. It includes:
  - Linting with golangci-lint
  - Running tests with race detection
  - Building the application

- **Release Workflow**: Triggered when a new tag is pushed. It:
  - Runs tests
  - Builds macOS binaries for both Intel and Apple Silicon
  - Creates a macOS DMG installer
  - Creates a ZIP archive with the executable and configuration files
  - Publishes GitHub releases with assets

- **Security Scanning**: CodeQL analysis for security vulnerabilities

See the `.github/workflows` directory for the workflow definitions.
