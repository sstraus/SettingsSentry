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
)

func main() {
	// Parse command-line flags first to get the logFilePath
	logFilePath := flag.String("logfile", "", "Optional: Path to log file. If provided, logs will be written to this file in addition to console output")

	// We need to do a preliminary parse to get the logFilePath
	flag.Parse()

	// Initialize the logger properly
	var err error
	appLogger, err = logger.NewLogger(*logFilePath)
	if err != nil {
		fmt.Printf("Error initializing logger: %v\n", err)
		os.Exit(1)
	}
	config.AppLogger = appLogger
	backup.AppLogger = appLogger

	// Add panic recovery for the main function
	defer func() {
		if r := recover(); r != nil {
			appLogger.Logf("Panic recovered in main: %v\nStack trace: %s", r, string(debug.Stack()))
			os.Exit(1)
		}
	}()

	Version := "1.1.5"
	appLogger.Logf("SettingsSentry v%s", Version)

	// Initialize dependencies
	osFs := interfaces.NewOsFileSystem()
	osCmdExecutor := interfaces.NewOsCommandExecutor()

	// Initialize globals in util package and dependent packages
	util.InitGlobals(appLogger, osFs, osCmdExecutor, false, Version)

	// Manually set dependencies in other packages
	config.Fs = util.Fs
	backup.Fs = util.Fs
	command.CmdExecutor = util.CmdExecutor
	printer.AppLogger = util.AppLogger // Printer needs logger

	var icloud_path string
	icloud_path, err = config.GetICloudFolderLocation()
	if err != nil {
		appLogger.Logf("Error: iCloud path not found - %v", err)
	} else {
		icloud_path = filepath.Join(icloud_path, "settingssentry_backups")
	}
	// Handle case where iCloud path failed but we might still proceed
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

	envConfigFolder := util.GetEnvWithDefault("SETTINGSSENTRY_CONFIG", "configs")
	envBackupFolder := util.GetEnvWithDefault("SETTINGSSENTRY_BACKUP", icloud_path)
	envAppName := os.Getenv("SETTINGSSENTRY_APP")
	envCommands := os.Getenv("SETTINGSSENTRY_COMMANDS") == "true"
	envDryRun := os.Getenv("SETTINGSSENTRY_DRY_RUN") == "true"

	// Define shared command-line flags with environment variable defaults
	configFolder := flag.String("config", envConfigFolder, "Path to the configuration folder (env: SETTINGSSENTRY_CONFIG)")
	backupFolder := flag.String("backup", envBackupFolder, "Path to the backup folder (env: SETTINGSSENTRY_BACKUP)")
	appName := flag.String("app", envAppName, "Optional: Name of the application to process (env: SETTINGSSENTRY_APP)")
	commands := flag.Bool("commands", envCommands, "Optional: Prevent pre-backup/restore commands execution (env: SETTINGSSENTRY_COMMANDS)")
	dryRunFlag := flag.Bool("dry-run", envDryRun, "Optional: Perform a dry run without making any changes (env: SETTINGSSENTRY_DRY_RUN)")
	versionsToKeep := flag.Int("versions", 1, "Number of backup versions to keep")

	// Define a function to display help information
	showHelp := func() {
		appLogger.Logf("Usage: SettingsSentry <action> [-config=<path>] [-backup=<path>] [-app=<n>] [-commands] [-logfile=<path>] [-dry-run]")
		appLogger.Logf("Actions:")
		appLogger.Logf("  backup      - Backup configuration files to the specified backup folder")
		appLogger.Logf("  restore     - Restore the files to their original locations")
		appLogger.Logf("  configsinit - Extract embedded default configs to a 'configs' directory next to the executable")
		appLogger.Logf("  install     - Install the application as a CRON job that runs at every reboot (you can also provide a valid cron expression as parameter)")
		appLogger.Logf("  remove      - Remove the previously installed CRON job")
		appLogger.Logf("Use -logfile=<path> to enable logging to a file. This will write logs to the specified file in addition to console output.")
		appLogger.Logf("If -logfile is not provided, logs will only be written to the console.")
		appLogger.Logf("Default values:")
		appLogger.Logf("  Configurations: %s", envConfigFolder)
		appLogger.Logf("  Backups: %s", envBackupFolder)
		appLogger.Logf("Documentation available at https://github.com/sstraus/SettingsSentry")

		installed, err := cronjob.IsCronJobInstalled()
		if err != nil {
			appLogger.Logf("Error checking CRON job installation: %v", err)
			return
		}

		if installed {
			appLogger.Logf("CRON job is currently installed - the application will perform backups at every reboot")
		}
	}

	// Override the default flag.Usage function to use our custom help
	flag.Usage = showHelp

	// Check if no arguments or -h flag is provided
	if len(os.Args) < 2 || (len(os.Args) == 2 && (os.Args[1] == "-h" || os.Args[1] == "--help")) {
		showHelp()
		return
	}

	// Check if the second argument is -h or --help
	if len(os.Args) >= 3 && (os.Args[2] == "-h" || os.Args[2] == "--help") {
		showHelp()
		return
	}

	// Parse flags for custom handling based on the specified action
	if err := flag.CommandLine.Parse(os.Args[2:]); err != nil {
		appLogger.Logf("Error parsing flags: %v\n", err)
		return
	}
	util.DryRun = *dryRunFlag
	backup.DryRun = util.DryRun

	action := os.Args[1]

	mainPrinter := printer.NewPrinter("", appLogger)
	backup.Printer = mainPrinter
	command.Printer = mainPrinter

	switch action {
	case "backup":
		backup.ProcessConfiguration(*configFolder, *backupFolder, *appName, true, *commands, *versionsToKeep)
	case "restore":
		backup.ProcessConfiguration(*configFolder, *backupFolder, *appName, false, *commands, *versionsToKeep)
	case "configsinit":
		err := util.ExtractEmbeddedConfigs(embeddedConfigsFiles)
		if err != nil {
			appLogger.Logf("Error extracting embedded configs: %v", err)
			os.Exit(1)
		}
	case "install":
		cronExpression := ""
		if len(os.Args) > 2 && !strings.HasPrefix(os.Args[2], "-") {
			cronExpression = os.Args[2]
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
