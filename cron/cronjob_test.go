package cronjob

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

func containsCronJob(crontab string) bool {
	return strings.Contains(crontab, comment)
}

func withCrontabBackup(t *testing.T, testFunc func()) {
	cmd := exec.Command("crontab", "-l")
	var backupBuffer bytes.Buffer
	cmd.Stdout = &backupBuffer
	err := cmd.Run()

	// If there's no crontab, that's fine
	var originalCrontab string
	if err == nil {
		originalCrontab = backupBuffer.String()
	}

	testFunc()

	cmd = exec.Command("crontab", "-")
	cmd.Stdin = strings.NewReader(originalCrontab)
	if err := cmd.Run(); err != nil {
		t.Logf("Warning: Failed to restore original crontab: %v", err)
	}
}

func TestAddCronJob(t *testing.T) {
	withCrontabBackup(t, func() {
		schedule := "@reboot"
		err := AddCronJob(&schedule, "-commands")
		if err != nil {
			t.Errorf("AddCronJob() returned an error: %v", err)
		}

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
		schedule := "@reboot"
		err := AddCronJob(&schedule, "-commands")
		if err != nil {
			t.Errorf("AddCronJob() returned an error: %v", err)
		}

		if err := RemoveCronJob(); err != nil {
			t.Fatalf("Failed to remove cron job: %v", err)
		}

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
		if err := RemoveCronJob(); err != nil {
			t.Fatalf("Failed to remove cron job: %v", err)
		}

		installed, err := IsCronJobInstalled()
		if err != nil {
			t.Errorf("IsCronJobInstalled() returned an error: %v", err)
		}
		if installed {
			t.Errorf("IsCronJobInstalled() returned true when no job was installed")
		}

		schedule := "@reboot"
		err = AddCronJob(&schedule, "-commands")
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

func TestAddCronJobWithInvalidSchedule(t *testing.T) {
	withCrontabBackup(t, func() {
		schedule := "invalid cron"
		err := AddCronJob(&schedule, "-commands")
		if err == nil {
			t.Errorf("AddCronJob() did not return an error for invalid cron expression")
		}
	})
}
