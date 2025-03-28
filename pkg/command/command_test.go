package command

import (
	// Keep logger import if needed elsewhere, otherwise remove
	"SettingsSentry/pkg/printer"
	"SettingsSentry/pkg/testutil" // Added testutil
	"errors"
	"testing"
)

func setupTestDependencies() {
	// Use the shared helper, passing nil for FS and CmdExecutor as they aren't needed here
	testLogger := testutil.SetupTestGlobals(nil, nil)

	// Initialize package-specific dependencies
	testPrinter := printer.NewPrinter("Test", testLogger)
	Printer = testPrinter
	// printer.AppLogger is set via util.InitGlobals inside SetupTestGlobals
}

func TestSafeExecute(t *testing.T) {
	setupTestDependencies()

	err := SafeExecute("successful operation", func() error {
		return nil
	})
	if err != nil {
		t.Errorf("Expected no error for successful operation, got: %v", err)
	}

	expectedErr := errors.New("test error")
	err = SafeExecute("error operation", func() error {
		return expectedErr
	})
	if err != expectedErr {
		t.Errorf("Expected error '%v', got: %v", expectedErr, err)
	}

	// We expect SafeExecute to recover and potentially log, but not crash the test
	// We also don't expect an error return value from the panic itself
	err = SafeExecute("panic operation", func() error {
		panic("test panic")
	})
	if err != nil {
		t.Errorf("Expected no error after panic recovery, got: %v", err)
	}

}
