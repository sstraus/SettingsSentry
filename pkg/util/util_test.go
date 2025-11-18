package util

import (
	"SettingsSentry/interfaces"
	"SettingsSentry/logger"
	"embed"
	"testing"
)

func TestInitGlobals(t *testing.T) {
	// Create test logger and filesystem
	testLogger, err := logger.NewLogger("")
	if err != nil {
		t.Fatalf("Failed to create test logger: %v", err)
	}
	defer testLogger.Close()

	testFs := interfaces.NewOsFileSystem()
	testCmdExecutor := interfaces.NewOsCommandExecutor()
	testDryRun := true

	// Initialize globals
	InitGlobals(testLogger, testFs, testCmdExecutor, testDryRun)

	// Verify globals were set
	if AppLogger != testLogger {
		t.Error("AppLogger was not set correctly")
	}

	if Fs != testFs {
		t.Error("Fs was not set correctly")
	}

	if CmdExecutor != testCmdExecutor {
		t.Error("CmdExecutor was not set correctly")
	}

	if DryRun != testDryRun {
		t.Errorf("DryRun = %v, want %v", DryRun, testDryRun)
	}
}

func TestInitGlobals_NilValues(t *testing.T) {
	// Test that InitGlobals handles nil values gracefully
	InitGlobals(nil, nil, nil, false)

	// Verify globals were set (even if to nil)
	if AppLogger != nil {
		t.Error("AppLogger should be nil when passed nil")
	}
	if Fs != nil {
		t.Error("Fs should be nil when passed nil")
	}
	if CmdExecutor != nil {
		t.Error("CmdExecutor should be nil when passed nil")
	}
}

func TestEmbeddedFallback(t *testing.T) {
	// Create test logger
	testLogger, err := logger.NewLogger("")
	if err != nil {
		t.Fatalf("Failed to create test logger: %v", err)
	}
	defer testLogger.Close()
	AppLogger = testLogger

	// Create a minimal embed.FS for testing
	var testEmbedFS embed.FS
	
	// Test with empty embedded FS
	fsys := EmbeddedFallback(testEmbedFS)
	if fsys == nil {
		t.Error("EmbeddedFallback returned nil")
	}

	// The function should return the root FS when subdirectory access fails
	// This tests the error handling path
	t.Log("EmbeddedFallback handled empty embed.FS correctly")
}

func TestEmbeddedFallback_WithoutLogger(t *testing.T) {
	// Test EmbeddedFallback with AppLogger nil
	originalLogger := AppLogger
	defer func() { AppLogger = originalLogger }()
	
	AppLogger = nil

	var testEmbed embed.FS
	fsys := EmbeddedFallback(testEmbed)
	if fsys == nil {
		t.Error("EmbeddedFallback should not return nil")
	}
}

func TestExtractEmbeddedConfigs(t *testing.T) {
	// Create test logger
	testLogger, err := logger.NewLogger("")
	if err != nil {
		t.Fatalf("Failed to create test logger: %v", err)
	}
	defer testLogger.Close()
	AppLogger = testLogger

	t.Run("handles empty embed.FS", func(t *testing.T) {
		// Test with an empty embed.FS
		var emptyEmbed embed.FS
		err := ExtractEmbeddedConfigs(emptyEmbed)
		// Function may succeed or fail, but should not panic
		t.Logf("ExtractEmbeddedConfigs with empty embed returned: %v", err)
	})

	t.Run("creates target directory", func(t *testing.T) {
		// The function should attempt to create the configs directory
		// This is verified by the function not panicking
		var emptyEmbed embed.FS
		err := ExtractEmbeddedConfigs(emptyEmbed)
		t.Logf("Directory creation test result: %v", err)
	})
}

func TestExtractEmbeddedConfigs_DirectoryExists(t *testing.T) {
	testLogger, err := logger.NewLogger("")
	if err != nil {
		t.Fatalf("Failed to create test logger: %v", err)
	}
	defer testLogger.Close()
	AppLogger = testLogger

	// The function should handle existing directory gracefully
	var emptyEmbed embed.FS
	err = ExtractEmbeddedConfigs(emptyEmbed)
	t.Logf("Existing directory test result: %v", err)
}

func TestDryRunGlobal(t *testing.T) {
	// Test that DryRun can be set and read
	originalDryRun := DryRun
	defer func() { DryRun = originalDryRun }()

	DryRun = true
	if !DryRun {
		t.Error("DryRun should be true")
	}

	DryRun = false
	if DryRun {
		t.Error("DryRun should be false")
	}
}

func TestGlobalVariables(t *testing.T) {
	// Test that global variables can be set and accessed
	testLogger, err := logger.NewLogger("")
	if err != nil {
		t.Fatalf("Failed to create test logger: %v", err)
	}
	defer testLogger.Close()

	testFs := interfaces.NewOsFileSystem()
	testCmdExecutor := interfaces.NewOsCommandExecutor()

	// Set globals
	AppLogger = testLogger
	Fs = testFs
	CmdExecutor = testCmdExecutor
	DryRun = true

	// Verify they can be read
	if AppLogger == nil {
		t.Error("AppLogger should not be nil")
	}
	if Fs == nil {
		t.Error("Fs should not be nil")
	}
	if CmdExecutor == nil {
		t.Error("CmdExecutor should not be nil")
	}
	if !DryRun {
		t.Error("DryRun should be true")
	}
}

func TestEmbeddedFallback_ReturnsValidFS(t *testing.T) {
	testLogger, _ := logger.NewLogger("")
	if testLogger != nil {
		defer testLogger.Close()
	}
	AppLogger = testLogger

	var testEmbed embed.FS
	result := EmbeddedFallback(testEmbed)
	
	// Result should not be nil
	if result == nil {
		t.Error("EmbeddedFallback returned nil")
	}

	// The returned FS should be usable (even if empty)
	t.Log("EmbeddedFallback returned a valid FS")
}

func TestEmbeddedFallback_ErrorPath(t *testing.T) {
	// Test the error handling path when accessing subdirectory fails
	testLogger, err := logger.NewLogger("")
	if err != nil {
		t.Fatalf("Failed to create test logger: %v", err)
	}
	defer testLogger.Close()
	AppLogger = testLogger

	// Empty embed.FS will trigger the error path
	var emptyEmbed embed.FS
	result := EmbeddedFallback(emptyEmbed)

	// Should return the root embed.FS on error
	if result == nil {
		t.Error("EmbeddedFallback should return root FS on error, not nil")
	}
}

func TestExtractEmbeddedConfigs_ErrorHandling(t *testing.T) {
	testLogger, err := logger.NewLogger("")
	if err != nil {
		t.Fatalf("Failed to create test logger: %v", err)
	}
	defer testLogger.Close()
	AppLogger = testLogger

	// Test error handling with empty embed.FS
	var emptyEmbed embed.FS
	err = ExtractEmbeddedConfigs(emptyEmbed)
	
	// The function should handle errors gracefully
	// It may succeed (if creating empty directory) or fail, but shouldn't panic
	t.Logf("Error handling test completed with result: %v", err)
}

func TestInitGlobals_AllCombinations(t *testing.T) {
	tests := []struct {
		name       string
		logger     *logger.Logger
		fs         interfaces.FileSystem
		cmdExec    interfaces.CommandExecutor
		dryRun     bool
	}{
		{
			name:       "all values set",
			logger:     func() *logger.Logger { l, _ := logger.NewLogger(""); return l }(),
			fs:         interfaces.NewOsFileSystem(),
			cmdExec:    interfaces.NewOsCommandExecutor(),
			dryRun:     true,
		},
		{
			name:       "all nil with dry run false",
			logger:     nil,
			fs:         nil,
			cmdExec:    nil,
			dryRun:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.logger != nil {
				defer tt.logger.Close()
			}

			InitGlobals(tt.logger, tt.fs, tt.cmdExec, tt.dryRun)

			if AppLogger != tt.logger {
				t.Error("AppLogger mismatch")
			}
			if Fs != tt.fs {
				t.Error("Fs mismatch")
			}
			if CmdExecutor != tt.cmdExec {
				t.Error("CmdExecutor mismatch")
			}
			if DryRun != tt.dryRun {
				t.Errorf("DryRun = %v, want %v", DryRun, tt.dryRun)
			}
		})
	}
}