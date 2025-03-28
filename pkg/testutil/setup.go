package testutil

import (
	"SettingsSentry/interfaces"
	"SettingsSentry/logger"
	"SettingsSentry/pkg/util"
)

// SetupTestGlobals initializes common global dependencies for testing.
// It accepts specific FileSystem and CommandExecutor implementations
// needed for the test context and returns the initialized logger.
// It sets DryRun to false by default.
func SetupTestGlobals(fs interfaces.FileSystem, cmdExec interfaces.CommandExecutor) *logger.Logger {
	testLogger, _ := logger.NewLogger("") // Assuming no log file for tests
	util.InitGlobals(testLogger, fs, cmdExec, false)
	return testLogger
}
