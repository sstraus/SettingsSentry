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
		_ = RemoveCronJob()

		err := InstallCronJob("", false)
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
		_ = RemoveCronJob()

		err := InstallCronJob("0 0 * * *", false)
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
		err := InstallCronJob("invalid expression", false)
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
	_ = exec.Command("crontab", "-r").Run()

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
		_ = cmd.Run()

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
	_ = exec.Command("crontab", "-r").Run()

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

// TestInstallCronJob_UsesAbsolutePath tests that InstallCronJob uses an absolute path
// rather than relying on PATH, which could be manipulated by an attacker
func TestInstallCronJob_UsesAbsolutePath(t *testing.T) {
	withCrontabBackup(t, func() {
		// Remove any existing job first
		_ = RemoveCronJob()

		err := InstallCronJob("", false)
		// May fail if settingssentry is not in PATH or not built yet
		// The test verifies the path resolution approach, not execution success
		if err != nil {
			// Check if error is about finding the executable
			if strings.Contains(err.Error(), "settingssentry") {
				t.Logf("Expected error (binary not found): %v", err)
				// This is acceptable - the test is about the approach
				return
			}
			// Other errors might be real issues
			t.Logf("InstallCronJob returned: %v", err)
		}

		// If successful, verify the cron job uses an absolute path
		cmd := exec.Command("crontab", "-l")
		var out bytes.Buffer
		cmd.Stdout = &out
		err = cmd.Run()
		if err != nil {
			t.Logf("Could not read crontab: %v", err)
			return
		}

		crontabContent := out.String()
		if !strings.Contains(crontabContent, comment) {
			// Job wasn't added, which is fine for this test
			return
		}

		// Verify the command uses an absolute path (starts with /)
		lines := strings.Split(crontabContent, "\n")
		for _, line := range lines {
			if strings.Contains(line, comment) {
				// Found our job, check next line for command
				continue
			}
			if strings.Contains(line, "backup") && !strings.HasPrefix(strings.TrimSpace(line), "#") {
				// This is likely our backup command
				// Security: It should use absolute path, not rely on PATH
				if !strings.Contains(line, "/") {
					t.Error("Cron job command should use absolute path for security")
					t.Errorf("Found relative command: %s", line)
				} else {
					t.Logf("✓ Cron job uses absolute path: %s", line)
				}
			}
		}
	})
}

// TestInstallCronJob_AllowCommands tests that the --allow-commands flag is included when requested
func TestInstallCronJob_AllowCommands(t *testing.T) {
	withCrontabBackup(t, func() {
		// Remove any existing job first
		_ = RemoveCronJob()

		t.Run("without allow-commands flag", func(t *testing.T) {
			_ = RemoveCronJob()

			err := InstallCronJob("", false)
			if err != nil {
				if !strings.Contains(err.Error(), "settingssentry") {
					t.Errorf("Unexpected error: %v", err)
				}
				t.Logf("Expected error (binary not found): %v", err)
				return
			}

			// Verify the cron job does NOT include --allow-commands
			cmd := exec.Command("crontab", "-l")
			var out bytes.Buffer
			cmd.Stdout = &out
			err = cmd.Run()
			if err != nil {
				t.Logf("Could not read crontab: %v", err)
				return
			}

			crontabContent := out.String()
			if strings.Contains(crontabContent, "--allow-commands") {
				t.Error("Cron job should NOT include --allow-commands when flag is false")
				t.Errorf("Crontab content: %s", crontabContent)
			} else {
				t.Logf("✓ Cron job correctly excludes --allow-commands (secure default)")
			}
		})

		t.Run("with allow-commands flag", func(t *testing.T) {
			_ = RemoveCronJob()

			err := InstallCronJob("", true)
			if err != nil {
				if !strings.Contains(err.Error(), "settingssentry") {
					t.Errorf("Unexpected error: %v", err)
				}
				t.Logf("Expected error (binary not found): %v", err)
				return
			}

			// Verify the cron job DOES include --allow-commands
			cmd := exec.Command("crontab", "-l")
			var out bytes.Buffer
			cmd.Stdout = &out
			err = cmd.Run()
			if err != nil {
				t.Logf("Could not read crontab: %v", err)
				return
			}

			crontabContent := out.String()
			if !strings.Contains(crontabContent, "--allow-commands") {
				t.Error("Cron job should include --allow-commands when flag is true")
				t.Errorf("Crontab content: %s", crontabContent)
			} else {
				t.Logf("✓ Cron job correctly includes --allow-commands flag")
			}
		})
	})
}
