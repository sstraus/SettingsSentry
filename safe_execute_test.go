package main

import (
	"errors"
	"testing"
)

func TestSafeExecute(t *testing.T) {
	// Save the original printer and restore it after the test
	originalPrinter := printer
	defer func() { printer = originalPrinter }()

	// Create a test printer
	testPrinter := NewPrinter("TestApp")
	printer = testPrinter

	// Test case 1: Function executes successfully
	err := safeExecute("successful operation", func() error {
		return nil
	})
	if err != nil {
		t.Errorf("Expected no error for successful operation, got: %v", err)
	}

	// Test case 2: Function returns an error
	expectedErr := errors.New("test error")
	err = safeExecute("error operation", func() error {
		return expectedErr
	})
	if err != expectedErr {
		t.Errorf("Expected error %v, got: %v", expectedErr, err)
	}

	// Test case 3: Function panics
	// We can't easily capture the output from a panic recovery in this test,
	// but we can at least verify that the function recovers from the panic
	err = safeExecute("panic operation", func() error {
		panic("test panic")
	})
	if err != nil {
		t.Errorf("Expected no error after panic recovery, got: %v", err)
	}
}
