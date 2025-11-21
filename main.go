package main

import (
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
	"runtime/debug"
)

//go:embed configs/*.cfg
var embeddedConfigsFiles embed.FS

var (
	Version string = "1.1.8"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// run is the main application logic that returns an error instead of calling os.Exit
func run(args []string) error {
	// Parse logfile flag first
	logFilePath := flag.String("logfile", "", "Optional: Path to log file. If provided, logs will be written to this file in addition to console output")
	flag.Parse()

	// Initialize logger
	appLogger, err := logger.NewLogger(*logFilePath)
	if err != nil {
		return fmt.Errorf("error initializing logger: %w", err)
	}

	// Set up panic recovery
	defer func() {
		if r := recover(); r != nil {
			appLogger.Logf("Panic recovered in run: %v\nStack trace: %s", r, string(debug.Stack()))
		}
	}()

	appLogger.Logf("SettingsSentry v%s", Version)

	// Initialize file system and command executor
	osFs := interfaces.NewOsFileSystem()
	osCmdExecutor := interfaces.NewOsCommandExecutor()

	// Initialize global utilities
	util.InitGlobals(appLogger, osFs, osCmdExecutor, false)

	// Set package-level dependencies
	config.AppLogger = appLogger
	config.Fs = util.Fs
	backup.AppLogger = appLogger
	backup.Fs = util.Fs
	command.CmdExecutor = util.CmdExecutor
	printer.AppLogger = util.AppLogger

	// Create CLI instance
	cli := NewCLI(appLogger, osFs, osCmdExecutor, embeddedConfigsFiles, Version)

	// Check for help flags
	if len(args) == 0 || (len(args) == 1 && (args[0] == "-h" || args[0] == "--help")) {
		cli.ShowHelp()
		return nil
	}

	// Parse flags and get action
	action, flags, err := cli.ParseFlags(args)
	if err != nil {
		cli.ShowHelp()
		return fmt.Errorf("flag parsing error: %w", err)
	}

	// Execute the action
	return cli.ExecuteAction(action, flags)
}
