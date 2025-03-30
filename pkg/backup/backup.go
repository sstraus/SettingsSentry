package backup

import (
	"SettingsSentry/interfaces"
	"SettingsSentry/logger"
	"SettingsSentry/pkg/command"
	"SettingsSentry/pkg/config"
	"SettingsSentry/pkg/printer"
	"archive/zip"
	"fmt"
	"io"
	iofs "io/fs" // Use alias for standard io/fs
	"os"
	"path/filepath"
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

// GetLatestVersionPath returns the path to the latest backup version (directory or zip)
// and a boolean indicating if it's a zip file.
func GetLatestVersionPath(baseBackupPath string) (path string, isZip bool, err error) {
	// Check if the base path exists
	_, err = Fs.Stat(baseBackupPath)
	if err != nil {
		err = AppLogger.LogErrorf("backup path does not exist: %w", err)
		return "", false, err
	}

	// Read the directory entries
	entries, err := Fs.ReadDir(baseBackupPath)
	if err != nil {
		err = AppLogger.LogErrorf("failed to read backup directory: %w", err)
		return "", false, err
	}
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
			timestampStr = entryName
		} else if isZipFile {
			timestampStr = strings.TrimSuffix(entryName, ".zip")
		} else {
			continue
		}

		t, parseErr := time.Parse("20060102-150405", timestampStr)
		if parseErr != nil {
			continue
		}

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
		return nil
	}

	_, err := Fs.Stat(baseBackupPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return AppLogger.LogErrorf("failed to stat backup path for cleanup: %w", err)
	}

	entries, err := Fs.ReadDir(baseBackupPath)
	if err != nil {
		return AppLogger.LogErrorf("failed to read backup directory for cleanup: %w", err)
	}

	type versionInfo struct {
		path      string
		timestamp time.Time
		isZip     bool
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
			continue
		}

		t, parseErr := time.Parse("20060102-150405", timestampStr)
		if parseErr != nil {
			continue
		}

		versions = append(versions, versionInfo{
			path:      Fs.Join(baseBackupPath, entryName),
			timestamp: t,
			isZip:     isZipFile,
		})
	}

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].timestamp.After(versions[j].timestamp)
	})

	if len(versions) > maxVersions {
		for i := maxVersions; i < len(versions); i++ {
			_, statErr := Fs.Stat(versions[i].path)
			if statErr != nil {
				if os.IsNotExist(statErr) {
					AppLogger.Logf("Skipping version that no longer exists: %s", versions[i].path)
					continue
				}
				AppLogger.Logf("Error stating old version %s: %v", versions[i].path, statErr)
				continue
			}
			if DryRun {
				AppLogger.Logf("Would remove old version: %s", versions[i].path)
			} else {
				AppLogger.Logf("Removing old version: %s", versions[i].path)
				err := Fs.RemoveAll(versions[i].path)
				if err != nil {
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
		AppLogger.Logf("copyFile: Fs.Open failed for '%s': %v", src, err)
		return AppLogger.LogErrorf("failed to open source file '%s': %w", src, err)
	}
	defer func() {
		if err := srcFile.Close(); err != nil {
			AppLogger.Logf("Error closing source file %s: %v", src, err)
		}
	}()

	err = Fs.MkdirAll(Fs.Dir(dst), 0755)
	if err != nil {
		return AppLogger.LogErrorf("failed to create destination directory '%s': %w", Fs.Dir(dst), err)
	}

	dstFile, err := Fs.Create(dst)
	if err != nil {
		return AppLogger.LogErrorf("failed to create destination file '%s': %w", dst, err)
	}
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
	srcInfo, err := Fs.Stat(src)
	if err != nil {
		return AppLogger.LogErrorf("failed to get source directory info '%s': %w", src, err)
	}

	err = Fs.MkdirAll(dst, srcInfo.Mode())
	if err != nil {
		return AppLogger.LogErrorf("failed to create destination directory '%s': %w", dst, err)
	}

	entries, err := Fs.ReadDir(src)
	if err != nil {
		return AppLogger.LogErrorf("failed to read source directory '%s': %w", src, err)
	}

	for _, entry := range entries {
		srcPath := Fs.Join(src, entry.Name())
		dstPath := Fs.Join(dst, entry.Name())

		if entry.IsDir() {
			err = copyDirectory(srcPath, dstPath)
			if err != nil {
				return err
			}
		} else {
			err = copyFile(srcPath, dstPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// ProcessConfiguration processes configuration files for backup or restore.
// Accepts a slice of app names to process specific applications.
func ProcessConfiguration(configFolder, backupFolder string, appNames []string, isBackup bool, commands bool, versionsToKeep int, zipBackup bool, password string) {
	configFolder = config.ExpandEnvVars(configFolder)
	backupFolder = config.ExpandEnvVars(backupFolder)

	if !strings.Contains(configFolder, string(os.PathSeparator)) {
		exePath, err := os.Executable()
		if err != nil {
			AppLogger.Logf("Error getting executable path: %v", err)
			return
		}
		configFolder = Fs.Join(Fs.Dir(exePath), configFolder)
	}

	var currentFS iofs.FS
	var files []iofs.DirEntry
	var configReadDir string

	_, err := Fs.Stat(configFolder)
	if err != nil {
		AppLogger.Logf("Config folder '%s' not found or inaccessible: %v. Attempting to use embedded configs.", configFolder, err)
		AppLogger.Logf("Error: Embedded fallback logic needs rootEmbedFS passed.")
		return

	} else {
		AppLogger.Logf("Using config folder: %s", configFolder)

		currentFS = os.DirFS(configFolder)
		configReadDir = configFolder
		files, err = iofs.ReadDir(currentFS, ".")
		if err != nil {
			AppLogger.Logf("Error reading config folder '%s': %v", configFolder, err)
			return
		}
		if len(files) == 0 {
			AppLogger.Logf("No configuration files found in '%s'.", configFolder)
			return
		}
	}

	if isBackup {
		if DryRun {
			AppLogger.Logf("Would create backup folder: %s", backupFolder)
		} else {
			err = Fs.MkdirAll(backupFolder, 0755)
			if err != nil {
				AppLogger.Logf("Failed to create backup folder: %v", err)
				return
			}
		}
	} else {
		_, err := Fs.Stat(backupFolder)
		if err != nil {
			AppLogger.Logf("Backup folder does not exist or is not accessible: %v", err)
			return
		}
	}

	homeDir, err := config.GetHomeDirectory()
	if err != nil {
		AppLogger.Logf("Error getting home directory: %v", err)
		return
	}

	var stagingDir string

	timestamp := time.Now().Format("20060102-150405")

	if isBackup && zipBackup {
		var tempErr error
		stagingDir, tempErr = os.MkdirTemp("", "settingssentry-zip-")
		if tempErr != nil {
			_ = AppLogger.LogErrorf("Failed to create temporary staging directory: %v", tempErr)
			return
		}
		AppLogger.Logf("Using staging directory for zip backup: %s", stagingDir)
		defer func() {
			if stagingDir != "" {
				AppLogger.Logf("Cleaning up staging directory: %s", stagingDir)
				if err := Fs.RemoveAll(stagingDir); err != nil {
					AppLogger.Logf("Error removing staging directory %s: %v", stagingDir, err)
				}
			}
		}()
	}

	foundCfg := false
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".cfg") {
			continue
		}

		// Filter based on appNames slice if it's provided
		if len(appNames) > 0 {
			matchFound := false
			currentFileNameLower := strings.ToLower(file.Name())
			for _, requestedAppName := range appNames {
				// Check if current file matches AppName.cfg (case-insensitive)
				if currentFileNameLower == strings.ToLower(requestedAppName)+".cfg" {
					matchFound = true
					break
				}
			}
			if !matchFound {
				continue // Skip this file if no matching app name was found
			}
		}
		// If we reach here, either appNames was empty, or a match was found.
		foundCfg = true

		cfg, err := config.ParseConfig(currentFS, file.Name())
		if err != nil {
			AppLogger.Logf("Error parsing config file '%s': %v", file.Name(), err)
			continue
		}

		if Printer == nil {
			AppLogger.Logf("Error: Printer is not initialized.")
			continue
		}
		Printer.Reset()
		Printer.SetAppName(cfg.Name)

		// --- Pre-load zip content map for restore operations ---
		var zipFileMap map[string]bool
		var zipReader *zip.ReadCloser
		var latestVersionForCfg string
		var latestIsZipForCfg bool

		if !isBackup {
			var findErr error
			latestVersionForCfg, latestIsZipForCfg, findErr = GetLatestVersionPath(backupFolder)
			if findErr != nil {
				AppLogger.Logf("Failed to find latest version in '%s' for app '%s': %v", backupFolder, cfg.Name, findErr)
				continue
			}

			if latestIsZipForCfg {
				var openErr error
				zipReader, openErr = zip.OpenReader(latestVersionForCfg)
				if openErr != nil {
					AppLogger.Logf("Failed to open zip backup '%s' for app '%s': %v", latestVersionForCfg, cfg.Name, openErr)
					continue
				}
				defer func() {
					if zipReader != nil {
						if err := zipReader.Close(); err != nil {
							AppLogger.Logf("Error closing zip reader for %s: %v", latestVersionForCfg, err)
						}
					}
				}()

				zipFileMap = make(map[string]bool)
				for _, f := range zipReader.File {
					zipFileMap[f.Name] = true
				}
			}
		}
		// --- End pre-load ---

		if isBackup && commands {
			for _, backupCmd := range cfg.PreBackupCommands {
				if DryRun {
					Printer.Print("Would execute pre-backup command: %s", backupCmd)
					continue
				}
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
			if strings.HasPrefix(configFile, "~/") {
				configFile = Fs.Join(homeDir, configFile[2:])
			} else if !strings.HasPrefix(configFile, "/") && !strings.HasPrefix(configFile, ".") {
				configFile = Fs.Join(homeDir, configFile)
			} else if strings.HasPrefix(configFile, ".") {
				configFile = Fs.Join(homeDir, configFile)
			}

			var versionedBackupPath string
			var latestVersion string
			var targetPath string

			var latestIsZip bool

			if isBackup {
				latestIsZip = zipBackup
				if zipBackup {
					targetPath = Fs.Join(stagingDir, cfg.Name, Fs.Base(configFile))
				} else {
					targetPath = Fs.Join(backupFolder, timestamp, cfg.Name, Fs.Base(configFile))
				}
				versionedBackupPath = targetPath

				_, err := Fs.Stat(configFile)
				if os.IsNotExist(err) {
					continue
				} else if err != nil {
					AppLogger.Logf("Error checking source file %s: %v", configFile, err)
					continue
				}

				if DryRun {
					finalDirForLog := Fs.Dir(Fs.Join(backupFolder, timestamp, cfg.Name, Fs.Base(configFile)))
					if zipBackup {
						finalDirForLog = Fs.Dir(Fs.Join(stagingDir, cfg.Name, Fs.Base(configFile)))
						Printer.Print("Would create staging directory: %s\n", finalDirForLog)
					} else {
						Printer.Print("Would create versioned backup directory: %s\n", finalDirForLog)
					}
				} else {
					err = Fs.MkdirAll(Fs.Dir(targetPath), 0755)
					if err != nil {
						AppLogger.Logf("Failed to create target directory '%s': %v", Fs.Dir(targetPath), err)
						continue
					}
				}
			} else {
				latestVersion = latestVersionForCfg
				latestIsZip = latestIsZipForCfg

				if latestIsZip {
					versionedBackupPath = filepath.ToSlash(Fs.Join(cfg.Name, Fs.Base(configFile)))
				} else {
					versionedBackupPath = Fs.Join(latestVersion, cfg.Name, Fs.Base(configFile))
				}
			}
			// --- Encryption Logic Start ---
			if isBackup && password != "" {
				err := command.SafeExecute("encryption operation", func() error {
					plaintext, readErr := Fs.ReadFile(configFile)
					if os.IsNotExist(readErr) {
						if DryRun {
							Printer.Print("Would skip encryption of %s (doesn't exist)", configFile)
						}
						return nil
					} else if readErr != nil {
						Printer.Print("Error reading source file %s for encryption: %v\n", configFile, readErr)
						return readErr
					}

					encryptedData, encErr := encrypt(plaintext, password)
					if encErr != nil {
						Printer.Print("Error encrypting %s: %v", configFile, encErr)
						return encErr
					}

					encryptedTargetPath := targetPath + ".encrypted"

					if DryRun {
						Printer.Print("Would encrypt %s to %s", configFile, encryptedTargetPath)
						return nil
					}

					if mkErr := Fs.MkdirAll(Fs.Dir(encryptedTargetPath), 0755); mkErr != nil {
						AppLogger.Logf("Failed to create target directory '%s' for encrypted file: %v", Fs.Dir(encryptedTargetPath), mkErr)
						return mkErr
					}

					if writeErr := Fs.WriteFile(encryptedTargetPath, encryptedData, 0644); writeErr != nil {
						Printer.Print("Error writing encrypted file %s: %v", encryptedTargetPath, writeErr)
						return writeErr
					}

					Printer.Print("Encrypted %s to %s", configFile, encryptedTargetPath)
					return nil
				})

				if err != nil {
					AppLogger.Logf("Encryption operation failed for %s: %v", configFile, err)
				}
				continue
			}
			// --- Encryption Logic End ---

			// --- Decryption Logic Start ---
			if !isBackup {
				encryptedSourcePath := versionedBackupPath + ".encrypted"
				encryptedSourcePathInZip := filepath.ToSlash(Fs.Join(cfg.Name, Fs.Base(configFile))) + ".encrypted"
				encryptedFileExists := false
				var readErr error

				if latestIsZip {
					if zipFileMap != nil {
						_, encryptedFileExists = zipFileMap[encryptedSourcePathInZip]
					} else {
						AppLogger.Logf("Error: zipFileMap is nil despite latestIsZip being true for app %s", cfg.Name)
						encryptedFileExists = false
					}
				} else {
					_, readErr = Fs.Stat(encryptedSourcePath)
					if readErr == nil {
						encryptedFileExists = true
					}
				}

				if encryptedFileExists {
					if password == "" {
						_ = AppLogger.LogErrorf("Encrypted backup file found for '%s' but no password provided. Use -password flag.", configFile)
						continue
					}

					err := command.SafeExecute("decryption operation", func() error {
						var encryptedData []byte
						var readErr error

						if latestIsZip {
							tempFile, err := os.CreateTemp("", "settingssentry-decrypt-")
							if err != nil {
								return fmt.Errorf("failed to create temp file for zip extraction: %w", err)
							}
							tempFilePath := tempFile.Name()
							// Check error on close immediately
							if closeErr := tempFile.Close(); closeErr != nil {
								// Attempt to remove before returning, but prioritize close error
								_ = os.Remove(tempFilePath) // Ignore remove error if close failed
								return fmt.Errorf("failed to close temp file handle: %w", closeErr)
							}
							defer func() {
								if removeErr := os.Remove(tempFilePath); removeErr != nil {
									AppLogger.Logf("Warning: Failed to remove temp decryption file %s: %v", tempFilePath, removeErr)
								}
							}() // Check remove error in defer

							err = extractFromZip(latestVersion, encryptedSourcePathInZip, tempFilePath)
							if err != nil {
								Printer.Print("Info: Encrypted file %s not found in zip archive %s.", encryptedSourcePathInZip, latestVersion)
								return nil
							}
							encryptedData, readErr = os.ReadFile(tempFilePath)
						} else {
							encryptedData, readErr = Fs.ReadFile(encryptedSourcePath)
						}

						if readErr != nil {
							Printer.Print("Error reading encrypted file %s: %v", encryptedSourcePath, readErr)
							return nil
						}

						plaintext, decErr := decrypt(encryptedData, password)
						if decErr != nil {
							Printer.Print("Error decrypting %s: %v (Wrong password or corrupt data?)", encryptedSourcePath, decErr)
							return fmt.Errorf("decryption failed")
						}

						if DryRun {
							Printer.Print("Would restore (decrypted) %s to %s", encryptedSourcePath, configFile)
							return nil
						}

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
						AppLogger.Logf("Decryption/Restore operation failed for %s: %v", configFile, err)
					}
					continue
				}
			}
			// --- Decryption Logic End ---

			// --- Standard Backup/Restore (if not handled by encryption/decryption) ---
			if isBackup {
				err := command.SafeExecute("backup operation", func() error {
					info, err := Fs.Stat(configFile)
					if os.IsNotExist(err) {
						if DryRun {
							Printer.Print("Would skip backup of %s (doesn't exist)", configFile)
						}
						return nil
					} else if err != nil {
						Printer.Print("Error accessing %s: %v\n", configFile, err)
						return err
					}

					if DryRun {
						Printer.Print("Would back up %s to %s", configFile, versionedBackupPath) // TODO: Fix log path for zip
						return nil
					}

					if info.IsDir() {
						err = copyDirectory(configFile, targetPath)
					} else {
						err = copyFile(configFile, targetPath)
					}

					if err != nil {
						Printer.Print("Error backing up %s: %v", configFile, err)
						return err
					}

					finalBackupPath := Fs.Join(backupFolder, timestamp, cfg.Name, Fs.Base(configFile))
					if zipBackup {
						finalBackupPath = Fs.Join(backupFolder, timestamp+".zip", cfg.Name, Fs.Base(configFile)) // Indicate path within zip
					}
					Printer.Print("Backed up %s to %s", configFile, finalBackupPath)
					return nil
				})

				if err != nil {
					AppLogger.Logf("Backup operation failed for %s: %v", configFile, err)
				}
			} else {
				err := command.SafeExecute("restore operation", func() error {

					if DryRun {
						Printer.Print("Would restore %s to %s", versionedBackupPath, configFile)
						return nil
					}

					err = Fs.MkdirAll(Fs.Dir(configFile), 0755)
					if err != nil {
						Printer.Print("Error creating destination directory '%s': %v", Fs.Dir(configFile), err)
						return err
					}

					if latestIsZip {
						err = extractFromZip(latestVersion, versionedBackupPath, configFile)
						// Need to handle directory extraction if source was a dir
						// TODO: Enhance extractFromZip to handle directories if needed, or adjust logic here.
						// Current extractFromZip likely only handles files.
					} else {
						backupSourceInfo, statErr := Fs.Stat(versionedBackupPath)
						if statErr != nil {
							//Printer.Print("Error accessing backup source %s: %v\n", versionedBackupPath, statErr)
							return nil
						}
						if backupSourceInfo.IsDir() {
							err = copyDirectory(versionedBackupPath, configFile)
						} else {
							err = copyFile(versionedBackupPath, configFile)
						}
					}

					if err != nil {
						Printer.Print("Error restoring %s: %v", versionedBackupPath, err)
						return err
					}

					Printer.Print("Restored %s to %s", versionedBackupPath, configFile)
					return nil
				})

				if err != nil {
					AppLogger.Logf("Restore operation failed for %s: %v", configFile, err)
				}
			}
		} // End loop through cfg.Files

		if !isBackup && commands {
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

	if isBackup && zipBackup {
		if stagingDir == "" {
			_ = AppLogger.LogErrorf("Staging directory path is empty, cannot create zip archive.")
		} else {
			targetZipPath := Fs.Join(backupFolder, timestamp+".zip")
			AppLogger.Logf("Creating zip archive: %s", targetZipPath)
			if err := createZipArchive(stagingDir, targetZipPath); err != nil {
				_ = AppLogger.LogErrorf("Failed to create zip archive: %v", err)
			} else {
				AppLogger.Logf("Successfully created zip archive: %s", targetZipPath)
			}
		}
	}

	if !foundCfg {
		AppLogger.Logf("No .cfg files found to process in %s.", configReadDir)
	}

	if isBackup && versionsToKeep > 0 {
		err := CleanupOldVersions(backupFolder, versionsToKeep)
		if err != nil {
			AppLogger.Logf("Failed to cleanup old versions: %v", err)
		}
	}
} // Closing brace for ProcessConfiguration

// createZipArchive creates a zip archive from the contents of a source directory.
func createZipArchive(sourceDir, targetZipPath string) error {
	zipFile, err := os.Create(targetZipPath)
	if err != nil {
		return fmt.Errorf("failed to create zip file '%s': %w", targetZipPath, err)
	}
	defer func() {
		if err := zipFile.Close(); err != nil {
			AppLogger.Logf("Error closing zip file %s: %v", targetZipPath, err)
		}
	}()

	zipWriter := zip.NewWriter(zipFile)
	defer func() {
		if err := zipWriter.Close(); err != nil {
			AppLogger.Logf("Error closing zip writer for %s: %v", targetZipPath, err)
		}
	}()

	err = filepath.Walk(sourceDir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(sourceDir, filePath)
		if err != nil {
			return fmt.Errorf("failed to get relative path for '%s': %w", filePath, err)
		}

		if relPath == "." {
			return nil
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return fmt.Errorf("failed to create zip header for '%s': %w", filePath, err)
		}

		header.Name = filepath.ToSlash(relPath) // Use forward slashes in zip archives

		header.Method = zip.Deflate // Set compression method (optional, Deflate is common)

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return fmt.Errorf("failed to create zip entry for '%s': %w", relPath, err)
		}

		if info.IsDir() {
			return nil
		}

		fileToZip, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("failed to open file '%s' for zipping: %w", filePath, err)
		}
		defer func() {
			if err := fileToZip.Close(); err != nil {
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

// extractFromZip extracts a specific file or directory from a zip archive to a destination path.
func extractFromZip(zipPath, entryPath, destinationPath string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip archive '%s': %w", zipPath, err)
	}
	defer func() {
		if err := r.Close(); err != nil {
			AppLogger.Logf("Error closing zip reader for %s: %v", zipPath, err)
		}
	}()

	entryPath = filepath.ToSlash(entryPath) // Normalize entryPath to use forward slashes, as used in zip headers

	found := false
	for _, f := range r.File {
		if f.Name == entryPath || strings.HasPrefix(f.Name, entryPath+"/") {
			found = true
			extractPath := ""
			if f.Name == entryPath {
				extractPath = destinationPath
			} else {
				relPath := strings.TrimPrefix(f.Name, entryPath+"/")
				extractPath = filepath.Join(destinationPath, relPath)
			}

			if f.FileInfo().IsDir() {
				if err := os.MkdirAll(extractPath, f.Mode()); err != nil {
					return fmt.Errorf("failed to create directory '%s': %w", extractPath, err)
				}
				continue
			}

			if err := os.MkdirAll(filepath.Dir(extractPath), os.ModePerm); err != nil {
				return fmt.Errorf("failed to create directory for file '%s': %w", extractPath, err)
			}

			rc, err := f.Open()
			if err != nil {
				return fmt.Errorf("failed to open file '%s' in zip: %w", f.Name, err)
			}

			dstFile, err := os.OpenFile(extractPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				_ = rc.Close()
				return fmt.Errorf("failed to create destination file '%s': %w", extractPath, err)
			}

			_, err = io.Copy(dstFile, rc)

			closeErrRc := rc.Close()
			closeErrDst := dstFile.Close()

			if closeErrRc != nil {
				AppLogger.Logf("Error closing zip entry reader for %s: %v", f.Name, closeErrRc)
			}
			if closeErrDst != nil {
				AppLogger.Logf("Error closing destination file %s: %v", extractPath, closeErrDst)
			}

			if err != nil {
				return fmt.Errorf("failed to copy content to '%s': %w", extractPath, err)
			}
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
		return nil
	}

	return nil
}
