package backup

import (
	"SettingsSentry/interfaces"
	"SettingsSentry/logger"
	"SettingsSentry/pkg/command"
	"SettingsSentry/pkg/config"
	"SettingsSentry/pkg/printer"
	"io"
	iofs "io/fs" // Use alias for standard io/fs
	"os"
	"sort"
	"strings"
	"time"
)

var (
	AppLogger *logger.Logger
	Fs        interfaces.FileSystem
	DryRun    bool
	Printer   *printer.Printer
)

func getVersionedBackupPath(baseBackupPath string, createNew bool) (string, error) {
	if !createNew {
		// For restore operations, just use the latest version
		return GetLatestVersionPath(baseBackupPath)
	}

	// For backup operations, create a new version
	timestamp := time.Now().Format("20060102-150405")
	versionedPath := Fs.Join(baseBackupPath, timestamp)

	_, err := Fs.Stat(versionedPath)
	if os.IsNotExist(err) {
		// Directory doesn't exist, will be created below
	} else if err != nil {
		// Handle other stat errors
		return "", AppLogger.LogErrorf("failed to stat potential versioned backup directory: %w", err)
	}

	// Create the versioned directory if not in dry run
	if !DryRun {
		err := Fs.MkdirAll(versionedPath, 0755)
		if err != nil {
			return "", AppLogger.LogErrorf("failed to create versioned backup directory: %w", err)
		}
	} else {
		AppLogger.Logf("Dry run: Would create versioned backup directory: %s", versionedPath)
	}

	return versionedPath, nil
}

// GetLatestVersionPath returns the path to the latest version
func GetLatestVersionPath(baseBackupPath string) (string, error) {
	// Check if the base path exists
	_, err := Fs.Stat(baseBackupPath)
	if err != nil {
		return "", AppLogger.LogErrorf("backup path does not exist: %w", err)
	}

	// Read the directory entries
	entries, err := Fs.ReadDir(baseBackupPath)
	if err != nil {
		return "", AppLogger.LogErrorf("failed to read backup directory: %w", err)
	}

	// Find the latest version (based on directory name format YYYYMMDD-HHMMSS)
	var latestEntry os.DirEntry
	var latestTime time.Time

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Try to parse the directory name as a timestamp
		t, err := time.Parse("20060102-150405", entry.Name())
		if err != nil {
			// Skip directories that don't match our format
			continue
		}

		if latestEntry == nil || t.After(latestTime) {
			latestEntry = entry
			latestTime = t
		}
	}

	if latestEntry == nil {
		return "", AppLogger.LogErrorf("no version backups found in %s", baseBackupPath)
	}

	return Fs.Join(baseBackupPath, latestEntry.Name()), nil
}

// CleanupOldVersions removes old versions to keep only the specified number
func CleanupOldVersions(baseBackupPath string, maxVersions int) error {
	if maxVersions <= 0 {
		// Keep all versions
		return nil
	}

	// Check if the base path exists
	_, err := Fs.Stat(baseBackupPath)
	if err != nil {
		// If base path doesn't exist, nothing to clean up
		if os.IsNotExist(err) {
			return nil
		}
		return AppLogger.LogErrorf("failed to stat backup path for cleanup: %w", err)
	}

	// Read the directory entries
	entries, err := Fs.ReadDir(baseBackupPath)
	if err != nil {
		return AppLogger.LogErrorf("failed to read backup directory for cleanup: %w", err)
	}

	// Collect all version directories with their timestamps
	type versionInfo struct {
		path      string
		timestamp time.Time
	}

	var versions []versionInfo

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Try to parse the directory name as a timestamp
		t, err := time.Parse("20060102-150405", entry.Name())
		if err != nil {
			// Skip directories that don't match our format
			continue
		}

		versions = append(versions, versionInfo{
			path:      Fs.Join(baseBackupPath, entry.Name()),
			timestamp: t,
		})
	}

	// Sort versions by timestamp (newest first)
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].timestamp.After(versions[j].timestamp)
	})

	// Remove old versions if we have more than maxVersions
	if len(versions) > maxVersions {
		for i := maxVersions; i < len(versions); i++ {
			_, statErr := Fs.Stat(versions[i].path)
			if statErr != nil {
				if os.IsNotExist(statErr) {
					AppLogger.Logf("Skipping version that no longer exists: %s", versions[i].path)
					continue
				}
				// Log other stat errors but attempt to continue cleanup
				AppLogger.Logf("Error stating old version %s: %v", versions[i].path, statErr)
				continue
			}
			if DryRun {
				AppLogger.Logf("Would remove old version: %s", versions[i].path)
			} else {
				AppLogger.Logf("Removing old version: %s", versions[i].path)
				err := Fs.RemoveAll(versions[i].path)
				if err != nil {
					// Log error but continue trying to remove others
					AppLogger.Logf("Failed to remove old version %s: %v", versions[i].path, err)
				}
			}
		}
	}

	return nil
}

// copyFile copies a single file from src to dst.
func copyFile(src, dst string) error {
	if Fs == nil {
		panic("Fs is nil in copyFile!")
	}
	if AppLogger == nil {
		panic("AppLogger is nil in copyFile!")
	}
	srcFile, err := Fs.Open(src)
	if err != nil {
		AppLogger.Logf("copyFile: Fs.Open failed for '%s': %v", src, err) // Log error
		return AppLogger.LogErrorf("failed to open source file '%s': %w", src, err)
	}
	// Defer close *after* checking for error
	defer srcFile.Close()

	// Ensure destination directory exists
	err = Fs.MkdirAll(Fs.Dir(dst), 0755)
	if err != nil {
		return AppLogger.LogErrorf("failed to create destination directory '%s': %w", Fs.Dir(dst), err)
	}

	dstFile, err := Fs.Create(dst)
	if err != nil {
		return AppLogger.LogErrorf("failed to create destination file '%s': %w", dst, err)
	}
	// Defer close *after* checking for error
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return AppLogger.LogErrorf("failed to copy file contents from '%s' to '%s': %w", src, dst, err)
	}
	return nil
}

// copyDirectory recursively copies a directory from src to dst.
func copyDirectory(src, dst string) error {
	// Get file info for the source directory
	srcInfo, err := Fs.Stat(src)
	if err != nil {
		return AppLogger.LogErrorf("failed to get source directory info '%s': %w", src, err)
	}

	// Create the destination directory
	err = Fs.MkdirAll(dst, srcInfo.Mode())
	if err != nil {
		return AppLogger.LogErrorf("failed to create destination directory '%s': %w", dst, err)
	}

	// Read the source directory
	entries, err := Fs.ReadDir(src)
	if err != nil {
		return AppLogger.LogErrorf("failed to read source directory '%s': %w", src, err)
	}

	// Copy each entry
	for _, entry := range entries {
		srcPath := Fs.Join(src, entry.Name())
		dstPath := Fs.Join(dst, entry.Name())

		if entry.IsDir() {
			// Recursively copy subdirectories
			err = copyDirectory(srcPath, dstPath)
			if err != nil {
				return err // Propagate error up
			}
		} else {
			// Copy files
			err = copyFile(srcPath, dstPath)
			if err != nil {
				return err // Propagate error up
			}
		}
	}

	return nil
}

// ProcessConfiguration processes configuration files for backup or restore.
func ProcessConfiguration(configFolder, backupFolder, appName string, isBackup bool, commands bool, versionsToKeep int) {
	// Expand environment variables in paths using config package function
	configFolder = config.ExpandEnvVars(configFolder)
	backupFolder = config.ExpandEnvVars(backupFolder)

	// Check if configFolder does not contain "/"
	if !strings.Contains(configFolder, string(os.PathSeparator)) {
		exePath, err := os.Executable() // Get the path of the executable
		if err != nil {
			AppLogger.Logf("Error getting executable path: %v", err)
			return
		}
		configFolder = Fs.Join(Fs.Dir(exePath), configFolder) // Append to the executable directory
	}

	var currentFS iofs.FS     // Filesystem to use (OS or embedded)
	var files []iofs.DirEntry // Directory entries from iofs.ReadDir
	var configReadDir string  // Directory path/source used for ReadDir (for logging)

	// Validate the config folder exists using the OS filesystem interface
	_, err := Fs.Stat(configFolder)
	if err != nil {
		// Config folder doesn't exist or isn't accessible via OS fs, try embedded
		AppLogger.Logf("Config folder '%s' not found or inaccessible: %v. Attempting to use embedded configs.", configFolder, err)
		AppLogger.Logf("Error: Embedded fallback logic needs rootEmbedFS passed.") // Temporary error
		return                                                                     // Exit until dependency is passed

	} else {
		// Config folder exists, use the OS filesystem
		AppLogger.Logf("Using config folder: %s", configFolder)
		currentFS = os.DirFS(configFolder) // Create an iofs.FS from the OS path
		configReadDir = configFolder
		// Read directory entries from the OS filesystem folder using iofs.ReadDir
		files, err = iofs.ReadDir(currentFS, ".") // Use iofs.ReadDir
		if err != nil {
			AppLogger.Logf("Error reading config folder '%s': %v", configFolder, err)
			return // Cannot proceed if OS folder is unreadable
		}
		if len(files) == 0 {
			AppLogger.Logf("No configuration files found in '%s'.", configFolder)
			return
		}
	}

	// Validate the backup folder exists or create it (using OS fs)
	if isBackup {
		if DryRun {
			AppLogger.Logf("Would create backup folder: %s", backupFolder)
		} else {
			err = Fs.MkdirAll(backupFolder, 0755) // Use global fs for OS operation
			if err != nil {
				AppLogger.Logf("Failed to create backup folder: %v", err)
				return
			}
		}
	} else {
		// For restore, the backup folder must exist (check using OS fs)
		_, err := Fs.Stat(backupFolder) // Use global fs for OS operation
		if err != nil {
			AppLogger.Logf("Backup folder does not exist or is not accessible: %v", err)
			return
		}
	}

	homeDir, err := config.GetHomeDirectory() // Use config package function
	if err != nil {
		AppLogger.Logf("Error getting home directory: %v", err)
		return
	}

	// Create timestamp here to avoid different paths during the backup
	timestamp := time.Now().Format("20060102-150405")

	foundCfg := false // Flag to track if any .cfg files were processed
	for _, file := range files {
		// Skip directories and non-cfg files
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".cfg") {
			continue
		}
		// Runtime skip removed as per user request. Exclusion handled externally.
		foundCfg = true // Found at least one .cfg file

		// If an app name is specified, only process that app's configuration
		if appName != "" && !strings.Contains(strings.ToLower(file.Name()), strings.ToLower(appName)) {
			continue
		}

		// Parse the configuration file using the determined filesystem (currentFS)
		// file.Name() is relative to the directory read by iofs.ReadDir
		cfg, err := config.ParseConfig(currentFS, file.Name()) // Use config package function
		if err != nil {
			AppLogger.Logf("Error parsing configuration file %s from %s: %v", file.Name(), configReadDir, err)
			continue
		}

		// Use Printer from printer package
		if Printer == nil {
			AppLogger.Logf("Error: Printer is not initialized.")
			continue // Or handle error appropriately
		}
		Printer.Reset() // Reset printer state for the new app
		Printer.SetAppName(cfg.Name)

		if isBackup && commands {
			// Pre-backup commands
			for _, backupCmd := range cfg.PreBackupCommands {
				if DryRun {
					Printer.Print("Would execute pre-backup command: %s", backupCmd)
					continue
				}
				// Use command package functions
				err := command.SafeExecute("pre-backup command execution", func() error {
					if !command.ExecuteCommandLine(backupCmd) {
						return AppLogger.LogErrorf("command execution failed")
					}
					return nil
				})
				if err != nil {
					AppLogger.Logf("Failed to execute pre-backup command: %v", err)
				}
			}
			// Post-backup commands
			for _, backupCmd := range cfg.PostBackupCommands {
				if DryRun {
					Printer.Print("Would execute post-backup command: %s", backupCmd)
					continue
				}
				err := command.SafeExecute("post-backup command execution", func() error {
					if !command.ExecuteCommandLine(backupCmd) {
						return AppLogger.LogErrorf("command execution failed")
					}
					return nil
				})
				if err != nil {
					AppLogger.Logf("Failed to execute post-backup command: %v", err)
				}
			}
		}

		for _, configFile := range cfg.Files {
			// Replace ~ with the user's home directory
			if strings.HasPrefix(configFile, "~/") {
				configFile = Fs.Join(homeDir, configFile[2:])
			} else if !strings.HasPrefix(configFile, "/") && !strings.HasPrefix(configFile, ".") {
				// Assuming relative paths are relative to home directory
				configFile = Fs.Join(homeDir, configFile)
			} else if strings.HasPrefix(configFile, ".") {
				// Handle paths that start with . (like .config) relative to home
				configFile = Fs.Join(homeDir, configFile)
			}

			// For backup operations, create a versioned directory
			var versionedBackupPath string
			var latestVersion string
			if isBackup {
				// Create timestamp-based directory
				versionedBackupPath = Fs.Join(backupFolder, timestamp, cfg.Name, Fs.Base(configFile))

				_, err := Fs.Stat(configFile)
				if os.IsNotExist(err) {
					// Don't log error if source doesn't exist for backup
					continue
				} else if err != nil {
					// Log other stat errors
					AppLogger.Logf("Error checking source file %s: %v", configFile, err)
					continue
				}

				if DryRun {
					Printer.Print("Would create versioned backup directory: %s\n", Fs.Dir(versionedBackupPath))
				} else {
					err = Fs.MkdirAll(Fs.Dir(versionedBackupPath), 0755)
					if err != nil {
						AppLogger.Logf("Failed to create versioned backup directory '%s': %v", Fs.Dir(versionedBackupPath), err)
						continue
					}
				}
			} else {
				// For restore operations, find the latest version
				latestVersion, err = GetLatestVersionPath(backupFolder)
				if err != nil {
					AppLogger.Logf("Failed to find latest version in '%s': %v", backupFolder, err)
					continue // Skip this config file if latest version not found
				}
				versionedBackupPath = Fs.Join(latestVersion, cfg.Name, Fs.Base(configFile))
			}

			if isBackup {
				// Backup operation with recovery
				err := command.SafeExecute("backup operation", func() error {
					// Check if the source file exists
					info, err := Fs.Stat(configFile)
					if os.IsNotExist(err) {
						if DryRun {
							Printer.Print("Would skip backup of %s (doesn't exist)", configFile)
						}
						return nil // Not an error if source doesn't exist
					} else if err != nil {
						Printer.Print("Error accessing %s: %v\n", configFile, err)
						return nil // Treat as non-fatal for this file
					}

					// In dry-run mode, just show what would be backed up
					if DryRun {
						Printer.Print("Would back up %s to %s", configFile, versionedBackupPath)
						return nil
					}

					// Copy the file/directory
					if info.IsDir() {
						err = copyDirectory(configFile, versionedBackupPath)
					} else {
						err = copyFile(configFile, versionedBackupPath)
					}

					if err != nil {
						Printer.Print("Error backing up %s: %v", configFile, err)
						return err // Return error to SafeExecute
					}

					Printer.Print("Backed up %s to %s", configFile, versionedBackupPath)
					return nil
				})

				if err != nil {
					// Log error from SafeExecute
					AppLogger.Logf("Backup operation failed for %s: %v", configFile, err)
				}
			} else {
				// Restore operation with recovery
				err := command.SafeExecute("restore operation", func() error {
					// Check if the backup file/dir exists
					info, err := Fs.Stat(versionedBackupPath)
					if os.IsNotExist(err) {
						// If backup doesn't exist, skip silently for this file
						return nil
					} else if err != nil {
						Printer.Print("Error accessing backup source %s: %v\n", versionedBackupPath, err)
						return nil // Treat as non-fatal for this file
					}

					// In dry-run mode, just show what would be restored
					if DryRun {
						Printer.Print("Would restore %s to %s", versionedBackupPath, configFile)
						return nil
					}

					// Create the destination directory if it doesn't exist
					err = Fs.MkdirAll(Fs.Dir(configFile), 0755)
					if err != nil {
						Printer.Print("Error creating destination directory '%s': %v", Fs.Dir(configFile), err)
						return err // Return error to SafeExecute
					}

					// Copy the file/directory
					if info.IsDir() {
						err = copyDirectory(versionedBackupPath, configFile)
					} else {
						err = copyFile(versionedBackupPath, configFile)
					}

					if err != nil {
						Printer.Print("Error restoring %s: %v", versionedBackupPath, err)
						return err // Return error to SafeExecute
					}

					Printer.Print("Restored %s to %s", versionedBackupPath, configFile)
					return nil
				})

				if err != nil {
					// Log error from SafeExecute
					AppLogger.Logf("Restore operation failed for %s: %v", configFile, err)
				}
			}
		}

		if !isBackup && commands {
			// Pre-restore commands
			for _, restoreCmd := range cfg.PreRestoreCommands {
				if DryRun {
					Printer.Print("Would execute pre-restore command: %s", restoreCmd)
					continue
				}
				err := command.SafeExecute("pre-restore command execution", func() error {
					if !command.ExecuteCommandLine(restoreCmd) {
						return AppLogger.LogErrorf("command execution failed")
					}
					return nil
				})
				if err != nil {
					AppLogger.Logf("Failed to execute pre-restore command: %v", err)
				}
			}
			// Post-restore commands
			for _, restoreCmd := range cfg.PostRestoreCommands {
				if DryRun {
					Printer.Print("Would execute post-restore command: %s", restoreCmd)
					continue
				}
				err := command.SafeExecute("post-restore command execution", func() error {
					if !command.ExecuteCommandLine(restoreCmd) {
						return AppLogger.LogErrorf("command execution failed")
					}
					return nil
				})
				if err != nil {
					AppLogger.Logf("Failed to execute post-restore command: %v", err)
				}
			}
		}
	}

	if !foundCfg {
		AppLogger.Logf("No .cfg files found to process in %s.", configReadDir)
	}

	// Cleanup old versions if needed
	if isBackup && versionsToKeep > 0 {
		err := CleanupOldVersions(backupFolder, versionsToKeep) // Call local CleanupOldVersions
		if err != nil {
			AppLogger.Logf("Failed to cleanup old versions: %v", err)
		}
	}
}
