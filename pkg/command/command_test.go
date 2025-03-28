package command

import (
	"SettingsSentry/logger"
	"SettingsSentry/pkg/printer"
	"SettingsSentry/pkg/util"
	"errors"
	"testing"
)

func setupTestDependencies() {
	testLogger, _ := logger.NewLogger("")
	testPrinter := printer.NewPrinter("Test", testLogger)
	util.InitGlobals(testLogger, nil, nil, false, "")
	printer.AppLogger = testLogger
	Printer = testPrinter
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
