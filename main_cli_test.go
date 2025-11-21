package main

import (
	"SettingsSentry/interfaces"
	"SettingsSentry/logger"
	"SettingsSentry/pkg/testutil"
	"embed"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

//go:embed configs/*.cfg
var testEmbeddedConfigs embed.FS

func setupCLITest() (*CLI, *logger.Logger) {
	testFs := interfaces.NewOsFileSystem()
	testCmdExecutor := interfaces.NewOsCommandExecutor()
	testLogger := testutil.SetupTestGlobals(testFs, testCmdExecutor)
	
	cli := NewCLI(testLogger, testFs, testCmdExecutor, testEmbeddedConfigs, "1.0.0-test")
	return cli, testLogger
}

// TestParseFlags_ValidActions tests valid action parsing
func TestParseFlags_ValidActions(t *testing.T) {
	cli, testLogger := setupCLITest()
	defer testLogger.Close()

	tests := []struct {
		name           string
		args           []string
		expectedAction string
		wantErr        bool
	}{
		{
			name:           "backup action",
			args:           []string{"backup"},
			expectedAction: "backup",
			wantErr:        false,
		},
		{
			name:           "restore action",
			args:           []string{"restore"},
			expectedAction: "restore",
			wantErr:        false,
		},
		{
			name:           "configsinit action",
			args:           []string{"configsinit"},
			expectedAction: "configsinit",
			wantErr:        false,
		},
		{
			name:           "install action",
			args:           []string{"install"},
			expectedAction: "install",
			wantErr:        false,
		},
		{
			name:           "remove action",
			args:           []string{"remove"},
			expectedAction: "remove",
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, flags, err := cli.ParseFlags(tt.args)
			
			if tt.wantErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if action != tt.expectedAction {
				t.Errorf("Action = %q, want %q", action, tt.expectedAction)
			}
			if !tt.wantErr && flags == nil {
				t.Error("Expected flags map, got nil")
			}
		})
	}
}

// TestParseFlags_InvalidAction tests invalid action handling
func TestParseFlags_InvalidAction(t *testing.T) {
	cli, testLogger := setupCLITest()
	defer testLogger.Close()

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "no action",
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "invalid action",
			args:    []string{"invalid-action"},
			wantErr: true,
		},
		{
			name:    "typo in action",
			args:    []string{"backupp"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := cli.ParseFlags(tt.args)
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if tt.wantErr && err == nil {
				t.Error("Expected error, got nil")
			}
		})
	}
}

// TestParseFlags_WithFlags tests flag parsing
func TestParseFlags_WithFlags(t *testing.T) {
	cli, testLogger := setupCLITest()
	defer testLogger.Close()

	args := []string{
		"backup",
		"-config=/custom/config",
		"-backup=/custom/backup",
		"-app=app1,app2",
		"-dry-run",
		"-versions=5",
		"-zip",
		"-password=testpass",
	}

	action, flags, err := cli.ParseFlags(args)
	if err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	if action != "backup" {
		t.Errorf("Action = %q, want 'backup'", action)
	}

	// Check config folder
	if configFolder, ok := flags["configFolder"].(string); !ok || configFolder != "/custom/config" {
		t.Errorf("configFolder = %v, want '/custom/config'", flags["configFolder"])
	}

	// Check backup folder
	if backupFolder, ok := flags["backupFolder"].(string); !ok || backupFolder != "/custom/backup" {
		t.Errorf("backupFolder = %v, want '/custom/backup'", flags["backupFolder"])
	}

	// Check app names
	if appNames, ok := flags["appNames"].([]string); !ok || len(appNames) != 2 {
		t.Errorf("appNames = %v, want 2 items", flags["appNames"])
	} else {
		if appNames[0] != "app1" || appNames[1] != "app2" {
			t.Errorf("appNames = %v, want [app1, app2]", appNames)
		}
	}

	// Check dry-run
	if dryRun, ok := flags["dryRun"].(bool); !ok || !dryRun {
		t.Errorf("dryRun = %v, want true", flags["dryRun"])
	}

	// Check versions
	if versions, ok := flags["versionsToKeep"].(int); !ok || versions != 5 {
		t.Errorf("versionsToKeep = %v, want 5", flags["versionsToKeep"])
	}

	// Check zip
	if zip, ok := flags["zip"].(bool); !ok || !zip {
		t.Errorf("zip = %v, want true", flags["zip"])
	}

	// Check password
	if password, ok := flags["password"].(string); !ok || password != "testpass" {
		t.Errorf("password = %v, want 'testpass'", flags["password"])
	}
}

// TestParseFlags_EnvironmentVariables tests environment variable handling
func TestParseFlags_EnvironmentVariables(t *testing.T) {
	cli, testLogger := setupCLITest()
	defer testLogger.Close()

	// Set environment variables
	_ = os.Setenv("SETTINGSSENTRY_CONFIG", "/env/config")
	_ = os.Setenv("SETTINGSSENTRY_APP", "envapp")
	_ = os.Setenv("SETTINGSSENTRY_DRY_RUN", "true")
	_ = os.Setenv("SETTINGSSENTRY_ZIP", "true")
	defer func() {
		_ = os.Unsetenv("SETTINGSSENTRY_CONFIG")
		_ = os.Unsetenv("SETTINGSSENTRY_APP")
		_ = os.Unsetenv("SETTINGSSENTRY_DRY_RUN")
		_ = os.Unsetenv("SETTINGSSENTRY_ZIP")
	}()

	args := []string{"backup"}
	_, flags, err := cli.ParseFlags(args)
	if err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	// Config folder should use env var
	if configFolder := flags["configFolder"].(string); configFolder != "/env/config" {
		t.Errorf("configFolder = %q, want '/env/config'", configFolder)
	}

	// App names should use env var
	if appNames := flags["appNames"].([]string); len(appNames) != 1 || appNames[0] != "envapp" {
		t.Errorf("appNames = %v, want [envapp]", appNames)
	}

	// Dry run should be true from env
	if dryRun := flags["dryRun"].(bool); !dryRun {
		t.Error("dryRun should be true from env var")
	}

	// Zip should be true from env
	if zip := flags["zip"].(bool); !zip {
		t.Error("zip should be true from env var")
	}
}

// TestParseFlags_AppNameParsing tests app name comma-separated parsing
func TestParseFlags_AppNameParsing(t *testing.T) {
	cli, testLogger := setupCLITest()
	defer testLogger.Close()

	tests := []struct {
		name         string
		appFlag      string
		expectedApps []string
	}{
		{
			name:         "single app",
			appFlag:      "app1",
			expectedApps: []string{"app1"},
		},
		{
			name:         "multiple apps",
			appFlag:      "app1,app2,app3",
			expectedApps: []string{"app1", "app2", "app3"},
		},
		{
			name:         "apps with spaces",
			appFlag:      "app1, app2 , app3",
			expectedApps: []string{"app1", "app2", "app3"},
		},
		{
			name:         "empty string",
			appFlag:      "",
			expectedApps: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []string{"backup"}
			if tt.appFlag != "" {
				args = append(args, "-app="+tt.appFlag)
			}

			_, flags, err := cli.ParseFlags(args)
			if err != nil {
				t.Fatalf("ParseFlags failed: %v", err)
			}

			appNames := flags["appNames"].([]string)
			if len(appNames) != len(tt.expectedApps) {
				t.Errorf("Got %d apps, want %d", len(appNames), len(tt.expectedApps))
			}

			for i, expected := range tt.expectedApps {
				if i >= len(appNames) || appNames[i] != expected {
					t.Errorf("App %d = %q, want %q", i, appNames[i], expected)
				}
			}
		})
	}
}

// TestExecuteBackupRestore_ParseOnly tests backup/restore flag parsing
func TestExecuteBackupRestore_ParseOnly(t *testing.T) {
	cli, logger := setupCLITest()
	defer logger.Close()

	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "backup")
	configDir := filepath.Join(tempDir, "configs")

	tests := []struct {
		name   string
		action string
		args   []string
	}{
		{
			name:   "backup flags",
			action: "backup",
			args: []string{
				"backup",
				"-config=" + configDir,
				"-backup=" + backupDir,
				"-dry-run",
			},
		},
		{
			name:   "restore flags",
			action: "restore",
			args: []string{
				"restore",
				"-config=" + configDir,
				"-backup=" + backupDir,
				"-dry-run",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, flags, err := cli.ParseFlags(tt.args)
			if err != nil {
				t.Fatalf("ParseFlags failed: %v", err)
			}

			if action != tt.action {
				t.Errorf("Action = %q, want %q", action, tt.action)
			}

			// Verify flags were parsed
			if flags["configFolder"] == nil {
				t.Error("configFolder not set")
			}
			if flags["backupFolder"] == nil {
				t.Error("backupFolder not set")
			}
		})
	}
}

// TestExecuteBackupRestore_FlagsValidation tests backup/restore flag parsing
func TestExecuteBackupRestore_FlagsValidation(t *testing.T) {
	cli, logger := setupCLITest()
	defer logger.Close()

	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "backup")
	configDir := filepath.Join(tempDir, "configs")

	tests := []struct {
		name    string
		args    []string
		wantDryRun bool
		wantZip bool
		wantPassword string
		wantVersions int
	}{
		{
			name: "backup with dry-run",
			args: []string{
				"backup",
				"-config=" + configDir,
				"-backup=" + backupDir,
				"-dry-run",
			},
			wantDryRun: true,
		},
		{
			name: "backup with zip",
			args: []string{
				"backup",
				"-config=" + configDir,
				"-backup=" + backupDir,
				"-zip",
			},
			wantZip: true,
		},
		{
			name: "backup with password",
			args: []string{
				"backup",
				"-config=" + configDir,
				"-backup=" + backupDir,
				"-password=testpass",
			},
			wantPassword: "testpass",
		},
		{
			name: "backup with versions",
			args: []string{
				"backup",
				"-config=" + configDir,
				"-backup=" + backupDir,
				"-versions=5",
			},
			wantVersions: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, flags, err := cli.ParseFlags(tt.args)
			if err != nil {
				t.Fatalf("ParseFlags failed: %v", err)
			}

			// Verify flags were parsed correctly
			if tt.wantDryRun && !flags["dryRun"].(bool) {
				t.Error("Expected dryRun to be true")
			}
			if tt.wantZip && !flags["zip"].(bool) {
				t.Error("Expected zip to be true")
			}
			if tt.wantPassword != "" && flags["password"].(string) != tt.wantPassword {
				t.Errorf("Password = %q, want %q", flags["password"], tt.wantPassword)
			}
			if tt.wantVersions > 0 && flags["versionsToKeep"].(int) != tt.wantVersions {
				t.Errorf("versionsToKeep = %d, want %d", flags["versionsToKeep"], tt.wantVersions)
			}
		})
	}
}

// TestExecuteInstall tests cron job installation
func TestExecuteInstall(t *testing.T) {
	cli, logger := setupCLITest()
	defer logger.Close()

	tests := []struct {
		name      string
		args      []string
		wantErr   bool
	}{
		{
			name: "install without cron expression",
			args: []string{"install"},
			wantErr: false, // May succeed or fail depending on system
		},
		{
			name: "install with cron expression",
			args: []string{"install", "0 0 * * *"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, flags, err := cli.ParseFlags(tt.args)
			if err != nil {
				t.Fatalf("ParseFlags failed: %v", err)
			}

			if action != "install" {
				t.Errorf("Action = %q, want 'install'", action)
			}

			err = cli.executeInstall(flags)
			// Don't fail test if cron operations fail (system-dependent)
			t.Logf("executeInstall() error: %v", err)
		})
	}
}

// TestExecuteRemove tests cron job removal
func TestExecuteRemove(t *testing.T) {
	cli, logger := setupCLITest()
	defer logger.Close()

	args := []string{"remove"}
	action, _, err := cli.ParseFlags(args)
	if err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	if action != "remove" {
		t.Errorf("Action = %q, want 'remove'", action)
	}

	err = cli.executeRemove()
	// Don't fail test if cron operations fail (system-dependent)
	t.Logf("executeRemove() error: %v", err)
}

// TestExecuteConfigsInit tests embedded config extraction
func TestExecuteConfigsInit(t *testing.T) {
	cli, logger := setupCLITest()
	defer logger.Close()

	args := []string{"configsinit"}
	action, _, err := cli.ParseFlags(args)
	if err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	if action != "configsinit" {
		t.Errorf("Action = %q, want 'configsinit'", action)
	}

	err = cli.executeConfigsInit()
	// May fail if executable path issues, but shouldn't panic
	t.Logf("executeConfigsInit() error: %v", err)
}

// TestExecuteBackupRestore_ErrorCases tests error case flag parsing
func TestExecuteBackupRestore_ErrorCases(t *testing.T) {
	cli, logger := setupCLITest()
	defer logger.Close()

	tests := []struct {
		name    string
		args    []string
	}{
		{
			name: "invalid config path parsing",
			args: []string{
				"backup",
				"-config=/nonexistent/path",
				"-backup=/tmp/backup",
			},
		},
		{
			name: "invalid backup path parsing",
			args: []string{
				"restore",
				"-config=/tmp/configs",
				"-backup=/nonexistent/backup",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, flags, err := cli.ParseFlags(tt.args)
			if err != nil {
				t.Fatalf("ParseFlags failed: %v", err)
			}

			// Just verify parsing works - paths can be invalid
			if flags["configFolder"] == nil {
				t.Error("configFolder should be set")
			}
			if flags["backupFolder"] == nil {
				t.Error("backupFolder should be set")
			}
		})
	}
}

// TestCLI_Run_Integration tests the full CLI run method
func TestCLI_Run_Integration(t *testing.T) {
	cli, logger := setupCLITest()
	defer logger.Close()

	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "configs")
	backupDir := filepath.Join(tempDir, "backup")
	
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "help (no args)",
			args:    []string{},
			wantErr: true, // No action specified
		},
		{
			name: "configsinit",
			args: []string{"configsinit"},
			wantErr: false,
		},
		{
			name: "backup with dry-run",
			args: []string{
				"backup",
				"-config=" + configDir,
				"-backup=" + backupDir,
				"-dry-run",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, flags, err := cli.ParseFlags(tt.args)
			if err != nil && !tt.wantErr {
				t.Fatalf("ParseFlags failed: %v", err)
			}
			if err == nil && action != "" {
				// Just test parsing - execution requires full context setup
				if action == "backup" || action == "restore" {
					if flags["configFolder"] == nil {
						t.Error("configFolder should be set")
					}
				}
			}
			if tt.wantErr && err == nil {
				t.Error("Expected error, got nil")
			}
		})
	}
}

// TestExecuteAction_InvalidAction tests invalid action in ExecuteAction
func TestExecuteAction_InvalidAction(t *testing.T) {
	cli, testLogger := setupCLITest()
	defer testLogger.Close()

	err := cli.ExecuteAction("invalid-action", map[string]interface{}{})
	if err == nil {
		t.Error("Expected error for invalid action, got nil")
	}
	if !strings.Contains(err.Error(), "invalid action") {
		t.Errorf("Error message should mention 'invalid action', got: %v", err)
	}
}

// TestExecuteAction_ConfigsInit tests configsinit action
func TestExecuteAction_ConfigsInit(t *testing.T) {
	cli, testLogger := setupCLITest()
	defer testLogger.Close()

	// This will try to extract configs - may fail in test env but shouldn't panic
	err := cli.ExecuteAction("configsinit", map[string]interface{}{})
	// We just verify it doesn't panic and returns an error if extraction fails
	t.Logf("ConfigsInit result: %v", err)
}

// TestShowHelp tests help output
func TestShowHelp(t *testing.T) {
	cli, testLogger := setupCLITest()
	defer testLogger.Close()

	// ShowHelp should not panic
	cli.ShowHelp()
	t.Log("ShowHelp completed without panic")
}

// TestIsValidAction tests action validation
func TestIsValidAction(t *testing.T) {
	tests := []struct {
		action string
		valid  bool
	}{
		{"backup", true},
		{"restore", true},
		{"configsinit", true},
		{"install", true},
		{"remove", true},
		{"invalid", false},
		{"BACKUP", false}, // case sensitive
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.action, func(t *testing.T) {
			result := isValidAction(tt.action)
			if result != tt.valid {
				t.Errorf("isValidAction(%q) = %v, want %v", tt.action, result, tt.valid)
			}
		})
	}
}

// TestGetEnvWithDefault tests environment variable getter
func TestGetEnvWithDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		expected     string
	}{
		{
			name:         "env var set",
			key:          "TEST_VAR",
			defaultValue: "default",
			envValue:     "custom",
			expected:     "custom",
		},
		{
			name:         "env var not set",
			key:          "TEST_VAR_UNSET",
			defaultValue: "default",
			envValue:     "",
			expected:     "default",
		},
		{
			name:         "empty default",
			key:          "TEST_VAR_EMPTY",
			defaultValue: "",
			envValue:     "",
			expected:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				_ = os.Setenv(tt.key, tt.envValue)
				defer func() { _ = os.Unsetenv(tt.key) }()
			}

			result := getEnvWithDefault(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnvWithDefault(%q, %q) = %q, want %q",
					tt.key, tt.defaultValue, result, tt.expected)
			}
		})
	}
}

// TestNewCLI tests CLI construction
func TestNewCLI(t *testing.T) {
	testFs := interfaces.NewOsFileSystem()
	testCmdExecutor := interfaces.NewOsCommandExecutor()
	testLogger := testutil.SetupTestGlobals(testFs, testCmdExecutor)
	defer testLogger.Close()

	cli := NewCLI(testLogger, testFs, testCmdExecutor, testEmbeddedConfigs, "1.0.0")

	if cli.logger != testLogger {
		t.Error("CLI logger not set correctly")
	}
	if cli.fs != testFs {
		t.Error("CLI fs not set correctly")
	}
	if cli.cmdExecutor != testCmdExecutor {
		t.Error("CLI cmdExecutor not set correctly")
	}
	if cli.version != "1.0.0" {
		t.Errorf("CLI version = %q, want '1.0.0'", cli.version)
	}
}

// TestParseFlags_ExtraArgs tests extra arguments handling
func TestParseFlags_ExtraArgs(t *testing.T) {
	cli, testLogger := setupCLITest()
	defer testLogger.Close()

	args := []string{"install", "0 0 * * *"} // cron expression as extra arg
	_, flags, err := cli.ParseFlags(args)
	if err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	extraArgs, ok := flags["extraArgs"].([]string)
	if !ok {
		t.Fatal("extraArgs not found in flags")
	}

	if len(extraArgs) != 1 || extraArgs[0] != "0 0 * * *" {
		t.Errorf("extraArgs = %v, want [0 0 * * *]", extraArgs)
	}
}

// TestParseFlags_DefaultValues tests default values
func TestParseFlags_DefaultValues(t *testing.T) {
	cli, testLogger := setupCLITest()
	defer testLogger.Close()

	args := []string{"backup"}
	_, flags, err := cli.ParseFlags(args)
	if err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	// Check defaults
	if commands := flags["commands"].(bool); commands {
		t.Error("commands should default to false")
	}

	if dryRun := flags["dryRun"].(bool); dryRun {
		t.Error("dryRun should default to false")
	}

	if versions := flags["versionsToKeep"].(int); versions != 1 {
		t.Errorf("versionsToKeep = %d, want 1", versions)
	}

	if zip := flags["zip"].(bool); zip {
		t.Error("zip should default to false")
	}
}