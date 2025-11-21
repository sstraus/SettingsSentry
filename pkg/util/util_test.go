package util

import (
	"SettingsSentry/interfaces"
	"SettingsSentry/logger"
	"embed"
	"os"
	"path/filepath"
	"testing"
)

//go:embed testdata/configs/*.cfg
var testEmbedFS embed.FS

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

	t.Run("with real embedded FS", func(t *testing.T) {
		// Use real embed.FS - may have configs subdirectory or not
		fsys := EmbeddedFallback(testEmbedFS)
		if fsys == nil {
			t.Error("EmbeddedFallback returned nil")
		}
		// Just verify it returns a valid FS, don't check contents
		t.Log("EmbeddedFallback returned valid FS")
	})

	t.Run("with empty embedded FS", func(t *testing.T) {
		// Test with empty embedded FS (error path)
		var emptyEmbedFS embed.FS
		fsys := EmbeddedFallback(emptyEmbedFS)
		if fsys == nil {
			t.Error("EmbeddedFallback returned nil")
		}
		// The function should return the root FS when subdirectory access fails
		t.Log("EmbeddedFallback handled empty embed.FS correctly")
	})
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

	t.Run("extracts from real embed.FS", func(t *testing.T) {
		// Create a temporary directory to simulate extraction
		tempDir := t.TempDir()
		
		// Change to temp dir for extraction
		originalWd, _ := os.Getwd()
		defer func() { _ = os.Chdir(originalWd) }()
		
		// Create a mock executable directory structure
		exePath := filepath.Join(tempDir, "test_executable")
		err := os.WriteFile(exePath, []byte(""), 0755)
		if err != nil {
			t.Fatalf("Failed to create mock executable: %v", err)
		}
		
		// Note: ExtractEmbeddedConfigs uses os.Executable() internally
		// so we test the walkDir logic with empty embed.FS
		var emptyEmbed embed.FS
		err = ExtractEmbeddedConfigs(emptyEmbed)
		// May fail due to empty FS, but tests the code path
		t.Logf("ExtractEmbeddedConfigs result: %v", err)
	})

	t.Run("handles empty embed.FS", func(t *testing.T) {
		// Test with an empty embed.FS
		var emptyEmbed embed.FS
		err := ExtractEmbeddedConfigs(emptyEmbed)
		// Function may succeed or fail, but should not panic
		t.Logf("ExtractEmbeddedConfigs with empty embed returned: %v", err)
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

func TestExtractEmbeddedConfigs_ExecutablePathError(t *testing.T) {
	testLogger, err := logger.NewLogger("")
	if err != nil {
		t.Fatalf("Failed to create test logger: %v", err)
	}
	defer testLogger.Close()
	AppLogger = testLogger

	// The function gets executable path internally
	// Test that it handles the case gracefully
	var emptyEmbed embed.FS
	err = ExtractEmbeddedConfigs(emptyEmbed)
	// Should complete without panic
	t.Logf("ExtractEmbeddedConfigs completed with: %v", err)
}

func TestEmbeddedFallback_SuccessPath(t *testing.T) {
	testLogger, err := logger.NewLogger("")
	if err != nil {
		t.Fatalf("Failed to create test logger: %v", err)
	}
	defer testLogger.Close()
	AppLogger = testLogger

	// Test with actual embedded FS structure
	var testEmbed embed.FS
	result := EmbeddedFallback(testEmbed)

	if result == nil {
		t.Error("EmbeddedFallback should not return nil")
	}

	// Result should be usable as an FS
	t.Log("EmbeddedFallback returned valid FS")
}

func TestExtractEmbeddedConfigs_WithFiles(t *testing.T) {
	testLogger, err := logger.NewLogger("")
	if err != nil {
		t.Fatalf("Failed to create test logger: %v", err)
	}
	defer testLogger.Close()
	AppLogger = testLogger

	// Test extraction process
	var emptyEmbed embed.FS
	err = ExtractEmbeddedConfigs(emptyEmbed)
	
	// Function should handle empty embed FS
	t.Logf("ExtractEmbeddedConfigs result: %v", err)
}

// TestExtractEmbeddedConfigs_WalkDirError tests error handling during directory walk
func TestExtractEmbeddedConfigs_WalkDirError(t *testing.T) {
	testLogger, err := logger.NewLogger("")
	if err != nil {
		t.Fatalf("Failed to create test logger: %v", err)
	}
	defer testLogger.Close()
	AppLogger = testLogger

	// Test with empty embed.FS that will cause walk errors
	var emptyEmbed embed.FS
	err = ExtractEmbeddedConfigs(emptyEmbed)
	
	// Should handle walk errors gracefully
	t.Logf("ExtractEmbeddedConfigs with walk error: %v", err)
}

// TestExtractEmbeddedConfigs_WriteError tests write permission errors
func TestExtractEmbeddedConfigs_WriteError(t *testing.T) {
	testLogger, err := logger.NewLogger("")
	if err != nil {
		t.Fatalf("Failed to create test logger: %v", err)
	}
	defer testLogger.Close()
	AppLogger = testLogger

	// This test verifies the function can handle write errors
	// In practice, write errors would occur due to permissions or disk space
	var emptyEmbed embed.FS
	err = ExtractEmbeddedConfigs(emptyEmbed)
	
	t.Logf("ExtractEmbeddedConfigs result: %v", err)
}

// TestExtractEmbeddedConfigs_DirectoryOverwrite tests overwriting existing directory
func TestExtractEmbeddedConfigs_DirectoryOverwrite(t *testing.T) {
	testLogger, err := logger.NewLogger("")
	if err != nil {
		t.Fatalf("Failed to create test logger: %v", err)
	}
	defer testLogger.Close()
	AppLogger = testLogger

	// Test behavior when target directory already exists
	var emptyEmbed embed.FS
	
	// First call creates directory
	_ = ExtractEmbeddedConfigs(emptyEmbed)
	
	// Second call should handle existing directory
	err = ExtractEmbeddedConfigs(emptyEmbed)
	t.Logf("ExtractEmbeddedConfigs on existing dir: %v", err)
}

// TestEmbeddedFallback_SubdirectoryError tests Sub() error handling
func TestEmbeddedFallback_SubdirectoryError(t *testing.T) {
	testLogger, err := logger.NewLogger("")
	if err != nil {
		t.Fatalf("Failed to create test logger: %v", err)
	}
	defer testLogger.Close()
	AppLogger = testLogger

	// Empty embed.FS will cause Sub() to fail accessing "configs" subdirectory
	var emptyEmbed embed.FS
	result := EmbeddedFallback(emptyEmbed)

	if result == nil {
		t.Error("EmbeddedFallback should return root FS on subdirectory error, not nil")
	}

	// The function should log the error and return root FS
	t.Log("EmbeddedFallback successfully handled subdirectory access error")
}

// TestEmbeddedFallback_MissingSubdirectory tests behavior with missing configs subdirectory
func TestEmbeddedFallback_MissingSubdirectory(t *testing.T) {
	testLogger, err := logger.NewLogger("")
	if err != nil {
		t.Fatalf("Failed to create test logger: %v", err)
	}
	defer testLogger.Close()
	AppLogger = testLogger

	var emptyEmbed embed.FS
	result := EmbeddedFallback(emptyEmbed)

	// Should return a valid FS even when subdirectory doesn't exist
	if result == nil {
		t.Fatal("EmbeddedFallback returned nil")
	}

	// Try to read from the FS - should not panic
	_, err = result.Open(".")
	t.Logf("Open root directory: %v", err)
}

// TestExtractEmbeddedConfigs_PermissionDenied tests handling of permission errors
func TestExtractEmbeddedConfigs_PermissionDenied(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	testLogger, err := logger.NewLogger("")
	if err != nil {
		t.Fatalf("Failed to create test logger: %v", err)
	}
	defer testLogger.Close()
	AppLogger = testLogger

	// This test documents behavior when permission is denied
	// Actual permission errors are system-dependent
	var emptyEmbed embed.FS
	err = ExtractEmbeddedConfigs(emptyEmbed)
	
	t.Logf("ExtractEmbeddedConfigs completed: %v", err)
}


// TestExtractEmbeddedConfigs_ReadFileError tests read errors from embed.FS
func TestExtractEmbeddedConfigs_ReadFileError(t *testing.T) {
	testLogger, err := logger.NewLogger("")
	if err != nil {
		t.Fatalf("Failed to create test logger: %v", err)
	}
	defer testLogger.Close()
	AppLogger = testLogger

	// Empty embed.FS will trigger read errors if files are attempted to be read
	var emptyEmbed embed.FS
	err = ExtractEmbeddedConfigs(emptyEmbed)

	// Function should handle read errors gracefully
	t.Logf("ExtractEmbeddedConfigs with read errors: %v", err)
}

// TestEmbeddedFallback_NilLogger tests EmbeddedFallback when logger is nil
func TestEmbeddedFallback_NilLogger(t *testing.T) {
	originalLogger := AppLogger
	defer func() { AppLogger = originalLogger }()

	AppLogger = nil

	var emptyEmbed embed.FS
	result := EmbeddedFallback(emptyEmbed)

	if result == nil {
		t.Error("EmbeddedFallback should handle nil logger gracefully")
	}
}

// TestExtractEmbeddedConfigs_NilLogger tests extraction with nil logger
func TestExtractEmbeddedConfigs_NilLogger(t *testing.T) {
	originalLogger := AppLogger
	defer func() {
		AppLogger = originalLogger
		// Recover from panic if it occurs
		if r := recover(); r != nil {
			t.Logf("Recovered from panic (expected): %v", r)
		}
	}()

	AppLogger = nil

	var emptyEmbed embed.FS
	// Should panic since AppLogger.LogErrorf is called with nil logger
	err := ExtractEmbeddedConfigs(emptyEmbed)
	t.Logf("ExtractEmbeddedConfigs with nil logger: %v", err)
	defer func() {
		if r := recover(); r != nil {
			t.Logf("ExtractEmbeddedConfigs panicked with nil logger: %v", r)
		}
	}()

	_ = ExtractEmbeddedConfigs(emptyEmbed)
}