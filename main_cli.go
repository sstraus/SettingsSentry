package main

import (
	cronjob "SettingsSentry/cron"
	"SettingsSentry/interfaces"
	"SettingsSentry/logger"
	"SettingsSentry/pkg/backup"
	"SettingsSentry/pkg/config"
	"SettingsSentry/pkg/printer"
	"SettingsSentry/pkg/util"
	"embed"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CLI handles command-line interface operations
type CLI struct {
	logger           *logger.Logger
	fs               interfaces.FileSystem
	cmdExecutor      interfaces.CommandExecutor
	embeddedConfigs  embed.FS
	version          string
	envConfigFolder  string
	envBackupFolder  string
	envAppName       string
	envCommands      bool
	envDryRun        bool
	envZip           bool
	envPassword      string
}

// NewCLI creates a new CLI instance
func NewCLI(logger *logger.Logger, fs interfaces.FileSystem, cmdExecutor interfaces.CommandExecutor, embeddedConfigs embed.FS, version string) *CLI {
	return &CLI{
		logger:          logger,
		fs:              fs,
		cmdExecutor:     cmdExecutor,
		embeddedConfigs: embeddedConfigs,
		version:         version,
	}
}

// ParseFlags parses command-line arguments and environment variables
func (c *CLI) ParseFlags(args []string) (action string, flags map[string]interface{}, err error) {
	if len(args) < 1 {
		return "", nil, errors.New("no action specified")
	}

	// Get iCloud path for default backup location
	icloudPath, err := config.GetICloudFolderLocation()
	if err != nil {
		c.logger.Logf("Error: iCloud path not found - %v", err)
		icloudPath = ""
	} else {
		icloudPath = filepath.Join(icloudPath, "settingssentry_backups")
	}
	if icloudPath == "" {
		homeDir, homeErr := config.GetHomeDirectory()
		if homeErr == nil {
			icloudPath = filepath.Join(homeDir, ".settingssentry_backups")
			c.logger.Logf("Warning: iCloud path not found, using default backup path: %s", icloudPath)
		} else {
			return "", nil, errors.New("cannot determine iCloud or home directory for default backup path")
		}
	}

	// Get environment variable defaults
	c.envConfigFolder = getEnvWithDefault("SETTINGSSENTRY_CONFIG", "configs")
	c.envBackupFolder = getEnvWithDefault("SETTINGSSENTRY_BACKUP", icloudPath)
	c.envAppName = os.Getenv("SETTINGSSENTRY_APP")
	c.envCommands = os.Getenv("SETTINGSSENTRY_COMMANDS") == "true"
	c.envDryRun = os.Getenv("SETTINGSSENTRY_DRY_RUN") == "true"
	c.envZip = os.Getenv("SETTINGSSENTRY_ZIP") == "true"
	c.envPassword = os.Getenv("SETTINGSSENTRY_PASSWORD")

	action = args[0]

	// Validate action
	if !isValidAction(action) {
		return "", nil, fmt.Errorf("invalid action: %s", action)
	}

	// Define flags using a specific FlagSet for the action
	actionFlags := flag.NewFlagSet(action, flag.ContinueOnError)
	configFolder := actionFlags.String("config", c.envConfigFolder, "Path to the configuration folder (env: SETTINGSSENTRY_CONFIG)")
	backupFolder := actionFlags.String("backup", c.envBackupFolder, "Path to the backup folder (env: SETTINGSSENTRY_BACKUP)")
	appNameFlag := actionFlags.String("app", c.envAppName, "Optional: Comma-separated list of application names to process (env: SETTINGSSENTRY_APP)")
	commands := actionFlags.Bool("allow-commands", c.envCommands, "Optional: Allow execution of pre-backup/restore commands from config files. SECURITY WARNING: Only enable for trusted configs! Commands execute with full user privileges. (env: SETTINGSSENTRY_COMMANDS)")
	dryRunFlag := actionFlags.Bool("dry-run", c.envDryRun, "Optional: Perform a dry run without making any changes (env: SETTINGSSENTRY_DRY_RUN)")
	versionsToKeep := actionFlags.Int("versions", 1, "Number of backup versions to keep")
	password := actionFlags.String("password", c.envPassword, "Optional: Password to encrypt/decrypt backups (env: SETTINGSSENTRY_PASSWORD)")
	zipFlag := actionFlags.Bool("zip", c.envZip, "Optional: Create backup as a zip archive instead of a directory (env: SETTINGSSENTRY_ZIP)")
	logFilePath := actionFlags.String("logfile", "", "Optional: Path to log file.")

	// Parse arguments starting from the one after the action
	if err := actionFlags.Parse(args[1:]); err != nil {
		return "", nil, fmt.Errorf("error parsing flags: %w", err)
	}

	// Validate versionsToKeep is non-negative
	if *versionsToKeep < 0 {
		return "", nil, fmt.Errorf("versions must be non-negative, got %d", *versionsToKeep)
	}

	// Split the appNameFlag string into a slice
	var appNames []string
	if *appNameFlag != "" {
		appNames = strings.Split(*appNameFlag, ",")
		// Trim whitespace from each app name and filter out empty strings
		var filteredNames []string
		for _, name := range appNames {
			trimmed := strings.TrimSpace(name)
			if trimmed != "" {
				filteredNames = append(filteredNames, trimmed)
			}
		}
		appNames = filteredNames
	}

	flags = map[string]interface{}{
		"configFolder":   *configFolder,
		"backupFolder":   *backupFolder,
		"appNames":       appNames,
		"commands":       *commands,
		"dryRun":         *dryRunFlag,
		"versionsToKeep": *versionsToKeep,
		"password":       *password,
		"zip":            *zipFlag,
		"logFilePath":    *logFilePath,
		"extraArgs":      actionFlags.Args(),
	}

	return action, flags, nil
}

// ExecuteAction executes the specified action with the given flags
func (c *CLI) ExecuteAction(action string, flags map[string]interface{}) error {
	switch action {
	case "backup", "restore":
		return c.executeBackupRestore(action, flags)
	case "configsinit":
		return c.executeConfigsInit()
	case "install":
		return c.executeInstall(flags)
	case "remove":
		return c.executeRemove()
	default:
		return fmt.Errorf("invalid action: %s", action)
	}
}

// executeBackupRestore handles backup and restore actions
func (c *CLI) executeBackupRestore(action string, flags map[string]interface{}) error {
	configFolder := flags["configFolder"].(string)
	backupFolder := flags["backupFolder"].(string)
	appNames := flags["appNames"].([]string)
	commands := flags["commands"].(bool)
	dryRun := flags["dryRun"].(bool)
	versionsToKeep := flags["versionsToKeep"].(int)
	zipFlag := flags["zip"].(bool)
	password := flags["password"].(string)

	util.DryRun = dryRun
	backup.DryRun = dryRun

	mainPrinter := printer.NewPrinter("", c.logger)
	backup.Printer = mainPrinter

	isBackup := action == "backup"
	backup.ProcessConfiguration(configFolder, backupFolder, appNames, isBackup, commands, versionsToKeep, zipFlag, password)
	return nil
}

// executeConfigsInit handles configsinit action
func (c *CLI) executeConfigsInit() error {
	err := util.ExtractEmbeddedConfigs(c.embeddedConfigs)
	if err != nil {
		return fmt.Errorf("error extracting embedded configs: %w", err)
	}
	return nil
}

// executeInstall handles install action
func (c *CLI) executeInstall(flags map[string]interface{}) error {
	cronExpression := ""
	extraArgs := flags["extraArgs"].([]string)
	if len(extraArgs) > 0 {
		cronExpression = extraArgs[0]
	}

	// Check if --allow-commands flag is set
	allowCommands := false
	if val, ok := flags["commands"].(bool); ok {
		allowCommands = val
	}

	err := cronjob.InstallCronJob(cronExpression, allowCommands)
	if err != nil {
		return fmt.Errorf("failed to install cron job: %w", err)
	}

	if allowCommands {
		c.logger.Logf("CRON job installed successfully with --allow-commands enabled")
		c.logger.Logf("WARNING: Pre/post backup commands will execute with full user privileges")
	} else {
		c.logger.Logf("CRON job installed successfully (commands disabled for security)")
	}
	return nil
}

// executeRemove handles remove action
func (c *CLI) executeRemove() error {
	err := cronjob.RemoveCronJob()
	if err != nil {
		return fmt.Errorf("failed to remove cron job: %w", err)
	}
	c.logger.Logf("CRON job removed successfully")
	return nil
}

// ShowHelp displays help information
func (c *CLI) ShowHelp() {
	c.logger.Logf("Usage: settingssentry <action> [options]")
	c.logger.Logf("")
	c.logger.Logf("Actions:")
	c.logger.Logf("  backup      - Backup configuration files to the specified backup folder")
	c.logger.Logf("  restore     - Restore the files to their original locations")
	c.logger.Logf("  configsinit - Extract embedded default configs to a 'configs' directory next to the executable")
	c.logger.Logf("  install     - Install the application as a CRON job that runs at every reboot")
	c.logger.Logf("                You can provide a valid cron expression as parameter (e.g., '0 9 * * *')")
	c.logger.Logf("                Use --allow-commands to enable command execution in scheduled backups")
	c.logger.Logf("  remove      - Remove the previously installed CRON job")
	c.logger.Logf("")
	c.logger.Logf("Options:")
	c.logger.Logf("  -config=<path>        Path to the configuration folder (default: %s)", c.envConfigFolder)
	c.logger.Logf("  -backup=<path>        Path to the backup folder (default: %s)", c.envBackupFolder)
	c.logger.Logf("  -app=<app1,app2,...>  Comma-separated list of application names to process")
	c.logger.Logf("  -allow-commands       [SECURITY WARNING] Allow execution of pre/post backup/restore commands")
	c.logger.Logf("                        Commands execute with full user privileges. Only enable for trusted configs!")
	c.logger.Logf("  -dry-run              Perform a dry run without making any changes")
	c.logger.Logf("  -versions=<n>         Number of backup versions to keep (default: 1, 0 = keep all)")
	c.logger.Logf("  -zip                  Create backup as a zip archive instead of a directory")
	c.logger.Logf("  -password=<pwd>       Password to encrypt/decrypt backups (AES-256-GCM)")
	c.logger.Logf("  -logfile=<path>       Path to log file (logs to console + file if provided)")
	c.logger.Logf("")
	c.logger.Logf("Environment Variables:")
	c.logger.Logf("  SETTINGSSENTRY_CONFIG      Path to configuration folder")
	c.logger.Logf("  SETTINGSSENTRY_BACKUP      Path to backup folder")
	c.logger.Logf("  SETTINGSSENTRY_APP         Comma-separated app names")
	c.logger.Logf("  SETTINGSSENTRY_COMMANDS    Set to 'true' to allow command execution")
	c.logger.Logf("  SETTINGSSENTRY_DRY_RUN     Set to 'true' for dry-run mode")
	c.logger.Logf("  SETTINGSSENTRY_ZIP         Set to 'true' to create zip archives")
	c.logger.Logf("  SETTINGSSENTRY_PASSWORD    Password for encryption/decryption")
	c.logger.Logf("")
	c.logger.Logf("Examples:")
	c.logger.Logf("  settingssentry backup")
	c.logger.Logf("  settingssentry backup -dry-run")
	c.logger.Logf("  settingssentry backup -app=Brew,Git -zip -password=mypass")
	c.logger.Logf("  settingssentry restore -app=Brew")
	c.logger.Logf("  settingssentry install --allow-commands")
	c.logger.Logf("  settingssentry install '0 9 * * *'  # Daily at 9 AM")
	c.logger.Logf("")
	c.logger.Logf("Documentation: https://github.com/sstraus/SettingsSentry")
	c.logger.Logf("")

	installed, err := cronjob.IsCronJobInstalled()
	if err != nil {
		c.logger.Logf("Error checking CRON job installation: %v", err)
		return
	}

	if installed {
		c.logger.Logf("Status: CRON job is currently installed - backups will run automatically")
	}
}

// Helper functions

// isValidAction checks if the action is valid
func isValidAction(action string) bool {
	validActions := []string{"backup", "restore", "configsinit", "install", "remove"}
	for _, valid := range validActions {
		if action == valid {
			return true
		}
	}
	return false
}

// getEnvWithDefault gets an environment variable with a default value
func getEnvWithDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}