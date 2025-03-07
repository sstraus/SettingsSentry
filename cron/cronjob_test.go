package cronjob

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

// Helper function to check if a string contains the cron job comment
func containsCronJob(crontab string) bool {
	return strings.Contains(crontab, comment)
}

// Helper function to backup and restore crontab
func withCrontabBackup(t *testing.T, testFunc func()) {
	// Backup current crontab
	cmd := exec.Command("crontab", "-l")
	var backupBuffer bytes.Buffer
	cmd.Stdout = &backupBuffer
	err := cmd.Run()

	// If there's no crontab, that's fine
	var originalCrontab string
	if err == nil {
		originalCrontab = backupBuffer.String()
	}

	// Run the test
	testFunc()

	// Restore original crontab
	cmd = exec.Command("crontab", "-")
	cmd.Stdin = strings.NewReader(originalCrontab)
	if err := cmd.Run(); err != nil {
		t.Logf("Warning: Failed to restore original crontab: %v", err)
	}
}

func TestAddCronJob(t *testing.T) {
	withCrontabBackup(t, func() {
		// Test adding a cron job
		schedule := "@reboot"
		err := AddCronJob(&schedule, "-nocommands")
		if err != nil {
			t.Errorf("AddCronJob() returned an error: %v", err)
		}

		// Verify the cron job was added
		cmd := exec.Command("crontab", "-l")
		var out bytes.Buffer
		cmd.Stdout = &out
		err = cmd.Run()
		if err != nil {
			t.Errorf("Failed to list crontab: %v", err)
		}

		if !containsCronJob(out.String()) {
			t.Errorf("Cron job was not added to crontab")
		}
	})
}

func TestRemoveCronJob(t *testing.T) {
	withCrontabBackup(t, func() {
		// First add a cron job
		schedule := "@reboot"
		err := AddCronJob(&schedule, "-nocommands")
		if err != nil {
			t.Errorf("AddCronJob() returned an error: %v", err)
		}

		// Then remove it
		if err := RemoveCronJob(); err != nil {
			t.Fatalf("Failed to remove cron job: %v", err)
		}

		// Verify the cron job was removed
		cmd := exec.Command("crontab", "-l")
		var out bytes.Buffer
		cmd.Stdout = &out
		err = cmd.Run()

		// If there's no crontab, that means it was removed completely
		if err == nil && containsCronJob(out.String()) {
			t.Errorf("Cron job was not removed from crontab")
		}
	})
}

func TestIsCronJobInstalled(t *testing.T) {
	withCrontabBackup(t, func() {
		// First check when no cron job is installed
		if err := RemoveCronJob(); err != nil {
			t.Fatalf("Failed to remove cron job: %v", err)
		} // Make sure no job exists

		installed, err := IsCronJobInstalled()
		if err != nil {
			t.Errorf("IsCronJobInstalled() returned an error: %v", err)
		}
		if installed {
			t.Errorf("IsCronJobInstalled() returned true when no job was installed")
		}

		// Now add a cron job and check again
		schedule := "@reboot"
		err = AddCronJob(&schedule, "-nocommands")
		if err != nil {
			t.Errorf("AddCronJob() returned an error: %v", err)
		}

		installed, err = IsCronJobInstalled()
		if err != nil {
			t.Errorf("IsCronJobInstalled() returned an error: %v", err)
		}
		if !installed {
			t.Errorf("IsCronJobInstalled() returned false when a job was installed")
		}
	})
}

// Test invalid cron expression
func TestAddCronJobWithInvalidSchedule(t *testing.T) {
	withCrontabBackup(t, func() {
		// Test with an invalid cron expression
		schedule := "invalid cron"
		err := AddCronJob(&schedule, "-nocommands")
		if err == nil {
			t.Errorf("AddCronJob() did not return an error for invalid cron expression")
		}
	})
}
