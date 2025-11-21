# Changelog

All notable changes to SettingsSentry will be documented in this file.

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
