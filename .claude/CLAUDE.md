# SettingsSentry Project Instructions

## Project Context

SettingsSentry is a security-focused macOS backup tool for application configurations. Security is a PRIMARY concern - this tool handles user data and executes commands with full privileges.

## Security-First Development

- **Default to secure**: Dangerous features (command execution) MUST be disabled by default
- **Explicit opt-in**: Security-sensitive features require explicit `--allow-commands` style flags
- **Clear warnings**: Any security risk MUST be documented in help text, README, and runtime warnings
- **Path validation**: ALWAYS sanitize and validate file paths to prevent traversal attacks
- **No PATH reliance**: Use `os.Executable()` instead of `exec.LookPath()` to prevent binary substitution
- **Test security**: Every security fix MUST have tests validating both the vulnerability and the fix

## Release Process

### Version Bumping Rules
- **Patch (x.x.X)**: Bug fixes, new config files, documentation
- **Minor (x.X.0)**: New features, non-breaking changes
- **Major (X.0.0)**: Breaking changes (flag renames, API changes, behavior changes)

### Full Release Workflow
```bash
# 1. Update version in main.go
# 2. Update CHANGELOG.md with release notes
# 3. Run full test suite
make test
# 4. Commit changes
git add -A
git commit -m "release: SettingsSentry vX.Y.Z - description"
# 5. Create annotated tag
git tag -a vX.Y.Z -m "Release notes"
# 6. Push everything
git push && git push --tags
# 7. Build and upload artifacts
goreleaser release --clean
```

## Testing Requirements

- **ALWAYS run full test suite** before committing: `go test ./...`
- **All packages must pass**: Currently 10/10 packages
- **Security tests required**: Every security fix needs before/after validation tests
- **No test skipping**: If tests fail, fix them - never skip or delete
- **Integration tests**: Use real data and real APIs, no mocking in e2e tests

## Build System

### Makefile Commands
- `make build` - Builds binary (handles TestCommand.cfg exclusion automatically)
- `make test` - Runs all tests with race detection
- `make lint` - Runs golangci-lint
- `make release` - Builds with goreleaser (snapshot mode)

### goreleaser
- Builds for darwin/amd64 and darwin/arm64
- Packages binary with all config files
- Uploads to GitHub releases automatically
- Generates checksums for verification

### Config File Management
- **TestCommand.cfg exclusion**: This file is for tests only, NOT embedded in releases
- Build process temporarily renames it to `.not_embed` during build
- All other .cfg files in configs/ ARE embedded in the binary

## Issue Tracking with bd (beads)

We use bd (beads) for dependency-aware issue tracking:
```bash
bd init                           # Initialize (already done)
bd issue "Title" "Description"    # Create issue (returns ID like SettingsSentry-xxx)
bd depend <parent> <child>        # Mark dependency between issues
bd close <issue-id>               # Close completed issue
bd list                           # List all open issues
```

Benefits:
- Issues have semantic IDs (SettingsSentry-xxx) for easy reference in commits
- Dependency tracking ensures we fix things in correct order
- Integrated with git - issues stored in .beads/

## Flag Design Patterns

### Security-Sensitive Flags
- Use clear, explicit names: `--allow-commands` NOT `--commands`
- Boolean flags that enable dangerous features should include "allow" or "enable"
- ALWAYS include security warnings in help text
- Default to false (secure default)

### Environment Variables
All flags should have corresponding env vars:
- Flag: `--allow-commands` → Env: `SETTINGSSENTRY_COMMANDS`
- Flag: `--backup` → Env: `SETTINGSSENTRY_BACKUP`
- Document env vars in help text: `(env: SETTINGSSENTRY_XXX)`

## Code Patterns

### Path Handling
```go
// ALWAYS sanitize paths
resolved := filepath.Clean(path)

// ALWAYS validate no traversal
relPath, err := filepath.Rel(baseDir, resolved)
if err != nil || strings.HasPrefix(relPath, "..") {
    // Path escapes base directory - reject or use safe fallback
}
```

### Goroutine Synchronization
```go
// ALWAYS use WaitGroup for concurrent operations
var wg sync.WaitGroup
wg.Add(n)
go func() {
    defer wg.Done()
    // work
}()
wg.Wait() // Ensure completion before proceeding
```

### Command Execution
```go
// Security: Use absolute paths
exePath, err := os.Executable()
exePath, err = filepath.EvalSymlinks(exePath) // Resolve symlinks

// NOT this (vulnerable to PATH manipulation):
// exePath, err := exec.LookPath("settingssentry")
```

## Documentation Standards

### CHANGELOG.md
- Organized by version with ISO dates (YYYY-MM-DD)
- Sections: BREAKING CHANGES, Security Improvements, Features, Bug Fixes, Configuration Files, Development
- Security fixes include issue IDs: (SettingsSentry-xxx)
- Breaking changes explain impact and migration path

### README.md
- Security warnings MUST be prominent and clear
- All flags documented with security implications
- Examples show secure usage by default

### Commit Messages
```
type: brief description (50 chars max)

Detailed explanation of changes:
- What changed
- Why it changed
- Security implications if any

Issue references: SettingsSentry-xxx
```

Types: release, security, feat, fix, refactor, test, docs, chore

## Common Pitfalls

1. **Don't guess technical details**: If unsure about a security implication, research it or state uncertainty
2. **Breaking changes need major version bump**: Even "small" API changes like flag renames
3. **Test before commit**: ALWAYS run `make test` - no exceptions
4. **goreleaser needs clean git**: Commit everything before running goreleaser
5. **Config files get embedded**: Remember TestCommand.cfg exclusion pattern in Makefile
6. **Security defaults matter**: One insecure default can compromise the entire tool

## Project Values

1. **Security over convenience**: If a feature makes the tool less secure, it needs explicit opt-in
2. **Transparency**: Users must understand what the tool does, especially with commands and privileges
3. **Compatibility**: macOS Sonoma+ support is critical (why this tool exists vs Mackup)
4. **Reliability**: Users trust us with their configurations - don't break that trust
5. **Simplicity**: Clear flags, clear behavior, clear documentation

## Current State (v1.2.0)

- **Test Coverage**: 65.5% across core packages
- **Security Posture**: All known vulnerabilities fixed, TDD approach for new features
- **Platform Support**: macOS (darwin/amd64, darwin/arm64)
- **Issue Tracker**: bd (beads) initialized in .beads/
- **Config Files**: 116 application configs embedded (excluding TestCommand.cfg)
