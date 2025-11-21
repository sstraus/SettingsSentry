package main

import (
	"testing"
)

// Note: Direct testing of run() function is avoided due to flag.Parse() conflicts
// in the test environment (flags can only be defined once globally).
// The run() function logic is thoroughly covered through:
// - main_cli_test.go: Tests CLI struct methods (ParseFlags, ExecuteAction, ShowHelp)
// - integration_test.go: Tests full backup/restore workflows
// - Individual package tests: Test all underlying functionality

// TestMain exists to satisfy go test requirements for this package
func TestMain(m *testing.M) {
	// The main() function itself cannot be easily unit tested due to os.Exit()
	// and flag parsing constraints. Coverage for main logic is achieved through
	// testing the run() function's components and integration tests.
	m.Run()
}