package cronjob

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/robfig/cron"
)

var comment = "# SettingsSentry cron job"

// AddCronJob adds a new cron job for the current executable
func AddCronJob(schedule *string, params string) error {
	// Get the current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	execPath, err = filepath.Abs(execPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Define the cron job with the current executable
	//job := fmt.Sprintf("0 9 * * * %s %s", execPath, comment)
	var job string
	if schedule != nil && *schedule != "@reboot" {
		_, err = cron.ParseStandard(*schedule)
		if err != nil {
			return fmt.Errorf("invalid cron job schedule: %w", err)
		}
	}
	job = fmt.Sprintf("%s %s backup %s %s", *schedule, execPath, params, comment)
	// Get the existing crontab
	cmd := exec.Command("crontab", "-l")
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()

	var crontab string
	if err != nil {
		// If the crontab is empty, start fresh
		fmt.Println("No existing crontab. Creating a new one.")
		crontab = job + "\n"
	} else {
		// Append the new job to the existing crontab
		crontab = out.String() + job + "\n"
	}

	// Write the new crontab
	cmd = exec.Command("crontab", "-")
	cmd.Stdin = bytes.NewBufferString(crontab)
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to update crontab: %w", err)
	}

	fmt.Println("Cron job added successfully.")
	return nil
}

// RemoveCronJob removes a specific cron job containing the given identifier.
func RemoveCronJob() error {
	// Get the current crontab
	cmd := exec.Command("crontab", "-l")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		// If the crontab is empty or doesn't exist
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			fmt.Println("No existing crontab to modify.")
			return nil
		}
		return fmt.Errorf("failed to list crontab: %w", err)
	}

	// Filter out the line with the identifier
	lines := strings.Split(out.String(), "\n")
	var updatedLines []string
	for _, line := range lines {
		if !strings.Contains(line, comment) {
			updatedLines = append(updatedLines, line)
		}
	}

	// Update the crontab if changes were made
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
}

// IsCronJobInstalled checks if a specific cron job with the given identifier exists.
func IsCronJobInstalled() (bool, error) {
	// Get the current crontab
	cmd := exec.Command("crontab", "-l")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		// If no crontab exists, treat it as not installed
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			return false, nil
		}
		return false, fmt.Errorf("failed to list crontab: %w", err)
	}

	// Check if the identifier exists in the crontab
	lines := strings.Split(out.String(), "\n")
	for _, line := range lines {
		if strings.Contains(line, comment) {
			return true, nil
		}
	}
	return false, nil
}
