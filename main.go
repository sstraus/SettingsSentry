package main

import (
	cronjob "SettingsSentry/cron"
	"SettingsSentry/interfaces"
	"SettingsSentry/logger"
	"SettingsSentry/pkg/backup"
	"SettingsSentry/pkg/command"
	"SettingsSentry/pkg/config"
	"SettingsSentry/pkg/printer"
	"SettingsSentry/pkg/util"
	"embed"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
)

//go:embed configs/*.cfg
var embeddedConfigsFiles embed.FS

var (
	appLogger *logger.Logger = &logger.Logger{}
	Version   string         = "1.1.7"
)

func main() {
	logFilePath := flag.String("logfile", "", "Optional: Path to log file. If provided, logs will be written to this file in addition to console output")

	flag.Parse() // Preliminary parse for log file path

	var err error
	appLogger, err = logger.NewLogger(*logFilePath)
	if err != nil {
		fmt.Printf("Error initializing logger: %v\n", err)
		os.Exit(1)
	}
	config.AppLogger = appLogger
	backup.AppLogger = appLogger

	defer func() {
		if r := recover(); r != nil {
			appLogger.Logf("Panic recovered in main: %v\nStack trace: %s", r, string(debug.Stack()))
			os.Exit(1)
		}
	}()

	appLogger.Logf("SettingsSentry v%s", Version)

	osFs := interfaces.NewOsFileSystem()
	osCmdExecutor := interfaces.NewOsCommandExecutor()

	util.InitGlobals(appLogger, osFs, osCmdExecutor, false)

	config.Fs = util.Fs
	backup.Fs = util.Fs
	command.CmdExecutor = util.CmdExecutor
	printer.AppLogger = util.AppLogger

	var icloud_path string
	icloud_path, err = config.GetICloudFolderLocation()
	if err != nil {
		appLogger.Logf("Error: iCloud path not found - %v", err)
	} else {
		icloud_path = filepath.Join(icloud_path, "settingssentry_backups")
	}
	if icloud_path == "" {
		homeDir, homeErr := config.GetHomeDirectory()
		if homeErr == nil {
			icloud_path = filepath.Join(homeDir, ".settingssentry_backups")
			appLogger.Logf("Warning: iCloud path not found, using default backup path: %s", icloud_path)
		} else {
			appLogger.Logf("Fatal: Cannot determine iCloud or home directory for default backup path.")
			os.Exit(1)
		}
	}

	// Get environment variable defaults
	envConfigFolder := util.GetEnvWithDefault("SETTINGSSENTRY_CONFIG", "configs")
	envBackupFolder := util.GetEnvWithDefault("SETTINGSSENTRY_BACKUP", icloud_path)
	envAppName := os.Getenv("SETTINGSSENTRY_APP")
	envCommands := os.Getenv("SETTINGSSENTRY_COMMANDS") == "true"
	envDryRun := os.Getenv("SETTINGSSENTRY_DRY_RUN") == "true"
	envZip := os.Getenv("SETTINGSSENTRY_ZIP") == "true"
	envPassword := os.Getenv("SETTINGSSENTRY_PASSWORD")

	// Define showHelp function (needs env vars)
	showHelp := func() {
		appLogger.Logf("Usage: SettingsSentry <action> [-config=<path>] [-backup=<path>] [-app=<app1,app2,...>] [-commands] [-logfile=<path>] [-dry-run]")
		appLogger.Logf("")
		appLogger.Logf("Actions:")
		appLogger.Logf("  backup      - Backup configuration files to the specified backup folder")
		appLogger.Logf("  restore     - Restore the files to their original locations")
		appLogger.Logf("  configsinit - Extract embedded default configs to a 'configs' directory next to the executable")
		appLogger.Logf("  install     - Install the application as a CRON job that runs at every reboot (you can also provide a valid cron expression as parameter)")
		appLogger.Logf("  remove      - Remove the previously installed CRON job")
		appLogger.Logf("")
		appLogger.Logf("Use -logfile=<path> to enable logging to a file. This will write logs to the specified file in addition to console output.")
		appLogger.Logf("If -logfile is not provided, logs will only be written to the console.")
		appLogger.Logf("")
		appLogger.Logf("Default values:")
		appLogger.Logf("  Configurations: %s", envConfigFolder)
		appLogger.Logf("  Backups: %s", envBackupFolder)
		appLogger.Logf("")
		appLogger.Logf("Documentation available at https://github.com/sstraus/SettingsSentry")
		appLogger.Logf("")

		installed, err := cronjob.IsCronJobInstalled()
		if err != nil {
			appLogger.Logf("Error checking CRON job installation: %v", err)
			return
		}

		if installed {
			appLogger.Logf("CRON job is currently installed - the application will perform backups at every reboot")
		}
	}

	flag.Usage = showHelp // Set global usage

	// Check for general help flags before parsing action-specific flags
	if len(os.Args) < 2 || (len(os.Args) == 2 && (os.Args[1] == "-h" || os.Args[1] == "--help")) {
		showHelp()
		return
	}

	action := os.Args[1] // Action is the first argument

	// Define flags using a specific FlagSet for the action
	actionFlags := flag.NewFlagSet(action, flag.ExitOnError)
	configFolder := actionFlags.String("config", envConfigFolder, "Path to the configuration folder (env: SETTINGSSENTRY_CONFIG)")
	backupFolder := actionFlags.String("backup", envBackupFolder, "Path to the backup folder (env: SETTINGSSENTRY_BACKUP)")
	appNameFlag := actionFlags.String("app", envAppName, "Optional: Comma-separated list of application names to process (env: SETTINGSSENTRY_APP)")
	commands := actionFlags.Bool("commands", envCommands, "Optional: Prevent pre-backup/restore commands execution (env: SETTINGSSENTRY_COMMANDS)")
	dryRunFlag := actionFlags.Bool("dry-run", envDryRun, "Optional: Perform a dry run without making any changes (env: SETTINGSSENTRY_DRY_RUN)")
	versionsToKeep := actionFlags.Int("versions", 1, "Number of backup versions to keep")
	password := actionFlags.String("password", envPassword, "Optional: Password to encrypt/decrypt backups (env: SETTINGSSENTRY_PASSWORD)")
	zipFlag := actionFlags.Bool("zip", envZip, "Optional: Create backup as a zip archive instead of a directory (env: SETTINGSSENTRY_ZIP)")
	// Add logFilePath flag to this set as well, using the value parsed earlier
	_ = actionFlags.String("logfile", *logFilePath, "Optional: Path to log file.")

	// Parse arguments starting from the one after the action
	if err := actionFlags.Parse(os.Args[2:]); err != nil {
		appLogger.Logf("Error parsing flags for action '%s': %v\n", action, err)
		showHelp()
		return
	}

	// Handle -h/--help specifically after parsing action flags (in case it's the only arg after action)
	// This check might be redundant if flag.ExitOnError handles it, but added for clarity.
	if len(os.Args) == 3 && (os.Args[2] == "-h" || os.Args[2] == "--help") {
		showHelp()
		return
	}

	util.DryRun = *dryRunFlag
	backup.DryRun = util.DryRun

	// Split the appNameFlag string into a slice
	var appNames []string
	if *appNameFlag != "" {
		appNames = strings.Split(*appNameFlag, ",")
		// Trim whitespace from each app name
		for i := range appNames {
			appNames[i] = strings.TrimSpace(appNames[i])
		}
	}

	mainPrinter := printer.NewPrinter("", appLogger)
	backup.Printer = mainPrinter
	command.Printer = mainPrinter

	switch action {
	case "backup":
		backup.ProcessConfiguration(*configFolder, *backupFolder, appNames, true, *commands, *versionsToKeep, *zipFlag, *password)
	case "restore":
		backup.ProcessConfiguration(*configFolder, *backupFolder, appNames, false, *commands, *versionsToKeep, false, *password)
	case "configsinit":
		err := util.ExtractEmbeddedConfigs(embeddedConfigsFiles)
		if err != nil {
			appLogger.Logf("Error extracting embedded configs: %v", err)
			os.Exit(1)
		}
	case "install":
		cronExpression := ""
		// Check remaining non-flag arguments for cron expression
		nonFlagArgs := actionFlags.Args()
		if len(nonFlagArgs) > 0 {
			cronExpression = nonFlagArgs[0]
		}
		err := cronjob.InstallCronJob(cronExpression)
		if err != nil {
			appLogger.Logf("Failed to install cron job: %v", err)
			os.Exit(1)
		}
		appLogger.Logf("CRON job installed successfully")
	case "remove":
		err := cronjob.RemoveCronJob()
		if err != nil {
			appLogger.Logf("Failed to remove cron job: %v", err)
			os.Exit(1)
		}
		appLogger.Logf("CRON job removed successfully")
	default:
		appLogger.Logf("Invalid action specified. Please use one of the following: 'backup', 'restore', 'configsinit', 'install', or 'remove'")
		os.Exit(1)
	}
}
