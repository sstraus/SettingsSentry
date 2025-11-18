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

func TestInstallCronJob(t *testing.T) {
	withCrontabBackup(t, func() {
		// Remove any existing job first
		RemoveCronJob()

		err := InstallCronJob("")
		// May fail if settingssentry is not in PATH, which is expected
		if err != nil {
			if !strings.Contains(err.Error(), "settingssentry") {
				t.Errorf("InstallCronJob() returned unexpected error: %v", err)
			} else {
				t.Logf("Expected error (settingssentry not in PATH): %v", err)
			}
		}
	})
}

func TestInstallCronJob_WithExpression(t *testing.T) {
	withCrontabBackup(t, func() {
		// Remove any existing job first
		RemoveCronJob()

		err := InstallCronJob("0 0 * * *")
		// May fail if settingssentry is not in PATH
		if err != nil {
			if !strings.Contains(err.Error(), "settingssentry") {
				t.Errorf("InstallCronJob() with expression returned unexpected error: %v", err)
			} else {
				t.Logf("Expected error (settingssentry not in PATH): %v", err)
			}
		}
	})
}

func TestInstallCronJob_InvalidExpression(t *testing.T) {
	withCrontabBackup(t, func() {
		err := InstallCronJob("invalid expression")
		if err == nil {
			t.Error("InstallCronJob() should return error for invalid cron expression")
		}
	})
}

func TestSafeExecute_Success(t *testing.T) {
	called := false
	err := safeExecute("test operation", func() error {
		called = true
		return nil
	})

	if err != nil {
		t.Errorf("safeExecute() returned error: %v", err)
	}
	if !called {
		t.Error("Function was not called")
	}
}

func TestSafeExecute_WithError(t *testing.T) {
	expectedErr := exec.Command("false").Run() // Command that fails
	err := safeExecute("test operation", func() error {
		return expectedErr
	})

	if err == nil {
		t.Error("safeExecute() should return the error from function")
	}
}

func TestSafeExecute_WithPanic(t *testing.T) {
	// Should recover from panic and not crash
	err := safeExecute("panic operation", func() error {
		panic("test panic")
	})

	// After panic recovery, function returns normally
	// The panic is caught and logged, but doesn't return an error
	if err != nil {
		t.Logf("safeExecute returned: %v", err)
	}
}

func TestAddCronJob_AtReboot(t *testing.T) {
	withCrontabBackup(t, func() {
		schedule := "@reboot"
		err := AddCronJob(&schedule, "test command")
		if err != nil {
			t.Errorf("AddCronJob() with @reboot failed: %v", err)
		}

		// Verify it was added
		cmd := exec.Command("crontab", "-l")
		var out bytes.Buffer
		cmd.Stdout = &out
		err = cmd.Run()
		if err != nil {
			t.Fatalf("Failed to list crontab: %v", err)
		}

		if !strings.Contains(out.String(), "@reboot") {
			t.Error("@reboot schedule was not added to crontab")
		}
		if !strings.Contains(out.String(), "test command") {
			t.Error("Command was not added to crontab")
		}
	})
}

func TestAddCronJob_ValidCronExpression(t *testing.T) {
	withCrontabBackup(t, func() {
		schedule := "0 0 * * *"
		err := AddCronJob(&schedule, "daily command")
		if err != nil {
			t.Errorf("AddCronJob() with valid expression failed: %v", err)
		}

		cmd := exec.Command("crontab", "-l")
		var out bytes.Buffer
		cmd.Stdout = &out
		err = cmd.Run()
		if err != nil {
			t.Fatalf("Failed to list crontab: %v", err)
		}

		if !strings.Contains(out.String(), "0 0 * * *") {
			t.Error("Cron expression was not added to crontab")
		}
	})
}

func TestRemoveCronJob_NoCrontab(t *testing.T) {
	// First, remove all cron jobs to ensure clean state
	exec.Command("crontab", "-r").Run()

	// Should not error when there's no crontab
	err := RemoveCronJob()
	if err != nil {
		t.Errorf("RemoveCronJob() should not error when no crontab exists: %v", err)
	}
}

func TestRemoveCronJob_NoMatchingJob(t *testing.T) {
	withCrontabBackup(t, func() {
		// Add a different cron job that doesn't match our comment
		cmd := exec.Command("crontab", "-")
		cmd.Stdin = strings.NewReader("0 0 * * * echo 'other job'\n")
		cmd.Run()

		err := RemoveCronJob()
		if err != nil {
			t.Errorf("RemoveCronJob() returned error: %v", err)
		}

		// The other job should still exist
		cmd = exec.Command("crontab", "-l")
		var out bytes.Buffer
		cmd.Stdout = &out
		err = cmd.Run()
		if err != nil {
			t.Fatalf("Failed to list crontab: %v", err)
		}

		if !strings.Contains(out.String(), "other job") {
			t.Error("Other cron jobs should not be removed")
		}
	})
}

func TestIsCronJobInstalled_NoCrontab(t *testing.T) {
	// Remove all cron jobs
	exec.Command("crontab", "-r").Run()

	installed, err := IsCronJobInstalled()
	if err != nil {
		t.Errorf("IsCronJobInstalled() returned error: %v", err)
	}
	if installed {
		t.Error("IsCronJobInstalled() should return false when no crontab exists")
	}
}

func TestCronComment(t *testing.T) {
	// Verify the comment constant
	if comment == "" {
		t.Error("comment constant should not be empty")
	}
	if !strings.Contains(comment, "SettingsSentry") {
		t.Error("comment should contain 'SettingsSentry'")
	}
}
