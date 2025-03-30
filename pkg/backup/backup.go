package backup

import (
	"SettingsSentry/interfaces"
	"SettingsSentry/logger"
	"SettingsSentry/pkg/command"
	"SettingsSentry/pkg/config"
	"SettingsSentry/pkg/printer"
	"archive/zip" // Added import
	"fmt"         // Added import for fmt.Errorf
	"io"
	iofs "io/fs" // Use alias for standard io/fs
	"os"
	"path/filepath" // Added import for filepath
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

// getVersionedBackupPath removed as unused (superseded by logic in ProcessConfiguration and GetLatestVersionPath)

// GetLatestVersionPath returns the path to the latest backup version (directory or zip)
// and a boolean indicating if it's a zip file.
func GetLatestVersionPath(baseBackupPath string) (path string, isZip bool, err error) {
	// Check if the base path exists
	_, err = Fs.Stat(baseBackupPath) // Assign to existing err
	if err != nil {
		err = AppLogger.LogErrorf("backup path does not exist: %w", err)
		return "", false, err // Return zero values and the error
	}

	// Read the directory entries
	entries, err := Fs.ReadDir(baseBackupPath)
	if err != nil {
		err = AppLogger.LogErrorf("failed to read backup directory: %w", err)
		return "", false, err // Return zero values and the error
	}
	// Find the latest version (directory or zip file)
	var latestPath string
	var latestIsZip bool
	var latestTime time.Time
	foundVersion := false

	for _, entry := range entries {
		entryName := entry.Name()
		isDir := entry.IsDir()
		isZipFile := !isDir && strings.HasSuffix(entryName, ".zip")
		timestampStr := ""

		if isDir {
			// Check if directory name matches timestamp format
			timestampStr = entryName
		} else if isZipFile {
			// Check if zip file name matches timestamp format (without .zip)
			timestampStr = strings.TrimSuffix(entryName, ".zip")
		} else {
			// Skip other files/directories
			continue
		}

		// Try to parse the timestamp string
		t, parseErr := time.Parse("20060102-150405", timestampStr)
		if parseErr != nil {
			// Skip entries that don't match our timestamp format
			continue
		}

		// Check if this version is the latest found so far
		if !foundVersion || t.After(latestTime) {
			latestPath = Fs.Join(baseBackupPath, entryName)
			latestIsZip = isZipFile
			latestTime = t
			foundVersion = true
		}
	}

	if !foundVersion {
		err = AppLogger.LogErrorf("no version backups found in %s", baseBackupPath)
		return "", false, err
	}

	return latestPath, latestIsZip, nil
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

	// Collect all version entries (dirs or zips) with their timestamps
	type versionInfo struct {
		path      string
		timestamp time.Time
		isZip     bool // Add field to track if it's a zip file
	}
	var versions []versionInfo

	for _, entry := range entries {
		entryName := entry.Name()
		isDir := entry.IsDir()
		isZipFile := !isDir && strings.HasSuffix(entryName, ".zip")
		timestampStr := ""

		if isDir {
			timestampStr = entryName
		} else if isZipFile {
			timestampStr = strings.TrimSuffix(entryName, ".zip")
		} else {
			continue // Skip other files/dirs
		}

		// Try to parse the timestamp string
		t, parseErr := time.Parse("20060102-150405", timestampStr)
		if parseErr != nil {
			// Skip entries that don't match our timestamp format
			continue
		}

		versions = append(versions, versionInfo{
			path:      Fs.Join(baseBackupPath, entryName),
			timestamp: t,
			isZip:     isZipFile, // Store whether it's a zip
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
				err := Fs.RemoveAll(versions[i].path) // Assuming RemoveAll works for files too
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
	defer func() {
		if err := srcFile.Close(); err != nil {
			AppLogger.Logf("Error closing source file %s: %v", src, err)
		}
	}()

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
	defer func() {
		if err := dstFile.Close(); err != nil {
			AppLogger.Logf("Error closing destination file %s: %v", dst, err)
		}
	}()

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
func ProcessConfiguration(configFolder, backupFolder, appName string, isBackup bool, commands bool, versionsToKeep int, zipBackup bool, password string) {
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

	var stagingDir string // For zip backups

	// Create timestamp here to avoid different paths during the backup
	timestamp := time.Now().Format("20060102-150405")

	// Create staging directory for zip backup if needed
	if isBackup && zipBackup {
		var tempErr error
		// Use os.MkdirTemp directly as it's a standard OS operation
		stagingDir, tempErr = os.MkdirTemp("", "settingssentry-zip-")
		if tempErr != nil {
			_ = AppLogger.LogErrorf("Failed to create temporary staging directory: %v", tempErr) // Ignore error return
			return                                                                               // Cannot proceed with zip backup without staging dir
		}
		AppLogger.Logf("Using staging directory for zip backup: %s", stagingDir)
		// Ensure staging directory is cleaned up
		defer func() {
			if stagingDir != "" {
				AppLogger.Logf("Cleaning up staging directory: %s", stagingDir)
				if err := Fs.RemoveAll(stagingDir); err != nil {
					// Log the error but don't stop execution as it's cleanup
					AppLogger.Logf("Error removing staging directory %s: %v", stagingDir, err)
				}
			}
		}()
	}

	foundCfg := false // Flag to track if any .cfg files were processed
	for _, file := range files {
		// Skip directories and non-cfg files
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".cfg") {
			continue
		}
		foundCfg = true // Found at least one .cfg file

		// If an app name is specified, only process that app's configuration file.
		// We expect the filename to be exactly appName.cfg (case-insensitive).
		if appName != "" && strings.ToLower(file.Name()) != strings.ToLower(appName)+".cfg" {
			continue // Skip this file if appName is given and filename doesn't match
		}
		// If we reach here, either appName was empty, or the filename matched.

		// Parse the configuration file using the determined filesystem (currentFS)
		// file.Name() is relative to the directory read by iofs.ReadDir
		cfg, err := config.ParseConfig(currentFS, file.Name()) // Use config package function
		if err != nil {
			AppLogger.Logf("Error parsing config file '%s': %v", file.Name(), err) // Log error properly
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
			var targetPath string // Use a general target path variable

			var latestIsZip bool // Declare here for broader scope

			if isBackup {
				if zipBackup {
					// Target path is inside the staging directory for zip backups
					targetPath = Fs.Join(stagingDir, cfg.Name, Fs.Base(configFile))
				} else {
					// Target path is the versioned directory for normal backups
					targetPath = Fs.Join(backupFolder, timestamp, cfg.Name, Fs.Base(configFile))
				}
				versionedBackupPath = targetPath // Keep versionedBackupPath for consistency in copy logic below? Or rename? Let's use targetPath directly.

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
					// For zip, the final dir is inside the zip; for normal, it's the versioned path.
					// Log the intended final structure path for clarity, even if staging is used.
					finalDirForLog := Fs.Dir(Fs.Join(backupFolder, timestamp, cfg.Name, Fs.Base(configFile)))
					if zipBackup {
						finalDirForLog = Fs.Dir(Fs.Join(stagingDir, cfg.Name, Fs.Base(configFile))) // Log staging dir creation
						Printer.Print("Would create staging directory: %s\n", finalDirForLog)
					} else {
						Printer.Print("Would create versioned backup directory: %s\n", finalDirForLog)
					}
				} else {
					// Create the target directory (either staging or final versioned)
					err = Fs.MkdirAll(Fs.Dir(targetPath), 0755)
					if err != nil {
						AppLogger.Logf("Failed to create target directory '%s': %v", Fs.Dir(targetPath), err)
						continue
					}
				}
			} else {
				// For restore operations, find the latest version (dir or zip)
				// latestIsZip declared earlier
				latestVersion, latestIsZip, err = GetLatestVersionPath(backupFolder)
				if err != nil {
					AppLogger.Logf("Failed to find latest version in '%s': %v", backupFolder, err)
					continue // Skip this config file if latest version not found
				}
				// Determine the source path for restore based on whether it's a zip or dir
				if latestIsZip {
					// Path *inside* the zip archive
					versionedBackupPath = filepath.ToSlash(Fs.Join(cfg.Name, Fs.Base(configFile))) // Use filepath.ToSlash for zip internal paths
					// latestVersion holds the path to the zip file itself
				} else {
					// Path is a direct filesystem path within the backup directory
					versionedBackupPath = Fs.Join(latestVersion, cfg.Name, Fs.Base(configFile))
					// latestVersion holds the path to the backup directory
				}
				_ = latestIsZip // Keep this for now, will be used in the next step
			}
			// --- Encryption Logic Start ---
			if isBackup && password != "" {
				err := command.SafeExecute("encryption operation", func() error {
					// Read source file content
					plaintext, readErr := Fs.ReadFile(configFile)
					if os.IsNotExist(readErr) {
						if DryRun {
							Printer.Print("Would skip encryption of %s (doesn't exist)", configFile)
						}
						return nil // Not an error if source doesn't exist
					} else if readErr != nil {
						Printer.Print("Error reading source file %s for encryption: %v\n", configFile, readErr)
						return readErr // Return the actual read error
					}

					// Encrypt content
					encryptedData, encErr := encrypt(plaintext, password)
					if encErr != nil {
						Printer.Print("Error encrypting %s: %v", configFile, encErr)
						return encErr // Return error to SafeExecute
					}

					// Determine encrypted target path
					encryptedTargetPath := targetPath + ".encrypted"

					if DryRun {
						Printer.Print("Would encrypt %s to %s", configFile, encryptedTargetPath) // Log the actual .encrypted path
						return nil
					}

					// Ensure target directory exists (either staging or final versioned)
					if mkErr := Fs.MkdirAll(Fs.Dir(encryptedTargetPath), 0755); mkErr != nil {
						AppLogger.Logf("Failed to create target directory '%s' for encrypted file: %v", Fs.Dir(encryptedTargetPath), mkErr)
						return mkErr // Return error to SafeExecute
					}

					// Write encrypted data
					if writeErr := Fs.WriteFile(encryptedTargetPath, encryptedData, 0644); writeErr != nil {
						Printer.Print("Error writing encrypted file %s: %v", encryptedTargetPath, writeErr)
						return writeErr // Return error to SafeExecute
					}

					Printer.Print("Encrypted %s to %s", configFile, encryptedTargetPath)
					return nil
				})

				if err != nil {
					// Log error from SafeExecute
					AppLogger.Logf("Encryption operation failed for %s: %v", configFile, err)
				}
				// Continue to next file, skipping standard backup logic below
				continue
			}
			// --- Encryption Logic End ---

			// --- Decryption Logic Start ---
			if !isBackup {
				encryptedSourcePath := versionedBackupPath + ".encrypted"
				encryptedSourcePathInZip := filepath.ToSlash(Fs.Join(cfg.Name, Fs.Base(configFile))) + ".encrypted" // Path inside zip
				encryptedFileExists := false
				var readErr error

				// Check if the encrypted file exists (either directly or in zip)
				if latestIsZip {
					// Need to inspect zip contents without extracting everything yet
					// TODO: Implement zip inspection helper or adjust logic
					// For now, assume we need to try extracting and check error
					// Check zip archive directly for the encrypted file entry
					var zipCheckErr error
					encryptedFileExists, zipCheckErr = zipContainsEncrypted(latestVersion, encryptedSourcePathInZip)
					if zipCheckErr != nil {
						AppLogger.Logf("Error checking zip archive %s for encrypted file %s: %v", latestVersion, encryptedSourcePathInZip, zipCheckErr)
						// If we can't check the zip, we can't safely proceed with decryption attempt for this file
						continue // Skip this file
					}
				} else {
					// Check if encrypted file exists directly on filesystem
					_, readErr = Fs.Stat(encryptedSourcePath)
					if readErr == nil {
						encryptedFileExists = true
					}
				}

				if encryptedFileExists {
					if password == "" {
						_ = AppLogger.LogErrorf("Encrypted backup file found for '%s' but no password provided. Use -password flag.", configFile)
						continue // Skip this file
					}

					// Proceed with decryption attempt
					err := command.SafeExecute("decryption operation", func() error {
						var encryptedData []byte
						var readErr error

						// Read encrypted data (from file system or zip)
						if latestIsZip {
							// Extract the specific encrypted file
							// Create a temporary file to extract to?
							tempFile, err := os.CreateTemp("", "settingssentry-decrypt-")
							if err != nil {
								return fmt.Errorf("failed to create temp file for zip extraction: %w", err)
							}
							tempFilePath := tempFile.Name()
							tempFile.Close()              // Close immediately, extractFromZip will reopen/write
							defer os.Remove(tempFilePath) // Clean up temp file

							// Use encryptedSourcePathInZip (relative path inside zip)
							err = extractFromZip(latestVersion, encryptedSourcePathInZip, tempFilePath)
							if err != nil {
								// If extraction failed, maybe the encrypted file wasn't actually there
								Printer.Print("Info: Encrypted file %s not found in zip archive %s.", encryptedSourcePathInZip, latestVersion)
								return nil // Treat as non-fatal, effectively skipping
							}
							encryptedData, readErr = os.ReadFile(tempFilePath)
						} else {
							encryptedData, readErr = Fs.ReadFile(encryptedSourcePath)
						}

						if readErr != nil {
							// This might happen if Stat passed but ReadFile failed, or if zip extract failed
							Printer.Print("Error reading encrypted file %s: %v", encryptedSourcePath, readErr)
							return nil // Treat as non-fatal for this file
						}

						// Decrypt data
						plaintext, decErr := decrypt(encryptedData, password)
						if decErr != nil {
							// Log specific error but return a generic one to SafeExecute if needed
							Printer.Print("Error decrypting %s: %v (Wrong password or corrupt data?)", encryptedSourcePath, decErr)
							return fmt.Errorf("decryption failed") // Return generic error
						}

						// Write decrypted data to final destination
						if DryRun {
							Printer.Print("Would restore (decrypted) %s to %s", encryptedSourcePath, configFile)
							return nil
						}

						// Ensure destination directory exists
						if mkErr := Fs.MkdirAll(Fs.Dir(configFile), 0755); mkErr != nil {
							Printer.Print("Error creating destination directory '%s': %v", Fs.Dir(configFile), mkErr)
							return mkErr
						}

						if writeErr := Fs.WriteFile(configFile, plaintext, 0644); writeErr != nil {
							Printer.Print("Error writing decrypted file %s: %v", configFile, writeErr)
							return writeErr
						}

						Printer.Print("Restored (decrypted) %s to %s", encryptedSourcePath, configFile)
						return nil
					})

					if err != nil {
						// Log error from SafeExecute (already logged specific decrypt error)
						AppLogger.Logf("Decryption/Restore operation failed for %s: %v", configFile, err)
					}
					// Continue to next file, skipping standard restore logic below
					continue
				}
			}
			// --- Decryption Logic End ---

			// --- Standard Backup/Restore (if not handled by encryption/decryption) ---
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
						return err // Return the actual stat error
					}

					// In dry-run mode, just show what would be backed up
					if DryRun {
						Printer.Print("Would back up %s to %s", configFile, versionedBackupPath) // TODO: Fix log path for zip
						return nil
					}

					// Copy the file/directory to the target path (staging or final)
					if info.IsDir() {
						err = copyDirectory(configFile, targetPath)
					} else {
						err = copyFile(configFile, targetPath)
					}

					if err != nil {
						Printer.Print("Error backing up %s: %v", configFile, err)
						return err // Return error to SafeExecute
					}

					// Log the final intended destination for user clarity
					finalBackupPath := Fs.Join(backupFolder, timestamp, cfg.Name, Fs.Base(configFile))
					if zipBackup {
						finalBackupPath = Fs.Join(backupFolder, timestamp+".zip", cfg.Name, Fs.Base(configFile)) // Indicate path within zip
					}
					Printer.Print("Backed up %s to %s", configFile, finalBackupPath)
					return nil
				})

				if err != nil {
					// Log error from SafeExecute
					AppLogger.Logf("Backup operation failed for %s: %v", configFile, err)
				}
			} else {
				// Restore operation with recovery
				err := command.SafeExecute("restore operation", func() error {
					// Note: Existence check is implicitly handled by copy/extract logic below or should be added there if needed.
					// The original check here was complex due to zip vs dir path differences.

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

					// Extract from zip or copy file/directory
					if latestIsZip {
						// latestVersion holds the path to the zip file
						// versionedBackupPath holds the path *inside* the zip
						err = extractFromZip(latestVersion, versionedBackupPath, configFile)
						// Need to handle directory extraction if source was a dir
						// TODO: Enhance extractFromZip to handle directories if needed, or adjust logic here.
						// Current extractFromZip likely only handles files.
					} else {
						// latestVersion holds the path to the backup directory
						// versionedBackupPath holds the full path to the source file/dir within the backup dir
						// Check info from the *source* within the backup dir
						backupSourceInfo, statErr := Fs.Stat(versionedBackupPath)
						if statErr != nil {
							// This check might be redundant if the earlier check (line 539) covers it
							Printer.Print("Error accessing backup source %s: %v\n", versionedBackupPath, statErr)
							return nil // Treat as non-fatal for this file
						}
						if backupSourceInfo.IsDir() {
							err = copyDirectory(versionedBackupPath, configFile)
						} else {
							err = copyFile(versionedBackupPath, configFile)
						}
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
		} // End loop through cfg.Files

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
	} // End loop through config files

	// Create zip archive from staging directory if zipBackup is enabled
	if isBackup && zipBackup {
		if stagingDir == "" {
			// Should not happen if staging dir creation succeeded earlier
			_ = AppLogger.LogErrorf("Staging directory path is empty, cannot create zip archive.") // Ignore error return
		} else {
			targetZipPath := Fs.Join(backupFolder, timestamp+".zip")
			AppLogger.Logf("Creating zip archive: %s", targetZipPath)
			if err := createZipArchive(stagingDir, targetZipPath); err != nil {
				_ = AppLogger.LogErrorf("Failed to create zip archive: %v", err) // Ignore error return
				// Note: Staging dir will still be cleaned up by the deferred RemoveAll
			} else {
				AppLogger.Logf("Successfully created zip archive: %s", targetZipPath)
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
} // Closing brace for ProcessConfiguration

// createZipArchive creates a zip archive from the contents of a source directory.
func createZipArchive(sourceDir, targetZipPath string) error {
	zipFile, err := os.Create(targetZipPath) // Use os.Create directly for zip file creation
	if err != nil {
		return fmt.Errorf("failed to create zip file '%s': %w", targetZipPath, err)
	}
	defer func() {
		if err := zipFile.Close(); err != nil {
			// Log error, but don't return it from createZipArchive as the primary operation might have succeeded
			AppLogger.Logf("Error closing zip file %s: %v", targetZipPath, err)
		}
	}()

	zipWriter := zip.NewWriter(zipFile)
	defer func() {
		if err := zipWriter.Close(); err != nil {
			// Log error, but don't return it from createZipArchive
			AppLogger.Logf("Error closing zip writer for %s: %v", targetZipPath, err)
		}
	}()

	// Walk the staging directory
	err = filepath.Walk(sourceDir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create a proper relative path for the zip header
		relPath, err := filepath.Rel(sourceDir, filePath)
		if err != nil {
			return fmt.Errorf("failed to get relative path for '%s': %w", filePath, err)
		}

		// Skip the root directory itself
		if relPath == "." {
			return nil
		}

		// Create zip header
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return fmt.Errorf("failed to create zip header for '%s': %w", filePath, err)
		}

		// Set the name in the header to the relative path
		header.Name = filepath.ToSlash(relPath) // Use forward slashes in zip archives

		// Set compression method (optional, Deflate is common)
		header.Method = zip.Deflate

		// Create entry in the zip file
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return fmt.Errorf("failed to create zip entry for '%s': %w", relPath, err)
		}

		// If it's a directory, we just created the header, nothing more to do
		if info.IsDir() {
			return nil
		}

		// If it's a file, open it and copy its content into the zip writer
		fileToZip, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("failed to open file '%s' for zipping: %w", filePath, err)
		}
		// Defer close inside the loop iteration
		defer func() {
			if err := fileToZip.Close(); err != nil {
				// Log error, but let Walk continue if possible
				AppLogger.Logf("Error closing file %s during zipping: %v", filePath, err)
			}
		}()

		_, err = io.Copy(writer, fileToZip)
		if err != nil {
			return fmt.Errorf("failed to copy file content for '%s' to zip: %w", filePath, err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("error walking staging directory '%s': %w", sourceDir, err)
	}

	return nil
}

// zipContainsEncrypted checks if a zip archive contains a specific entry.
func zipContainsEncrypted(zipPath, internalEncryptedPath string) (bool, error) {
	// Normalize internal path
	internalEncryptedPath = filepath.ToSlash(internalEncryptedPath)

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		// If zip doesn't exist or is invalid, the encrypted file isn't there
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to open zip archive '%s' for inspection: %w", zipPath, err)
	}
	defer func() {
		if err := r.Close(); err != nil {
			AppLogger.Logf("Error closing zip reader for inspection %s: %v", zipPath, err)
		}
	}()

	for _, f := range r.File {
		if f.Name == internalEncryptedPath {
			return true, nil // Found it
		}
	}

	return false, nil // Not found
}

// extractFromZip extracts a specific file or directory from a zip archive to a destination path.
func extractFromZip(zipPath, entryPath, destinationPath string) error {
	// Open the zip archive for reading.
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip archive '%s': %w", zipPath, err)
	}
	defer func() {
		if err := r.Close(); err != nil {
			AppLogger.Logf("Error closing zip reader for %s: %v", zipPath, err)
		}
	}()

	// Normalize entryPath to use forward slashes, as used in zip headers
	entryPath = filepath.ToSlash(entryPath)

	found := false
	// Iterate through the files in the archive.
	for _, f := range r.File {
		// Check if the file's path matches the entryPath or is within the entryPath (for directories)
		if f.Name == entryPath || strings.HasPrefix(f.Name, entryPath+"/") {
			found = true
			// Determine the full path for extraction
			extractPath := ""
			if f.Name == entryPath {
				// If it's an exact match (file or directory itself), use the destinationPath directly
				extractPath = destinationPath
			} else {
				// If it's inside a directory, calculate the relative path and join with destination
				relPath := strings.TrimPrefix(f.Name, entryPath+"/")
				extractPath = filepath.Join(destinationPath, relPath)
			}

			// Check if it's a directory or a file
			if f.FileInfo().IsDir() {
				// Create the directory.
				if err := os.MkdirAll(extractPath, f.Mode()); err != nil {
					return fmt.Errorf("failed to create directory '%s': %w", extractPath, err)
				}
				continue // Go to the next file in zip
			}

			// It's a file, ensure the directory exists first.
			if err := os.MkdirAll(filepath.Dir(extractPath), os.ModePerm); err != nil {
				return fmt.Errorf("failed to create directory for file '%s': %w", extractPath, err)
			}

			// Open the file inside the zip archive.
			rc, err := f.Open()
			if err != nil {
				return fmt.Errorf("failed to open file '%s' in zip: %w", f.Name, err)
			}

			// Create the destination file.
			dstFile, err := os.OpenFile(extractPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				_ = rc.Close() // Close the source file before returning error (ignore close error here)
				return fmt.Errorf("failed to create destination file '%s': %w", extractPath, err)
			}

			// Copy the contents.
			_, err = io.Copy(dstFile, rc)

			// Close files, log errors if they occur
			closeErrRc := rc.Close()
			closeErrDst := dstFile.Close()

			if closeErrRc != nil {
				AppLogger.Logf("Error closing zip entry reader for %s: %v", f.Name, closeErrRc)
			}
			if closeErrDst != nil {
				AppLogger.Logf("Error closing destination file %s: %v", extractPath, closeErrDst)
			}

			if err != nil {
				// Return the io.Copy error if it occurred
				return fmt.Errorf("failed to copy content to '%s': %w", extractPath, err)
			}
			// If io.Copy succeeded, return potential close errors (prefer dst error)
			if closeErrDst != nil {
				return fmt.Errorf("error closing destination file '%s': %w", extractPath, closeErrDst)
			}
			if closeErrRc != nil {
				return fmt.Errorf("error closing zip entry reader '%s': %w", f.Name, closeErrRc)
			}
		}
	}

	if !found {
		// Return nil error if the specific entry wasn't found, consistent with restore logic skipping non-existent files.
		// Alternatively, could return an error: fmt.Errorf("entry '%s' not found in zip archive '%s'", entryPath, zipPath)
		return nil
	}

	return nil
}
