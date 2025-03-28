package cronjob

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime/debug"
	"strings"

	"github.com/robfig/cron"
)

var comment = "# SettingsSentry cron job"

// safeExecute executes a function with panic recovery
func safeExecute(operation string, fn func() error) error {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Panic recovered in %s: %v\nStack trace: %s", operation, r, string(debug.Stack()))
		}
	}()

	return fn()
}

// AddCronJob adds a new cron job for the current executable
func AddCronJob(schedule *string, command string) error {
	return safeExecute("AddCronJob", func() error {
		var job string
		if schedule != nil && *schedule != "@reboot" {
			_, err := cron.ParseStandard(*schedule)
			if err != nil {
				return fmt.Errorf("invalid cron job schedule: %w", err)
			}
		}
		job = fmt.Sprintf("%s %s %s", *schedule, command, comment)

		cmd := exec.Command("crontab", "-l")
		var out bytes.Buffer
		cmd.Stdout = &out
		err := cmd.Run()

		var crontab string
		if err != nil {
			fmt.Println("No existing crontab. Creating a new one.")
			crontab = job + "\n"
		} else {
			crontab = out.String() + job + "\n"
		}

		cmd = exec.Command("crontab", "-")
		cmd.Stdin = bytes.NewBufferString(crontab)
		err = cmd.Run()
		if err != nil {
			return fmt.Errorf("failed to update crontab: %w", err)
		}

		fmt.Println("Cron job added successfully.")
		return nil
	})
}

// RemoveCronJob removes a specific cron job containing the given identifier.
func RemoveCronJob() error {
	return safeExecute("RemoveCronJob", func() error {
		cmd := exec.Command("crontab", "-l")
		var out bytes.Buffer
		cmd.Stdout = &out
		err := cmd.Run()
		if err != nil {
			if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
				fmt.Println("No existing crontab to modify.")
				return nil
			}
			return fmt.Errorf("failed to list crontab: %w", err)
		}

		lines := strings.Split(out.String(), "\n")
		var updatedLines []string
		for _, line := range lines {
			if !strings.Contains(line, comment) {
				updatedLines = append(updatedLines, line)
			}
		}

		if len(lines) != len(updatedLines) {
			newCrontab := strings.Join(updatedLines, "\n")
			cmd = exec.Command("crontab", "-")
			cmd.Stdin = bytes.NewBufferString(newCrontab)
			err = cmd.Run()
			if err != nil {
				return fmt.Errorf("failed to update crontab: %w", err)
			}
			fmt.Println("Cron job removed successfully.")
		} else {
			fmt.Println("No matching cron job found.")
		}

		return nil
	})
}

// IsCronJobInstalled checks if a specific cron job with the given identifier exists.
func IsCronJobInstalled() (bool, error) {
	var installed bool
	err := safeExecute("IsCronJobInstalled", func() error {
		cmd := exec.Command("crontab", "-l")
		var out bytes.Buffer
		cmd.Stdout = &out
		err := cmd.Run()
		if err != nil {
			if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
				installed = false
				return nil
			}
			return fmt.Errorf("failed to list crontab: %w", err)
		}

		lines := strings.Split(out.String(), "\n")
		for _, line := range lines {
			if strings.Contains(line, comment) {
				installed = true
				return nil
			}
		}
		installed = false
		return nil
	})

	return installed, err
}

// InstallCronJob installs a cron job for the application
func InstallCronJob(cronExpression string) error {
	when := "@reboot"
	if cronExpression != "" {
		when = cronExpression
	}

	exePath, err := exec.LookPath("settingssentry")
	if err != nil {
		return fmt.Errorf("failed to find SettingsSentry executable: %w", err)
	}

	command := fmt.Sprintf("%s backup", exePath)

	return AddCronJob(&when, command)
}
