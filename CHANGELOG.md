# Changelog

All notable changes to SettingsSentry will be documented in this file.

## [Unreleased]

## SettingsSentry v1.2.0 - 2026-01-08

### BREAKING CHANGES
- **Flag renamed**: `--commands` → `--allow-commands` for clarity
  - The old confusing help text "Prevent pre-backup/restore commands execution" has been replaced with clear security warning
  - Migration: If you use `--commands` or `SETTINGSSENTRY_COMMANDS=true`, update scripts to use `--allow-commands`
  - Reason: The old flag name was misleading - it actually ENABLES command execution (security risk), not prevents it

### Security Improvements
- **Enhanced command execution security** (SettingsSentry-7tl)
  - Commands are now DISABLED BY DEFAULT for security
  - Added prominent security warnings in help text and README
  - Documented that commands execute with full user privileges
  - Added security tests to verify commands are blocked when flag is false
- **Fixed Path Traversal in config file resolution** (SettingsSentry-50h)
  - Added path sanitization using `filepath.Clean()` and `filepath.Rel()` in `ResolveConfigFilePath`
  - Validates that resolved paths stay within home directory
  - Prevents attackers from accessing files outside intended directories (e.g., `~/../../etc/passwd`)
  - Returns safe fallback path when traversal attempts detected
  - Logs warnings for path traversal attempts
- **Fixed Path Traversal in backup path construction** (SettingsSentry-1z3)
  - Added `sanitizeConfigName()` function to prevent directory traversal in config names
  - Removes path separators and `../` sequences from config names
  - Prevents malicious configs (e.g., `name=../../../malicious`) from writing backups outside backup directory
  - Config names with traversal sequences are replaced with safe defaults
  - Logs warnings when config names are sanitized
- **Fixed Cron Job LookPath security vulnerability** (SettingsSentry-b7k)
  - Replaced `exec.LookPath()` with `os.Executable()` in cron job installation
  - Prevents PATH manipulation attacks where attacker could substitute malicious binary
  - Uses absolute path of currently running binary instead of searching PATH
  - Resolves symlinks with `filepath.EvalSymlinks()` for additional security
  - Added test to verify cron jobs use absolute paths
- **Reviewed and verified Zip Slip protection** (SettingsSentry-bzs)
  - Confirmed existing protection properly validates all extraction paths
  - Uses `filepath.Clean()` and `filepath.Abs()` to prevent directory traversal
  - Validates extracted paths stay within destination directory
  - Comprehensive tests verify protection against path traversal and symlink attacks
  - All Zip Slip security tests pass

### Features
- **Cron job installation now supports `--allow-commands` flag**
  - Use `settingssentry install --allow-commands` to enable command execution in scheduled backups
  - Commands are disabled by default in cron jobs for security
  - Clear warning displayed when installing with commands enabled
  - Added comprehensive tests for flag functionality

### Bug Fixes
- **Fixed Goroutine Resource Leak** (SettingsSentry-ijc)
  - Added WaitGroup synchronization to `ExecuteWithCallback` in `interfaces/command.go`
  - Prevents race conditions between goroutine completion and cmd.Wait()
  - Ensures all stdout/stderr output is captured before function returns
  - Eliminates potential resource leaks from unsynchronized goroutines

### Configuration Files
- Added 27 new application configuration files from Mackup database:
  - aldente.cfg, blesh.cfg, claude-code.cfg, codex.cfg, factory-droid.cfg
  - github-cli.cfg, gmailctl.cfg, gnu-stow.cfg, kiro.cfg, lazydocker.cfg
  - leiningen.cfg, lightpaper.cfg, mise.cfg, mole.cfg, offlineimap.cfg
  - opencode.cfg, opera.cfg, plover.cfg, rustrover.cfg, shadowsocksx-ng.cfg
  - terraform.cfg, things.cfg, tidy.cfg, vimwiki.cfg, windsurf.cfg
  - yazi.cfg, youtube-dl.cfg

### Development
- Initialized bd (beads) issue tracking for dependency-aware workflow
- Added .claude directory to .gitignore

## SettingsSentry v1.1.8 - 2025-11-21

Enhanced testing, security, and code quality improvements.

### Security Fixes
- Fixed **Zip Slip vulnerability** (CVE) in zip extraction - prevents directory traversal attacks
- Updated `golang.org/x/crypto` to v0.45.0 (fixes SSH memory consumption and panic vulnerabilities)

### New Features
- Added `make act-test` - run GitHub Actions CI tests locally
- Added `make act-lint` - run linter locally with GitHub Actions
- Added comprehensive CLI tests (803 lines of new tests)
- Added backup operations test suite (1,640 lines of tests)

### Improvements
- Improved test coverage to **65.5%** across core packages
- Fixed all 33 golangci-lint issues (errcheck violations)
- Fixed shell compatibility issues (bash vs sh)
- Refactored main package for better testability
- Separated backup operations into dedicated module
- Added `.actrc` configuration for local CI testing

### Testing
- ✅ All tests pass: `go test ./...`
- ✅ All linting passes: `make lint` (0 issues)
- ✅ Local CI testing with `act` works on macOS (ARM & Intel)

### Files Changed
- 24 files changed: +6,901 lines, -861 lines
- New test fixtures and test data added

---

## SettingsSentry v1.1.7 - 2024

The tool has reached maturity.

### New Features
- Versioned backups with timestamp-based directories
- Dry-run mode to preview operations without making changes
- Self-contained config files (you can extract them to customize)
- Optional ZIP archive backup format (`-zip` flag)
- Optional password-based encryption (`-password` flag)

### Core Features
- Cross-platform configuration backup and restore
- Support for multiple applications
- Pre/post backup and restore commands
- Cron job scheduling for automated backups
- iCloud Drive integration for backup storage
- Environment variable configuration support

### Supported Platforms
- macOS
- Linux
- Unix-like systems with cron support
