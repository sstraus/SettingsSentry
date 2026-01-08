# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

SettingsSentry is a security-focused macOS backup tool for application configurations. It copies files instead of symlinking (unlike Mackup), ensuring full compatibility with macOS Sonoma+. Security is paramount—this tool handles user data and can execute commands with full privileges.

## Architecture

### Core Flow

```
main.go → main_cli.go (CLI) → pkg/backup/backup_operations.go (BackupContext) → pkg/config/config.go (Config parsing)
                            ↓
                     interfaces/ (FileSystem, CommandExecutor abstractions for testability)
```

### Key Components

**Entry Point (`main.go`, `main_cli.go`)**
- `main.go`: Embeds config files (`//go:embed configs/*.cfg`), initializes logger, filesystem, and CLI
- `CLI` struct: Parses flags, validates actions, orchestrates execution
- Actions: `backup`, `restore`, `configsinit`, `install` (cron), `remove` (cron)

**Backup Operations (`pkg/backup/backup_operations.go`)**
- `BackupContext`: Central struct managing backup/restore lifecycle
- Pipeline: `SetupBackupDirectory()` → `LoadConfigFiles()` → `FilterConfigFiles()` → `BackupFile()`/`RestoreFile()` → `ExecuteCommands()` → `FinalizeBackup()`
- Handles: Versioned backups (timestamp dirs), zip archives, encryption (AES-256-GCM), path resolution with traversal protection

**Config System (`pkg/config/config.go`)**
- Parses `.cfg` files (INI-like format) from `configs/` directory
- `Config` struct: `Name`, `Files[]`, `Pre/PostBackupCommands`, `Pre/PostRestoreCommands`
- `ExpandEnvVars()`: Replaces `${VAR}` in paths, `ValidateConfig()`: Ensures required fields exist
- 116+ embedded application configs (excluding `TestCommand.cfg` used only in tests)

**Interfaces Layer (`interfaces/`)**
- `FileSystem`: Abstracts file operations (`Open`, `ReadDir`, `MkdirAll`, etc.) for testing
- `CommandExecutor`: Abstracts command execution with `Execute()` and `ExecuteWithCallback()` for streaming output
- Implementations: `OsFileSystem`, `OsCommandExecutor` (production), mock versions in tests

**Cron Integration (`cron/cronjob.go`)**
- `InstallCronJob()`: Uses `os.Executable()` (NOT `exec.LookPath()`) to prevent PATH attacks
- Generates crontab entries with `# SettingsSentry cron job` comment for identification
- Supports custom cron expressions or `@reboot` default

## Development Commands

### Building
```bash
make build          # Builds binary, excludes TestCommand.cfg from embed
go build -o settingssentry .  # Direct build (use make to handle test exclusions)
```

### Testing
```bash
make test           # Full suite with race detection
go test ./...       # Run all tests
go test -v ./pkg/backup  # Run specific package tests
go test -run TestBackupFile  # Run specific test
go test -tags=integration ./...  # Integration tests only
```

### Linting & Coverage
```bash
make lint           # golangci-lint (auto-installs if missing)
make coverage       # Generate HTML coverage report
go test -coverprofile=coverage.out ./...
```

### Local CI Testing
```bash
make act-test       # Run GitHub Actions tests locally (requires `act`)
make act-lint       # Run GitHub Actions linter locally
```

### Release
```bash
# 1. Update version in main.go (line 22)
# 2. Update CHANGELOG.md with release notes
# 3. Run tests
make test
# 4. Commit and tag
git commit -m "release: vX.Y.Z - description"
git tag -a vX.Y.Z -m "Release notes"
git push && git push --tags
# 5. Build and upload artifacts (darwin/amd64 + arm64 binaries)
goreleaser release --clean
```

## Security Requirements

**Default-Deny for Dangerous Features**
- Command execution DISABLED by default (`--allow-commands` flag required)
- Use explicit "allow"/"enable" in flag names for security features
- Runtime warnings when security-sensitive features are enabled

**Path Validation (Critical)**
```go
// ALWAYS sanitize and validate paths
resolved := filepath.Clean(path)
relPath, err := filepath.Rel(baseDir, resolved)
if err != nil || strings.HasPrefix(relPath, "..") {
    // Path traversal attempt - reject or use safe fallback
}
```

**Command Execution**
```go
// SECURITY: Use os.Executable() NOT exec.LookPath()
exePath, _ := os.Executable()
exePath, _ = filepath.EvalSymlinks(exePath)  // Resolve symlinks
// Prevents PATH manipulation attacks
```

**Goroutine Management**
```go
// ALWAYS use WaitGroup for concurrent operations
var wg sync.WaitGroup
wg.Add(n)
go func() {
    defer wg.Done()
    // ...
}()
wg.Wait()  // Ensure completion
```

## Testing Standards

- **Security tests required**: Every security fix needs before/after validation
- **No mocking in e2e tests**: Use real filesystem, real commands
- **All 10 packages must pass**: Zero tolerance for failing tests
- **Test naming**: `TestFunctionName_Scenario` (e.g., `TestBackupFile_WithEncryption`)

## Configuration File Format

```ini
[application]
name = AppName

[backup_commands]  # Optional - only executes if --allow-commands is set
some_command --flag

[restore_commands]
restore_command

[configuration_files]
.config/app/settings.json
~/Library/Preferences/com.app.plist
${XDG_CONFIG_HOME}/app  # Environment variables supported
```

**Critical**: `TestCommand.cfg` in `configs/` is NEVER embedded in releases (excluded in Makefile build process).

## Version Bumping

- **Patch (x.x.X)**: Bug fixes, new config files, documentation
- **Minor (x.X.0)**: New features, non-breaking changes
- **Major (X.0.0)**: Breaking changes (flag renames, API changes, default behavior changes)

Example: `--commands` → `--allow-commands` required major version bump (v1.2.0)

## Issue Tracking (bd/beads)

```bash
bd issue "Title" "Description"  # Creates SettingsSentry-xxx issue
bd depend <parent> <child>      # Track dependencies
bd close <issue-id>             # Mark complete
bd list                         # View open issues
```

Reference issue IDs in commits: `security: fix path traversal (SettingsSentry-50h)`

## Documentation

**CHANGELOG.md Format**
```markdown
## SettingsSentry vX.Y.Z - YYYY-MM-DD

### BREAKING CHANGES
- Description with migration path

### Security Improvements
- Fix description (SettingsSentry-xxx)

### Features
- Feature description
```

**Commit Format**
```
type: brief description (50 chars max)

Details:
- What changed
- Why it changed
- Security implications

Issue: SettingsSentry-xxx
```

Types: `release`, `security`, `feat`, `fix`, `refactor`, `test`, `docs`, `chore`

## Current State (v1.2.0)

- **Test Coverage**: 65.5%
- **Platform**: macOS only (darwin/amd64, darwin/arm64)
- **Embedded Configs**: 116 application configs
- **Security**: All known CVEs fixed, TDD approach enforced
