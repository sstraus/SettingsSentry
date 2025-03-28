package command

import (
	"SettingsSentry/interfaces"
	"SettingsSentry/logger"
	"SettingsSentry/pkg/printer"
	"runtime/debug"
)

var (
	AppLogger   *logger.Logger
	Printer     *printer.Printer
	CmdExecutor interfaces.CommandExecutor
)

func SafeExecute(operation string, fn func() error) error {
	defer func() {
		if r := recover(); r != nil {
			if Printer != nil {
				Printer.Print("Panic recovered in %s: %v\nStack trace: %s", operation, r, string(debug.Stack()))
			} else if AppLogger != nil {
				AppLogger.Logf("Panic recovered in %s: %v\nStack trace: %s", operation, r, string(debug.Stack()))
			}
		}
	}()
	return fn()
}

func ExecuteCommandLine(commandLine string) bool {
	if commandLine == "" {
		if Printer != nil {
			Printer.Print("No command provided")
		}
		// No command is not an error
		return true
	}

	if CmdExecutor == nil {
		if AppLogger != nil {
			AppLogger.Logf("Error: Command executor is not initialized.")
		}
		return false
	}

	stdoutHandler := func(line string) {
		if Printer != nil {
			Printer.Print("  → %s", line)
		}
	}

	stderrHandler := func(line string) {
		if Printer != nil {
			Printer.Print("  ⚠ %s", line)
		}
	}

	result := CmdExecutor.ExecuteWithCallback(commandLine, stdoutHandler, stderrHandler)

	if !result {
		if Printer != nil {
			Printer.Print("Error executing command: %s", commandLine)
		} else if AppLogger != nil {
			AppLogger.Logf("Error executing command: %s", commandLine)
		}
		return false
	}

	if Printer != nil {
		Printer.Print("Command executed: %s", commandLine)
	}

	return true
}
