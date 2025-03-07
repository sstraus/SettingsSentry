package main

import (
	cronjob "SettingsSentry/cron"
	"SettingsSentry/interfaces"
	"SettingsSentry/logger"
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strings"
	"time"
)

var (
	Version = "1.1.4"
)

// Global dependencies
var (
	// Global printer variable
	printer *Printer
	// Global file system
	fs interfaces.FileSystem = interfaces.NewOsFileSystem()
	// Global command executor
	cmdExecutor interfaces.CommandExecutor = interfaces.NewOsCommandExecutor()
	// Global dry run flag
	dryRun bool = false
	// Global logger
	appLogger *logger.Logger
)

// Printer implements the interfaces.Printer interface
type Printer struct {
	printAppName string
	firstPrint   bool
}

// Updated NewPrinter to accept an initial app name
func NewPrinter(appName string) *Printer {
	return &Printer{
		printAppName: "\n\033[1m" + appName + "\033[0m -> ", // Set the initial app name
		firstPrint:   true,
	}
}

func (p *Printer) Print(format string, args ...interface{}) {
	if p.firstPrint {
		// Prepend printAppName for the first print
		if appLogger != nil {
			appLogger.Logf("%s "+format, append([]interface{}{p.printAppName}, args...)...)
		} else {
			fmt.Printf("%s "+format+"\n", append([]interface{}{p.printAppName}, args...)...)
		}
		p.firstPrint = false // Reset the state after printing for the first time
	} else {
		// Normal print for subsequent calls
		if appLogger != nil {
			appLogger.Logf(format, args...)
		} else {
			fmt.Printf(format+"\n", args...)
		}
	}
}

func (p *Printer) Reset() {
	p.firstPrint = true
}

type Config struct {
	Name                string
	Files               []string
	PreBackupCommands   []string
	PostBackupCommands  []string
	PreRestoreCommands  []string
	PostRestoreCommands []string
}

// getHomeDirectory returns the user's home directory.
func getHomeDirectory() (string, error) {
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		return "", errors.New("unable to find home directory. HOME environment variable may not be set")
	}
	return homeDir, nil
}

// getXDGConfigHome returns the XDG_CONFIG_HOME directory
func getXDGConfigHome() (string, error) {
	homeDir, err := getHomeDirectory()
	if err != nil {
		return "", err
	}

	// Check if XDG_CONFIG_HOME is set
	xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfigHome == "" {
		// Default to ~/.config if not set
		xdgConfigHome = filepath.Join(homeDir, ".config")
	}

	// Ensure XDG_CONFIG_HOME is within the home directory
	if !strings.HasPrefix(xdgConfigHome, homeDir) {
		return "", fmt.Errorf("$XDG_CONFIG_HOME: %s must be somewhere within your home directory: %s", xdgConfigHome, homeDir)
	}

	return xdgConfigHome, nil
}

// get_iCloud_folder_location returns the path to the iCloud Drive folder.
func get_iCloud_folder_location() (string, error) {
	homeDir, err := getHomeDirectory()
	if err != nil {
		return "", err
	}

	// The iCloud Drive folder is typically located at ~/Library/Mobile Documents/com~apple~CloudDocs/
	iCloudPath := fs.Join(homeDir, "Library", "Mobile Documents", "com~apple~CloudDocs")

	// Check if the directory exists
	_, err = fs.Stat(iCloudPath)
	if err != nil {
		return "", fmt.Errorf("iCloud Drive folder not found: %w", err)
	}

	// Resolve any symlinks
	resolvedPath, err := fs.EvalSymlinks(iCloudPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve iCloud Drive path: %w", err)
	}

	return resolvedPath, nil
}

// expandEnvVars replaces environment variables in the given string
// Format: ${ENV_VAR} or $ENV_VAR
func expandEnvVars(value string) string {
	// Replace ${VAR} format
	result := os.Expand(value, func(key string) string {
		return os.Getenv(key)
	})

	return result
}

// validateConfig checks if the configuration is valid
func validateConfig(config Config) error {
	if config.Name == "" {
		return errors.New("application name is required in configuration")
	}

	if len(config.Files) == 0 {
		return errors.New("at least one configuration file must be specified")
	}

	// Validate that all config files have valid paths
	for _, file := range config.Files {
		if strings.TrimSpace(file) == "" {
			return errors.New("empty configuration file path specified")
		}

		// If the path starts with ~, expand it
		if strings.HasPrefix(file, "~") {
			homeDir, err := getHomeDirectory()
			if err != nil {
				return fmt.Errorf("failed to expand ~ in path %s: %w", file, err)
			}

			expandedPath := strings.Replace(file, "~", homeDir, 1)
			// We don't check if the file exists here because it might not exist yet for restore operations
			if strings.Contains(expandedPath, "*") || strings.Contains(expandedPath, "?") {
				// Check if the pattern is valid (contains directory that exists)
				dir := filepath.Dir(expandedPath)
				if _, err := fs.Stat(dir); err != nil {
					return fmt.Errorf("directory for glob pattern %s does not exist: %w", file, err)
				}
			}
		}
	}

	return nil
}

// parseConfig reads and parses the content of a .cfg file into a Config struct.
func parseConfig(filePath string) (Config, error) {
	var config Config

	// Read the file
	data, err := fs.ReadFile(filePath)
	if err != nil {
		return config, fmt.Errorf("failed to read config file: %w", err)
	}

	// Create a scanner to read the file line by line
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	var section string

	// Parse the file
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		// Check if this is a section header
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(line[1 : len(line)-1])
			continue
		}

		// Handle "name = value" format for application section
		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			key := strings.TrimSpace(strings.ToLower(parts[0]))
			value := strings.TrimSpace(parts[1])

			// Expand environment variables in the value
			value = expandEnvVars(value)

			if section == "application" && key == "name" {
				config.Name = value
				continue
			}
		}

		// Process the line based on the current section
		switch section {
		case "app":
			config.Name = expandEnvVars(line)
		case "application":
			// Already handled above for "name = value" format
			if !strings.Contains(line, "=") {
				config.Name = expandEnvVars(line)
			}
		case "files", "configuration_files":
			config.Files = append(config.Files, expandEnvVars(line))
		case "xdg_configuration_files":
			// Handle XDG configuration files
			path := expandEnvVars(line)
			if strings.HasPrefix(path, "/") {
				return config, fmt.Errorf("unsupported absolute path in xdg_configuration_files: %s", path)
			}

			// Get XDG_CONFIG_HOME
			xdgConfigHome, err := getXDGConfigHome()
			if err != nil {
				return config, fmt.Errorf("error getting XDG_CONFIG_HOME: %w", err)
			}

			// Combine with XDG_CONFIG_HOME
			fullPath := filepath.Join(xdgConfigHome, path)

			// Make path relative to home directory
			homeDir, err := getHomeDirectory()
			if err != nil {
				return config, fmt.Errorf("error getting home directory: %w", err)
			}

			relativePath := strings.Replace(fullPath, homeDir+"/", "", 1)

			config.Files = append(config.Files, relativePath)
		case "backup", "backup_commands", "pre_backup_commands":
			config.PreBackupCommands = append(config.PreBackupCommands, expandEnvVars(line))
		case "post_backup_commands":
			config.PostBackupCommands = append(config.PostBackupCommands, expandEnvVars(line))
		case "restore", "restore_commands", "pre_restore_commands":
			config.PreRestoreCommands = append(config.PreRestoreCommands, expandEnvVars(line))
		case "post_restore_commands":
			config.PostRestoreCommands = append(config.PostRestoreCommands, expandEnvVars(line))
		}
	}

	if err := scanner.Err(); err != nil {
		return config, fmt.Errorf("error scanning config file: %w", err)
	}

	// Validate the configuration
	if err := validateConfig(config); err != nil {
		return config, fmt.Errorf("invalid configuration in %s: %w", filePath, err)
	}

	return config, nil
}

// getVersionedBackupPath returns a versioned backup path
// This function is currently unused but kept for future use
//
//nolint:unused
func getVersionedBackupPath(baseBackupPath string, createNew bool) (string, error) {
	if !createNew {
		// For restore operations, just use the latest version
		return getLatestVersionPath(baseBackupPath)
	}

	// For backup operations, create a new version
	timestamp := time.Now().Format("20060102-150405")
	versionedPath := fs.Join(baseBackupPath, timestamp)

	_, err := fs.Stat(versionedPath)
	if os.IsNotExist(err) {
		return versionedPath, err
	}
	// Create the versioned directory
	if !dryRun {
		err := fs.MkdirAll(versionedPath, 0755)
		if err != nil {
			return "", fmt.Errorf("failed to create versioned backup directory: %w", err)
		}
	}

	return versionedPath, nil
}

// getLatestVersionPath returns the path to the latest version
func getLatestVersionPath(baseBackupPath string) (string, error) {
	// Check if the base path exists
	_, err := fs.Stat(baseBackupPath)
	if err != nil {
		return "", fmt.Errorf("backup path does not exist: %w", err)
	}

	// Read the directory entries
	entries, err := fs.ReadDir(baseBackupPath)
	if err != nil {
		return "", fmt.Errorf("failed to read backup directory: %w", err)
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
		return "", fmt.Errorf("no version backups found in %s", baseBackupPath)
	}

	return fs.Join(baseBackupPath, latestEntry.Name()), nil
}

// cleanupOldVersions removes old versions to keep only the specified number
func cleanupOldVersions(baseBackupPath string, maxVersions int) error {
	if maxVersions <= 0 {
		// Keep all versions
		return nil
	}

	// Check if the base path exists
	_, err := fs.Stat(baseBackupPath)
	if err != nil {
		return nil
	}

	// Read the directory entries
	entries, err := fs.ReadDir(baseBackupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup directory: %w", err)
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
			path:      fs.Join(baseBackupPath, entry.Name()),
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
			_, err := fs.Stat(versions[i].path)
			if err != nil {
				if appLogger != nil {
					appLogger.Logf("Skipping version that no longer exists: %s", versions[i].path)
				} else {
					fmt.Printf("Skipping version that no longer exists: %s\n", versions[i].path)
				}
				continue
			}
			if dryRun {
				if appLogger != nil {
					appLogger.Logf("Would remove old version: %s", versions[i].path)
				} else {
					fmt.Printf("Would remove old version: %s\n", versions[i].path)
				}
			} else {
				if appLogger != nil {
					appLogger.Logf("Removing old version: %s", versions[i].path)
				} else {
					fmt.Printf("Removing old version: %s\n", versions[i].path)
				}
				err := fs.RemoveAll(versions[i].path)
				if err != nil {
					if appLogger != nil {
						appLogger.Logf("Failed to remove old version %s: %v", versions[i].path, err)
					} else {
						fmt.Printf("Failed to remove old version %s: %v\n", versions[i].path, err)
					}
				}
			}
		}
	}

	return nil
}

// copyFile copies a single file from src to dst.
func copyFile(src, dst string) error {
	srcFile, err := fs.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	err = fs.MkdirAll(fs.Dir(dst), 0755)
	if err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	dstFile, err := fs.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}
	return nil
}

// copyDirectory recursively copies a directory from src to dst.
func copyDirectory(src, dst string) error {
	// Get file info for the source directory
	srcInfo, err := fs.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to get source directory info: %w", err)
	}

	// Create the destination directory
	err = fs.MkdirAll(dst, srcInfo.Mode())
	if err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Read the source directory
	entries, err := fs.ReadDir(src)
	if err != nil {
		return fmt.Errorf("failed to read source directory: %w", err)
	}

	// Copy each entry
	for _, entry := range entries {
		srcPath := fs.Join(src, entry.Name())
		dstPath := fs.Join(dst, entry.Name())

		if entry.IsDir() {
			// Recursively copy subdirectories
			err = copyDirectory(srcPath, dstPath)
			if err != nil {
				return err
			}
		} else {
			// Copy files
			err = copyFile(srcPath, dstPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// safeExecute executes a function with panic recovery
func safeExecute(operation string, fn func() error) error {
	defer func() {
		if r := recover(); r != nil {
			printer.Print("Panic recovered in %s: %v\nStack trace: %s", operation, r, string(debug.Stack()))
		}
	}()

	return fn()
}

// processConfiguration processes configuration files for backup or restore.
func processConfiguration(configFolder, backupFolder, appName string, isBackup bool, noCommands bool, versionsToKeep int) {
	// Expand environment variables in paths
	configFolder = expandEnvVars(configFolder)
	backupFolder = expandEnvVars(backupFolder)

	// Check if configFolder does not contain "/"
	if !strings.Contains(configFolder, string(os.PathSeparator)) {
		exePath, err := os.Executable() // Get the path of the executable
		if err != nil {
			if appLogger != nil {
				appLogger.Logf("Error getting executable path: %v", err)
			} else {
				fmt.Printf("Error getting executable path: %v\n", err)
			}
			return
		}
		configFolder = fs.Join(fs.Dir(exePath), configFolder) // Append to the executable directory
	}

	// Validate the config folder exists
	_, err := fs.Stat(configFolder)
	if err != nil {
		if appLogger != nil {
			appLogger.Logf("Config folder does not exist or is not accessible: %v", err)
		} else {
			fmt.Printf("Config folder does not exist or is not accessible: %v\n", err)
		}
		return
	}

	// Validate the backup folder exists or create it
	if isBackup {
		if dryRun {
			if appLogger != nil {
				appLogger.Logf("Would create backup folder: %s", backupFolder)
			} else {
				fmt.Printf("Would create backup folder: %s\n", backupFolder)
			}
		} else {
			err = fs.MkdirAll(backupFolder, 0755)
			if err != nil {
				if appLogger != nil {
					appLogger.Logf("Failed to create backup folder: %v", err)
				} else {
					fmt.Printf("Failed to create backup folder: %v\n", err)
				}
				return
			}
		}
	} else {
		// For restore, the backup folder must exist
		_, err := fs.Stat(backupFolder)
		if err != nil {
			if appLogger != nil {
				appLogger.Logf("Backup folder does not exist or is not accessible: %v", err)
			} else {
				fmt.Printf("Backup folder does not exist or is not accessible: %v\n", err)
			}
			return
		}
	}

	files, err := fs.ReadDir(configFolder)
	if err != nil {
		if appLogger != nil {
			appLogger.Logf("Error reading config folder: %v", err)
		} else {
			fmt.Printf("Error reading config folder: %v\n", err)
		}
		return
	}

	homeDir, err := getHomeDirectory()
	if err != nil {
		if appLogger != nil {
			appLogger.Logf("Error getting home directory: %v", err)
		} else {
			fmt.Printf("Error getting home directory: %v\n", err)
		}
		return
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".cfg") {
			continue
		}

		// If an app name is specified, only process that app's configuration
		if appName != "" && !strings.Contains(strings.ToLower(file.Name()), strings.ToLower(appName)) {
			continue
		}

		// Parse the configuration file
		config, err := parseConfig(fs.Join(configFolder, file.Name()))
		if err != nil {
			if appLogger != nil {
				appLogger.Logf("Error parsing configuration file %s: %v", file.Name(), err)
			} else {
				fmt.Printf("Error parsing configuration file %s: %v\n", file.Name(), err)
			}
			continue
		}

		printer = NewPrinter(config.Name)

		if isBackup && !noCommands {
			for _, backupCommand := range config.PreBackupCommands {
				// In dry-run mode, just show what would be executed

				if dryRun {
					printer.Print("Would execute pre-backup command: %s", backupCommand)
					continue
				}

				// Execute command with recovery
				err := safeExecute("pre-backup command execution", func() error {
					if !executeCommandLine(backupCommand) {
						return fmt.Errorf("command execution failed")
					}
					return nil
				})

				if err != nil {
					if appLogger != nil {
						appLogger.Logf("Failed to execute pre-backup command: %v", err)
					} else {
						fmt.Printf("Failed to execute pre-backup command: %v\n", err)
					}
				}
			}
			for _, backupCommand := range config.PostBackupCommands {
				// In dry-run mode, just show what would be executed
				if dryRun {
					printer.Print("Would execute post-backup command: %s", backupCommand)
					continue
				}

				// Execute command with recovery
				err := safeExecute("post-backup command execution", func() error {
					if !executeCommandLine(backupCommand) {
						return fmt.Errorf("command execution failed")
					}
					return nil
				})

				if err != nil {
					if appLogger != nil {
						appLogger.Logf("Failed to execute post-backup command: %v", err)
					} else {
						fmt.Printf("Failed to execute post-backup command: %v\n", err)
					}
				}
			}
		}

		for _, configFile := range config.Files {
			// Replace ~ with the user's home directory
			if strings.HasPrefix(configFile, "~/") {
				configFile = fs.Join(homeDir, configFile[2:])
			} else if !strings.HasPrefix(configFile, "/") && !strings.HasPrefix(configFile, ".") {
				configFile = fs.Join(homeDir, configFile)
			} else if strings.HasPrefix(configFile, ".") {
				// Handle paths that start with . (like .config)
				configFile = fs.Join(homeDir, configFile)
			}

			// For backup operations, create a versioned directory
			var versionedBackupPath string
			var latestVersion string
			if isBackup {
				// Create timestamp-based directory
				timestamp := time.Now().Format("20060102-150405")
				versionedBackupPath = fs.Join(backupFolder, timestamp, config.Name, fs.Base(configFile))

				_, err := fs.Stat(configFile)
				if os.IsNotExist(err) {
					continue
				}

				if dryRun {
					printer.Print("Would create versioned backup directory: %s\n", fs.Dir(versionedBackupPath))
				} else {
					err = fs.MkdirAll(fs.Dir(versionedBackupPath), 0755)
					if err != nil {
						if appLogger != nil {
							appLogger.Logf("Failed to create versioned backup directory: %v", err)
						} else {
							fmt.Printf("Failed to create versioned backup directory: %v\n", err)
						}
						continue
					}
				}
			} else {
				// For restore operations, find the latest version
				latestVersion, err = getLatestVersionPath(backupFolder)
				if err != nil {
					if appLogger != nil {
						appLogger.Logf("Failed to find latest version: %v", err)
					} else {
						fmt.Printf("Failed to find latest version: %v\n", err)
					}
					continue
				}
				versionedBackupPath = fs.Join(latestVersion, fs.Base(configFile))
			}

			if isBackup {
				// Backup operation with recovery
				err := safeExecute("backup operation", func() error {
					// Check if the source file exists
					_, err := fs.Stat(configFile)
					if os.IsNotExist(err) {
						if dryRun {
							printer.Print("Would not back up %s because it doesn't exist", configFile)
							return nil
						}
						return nil
					} else if err != nil {
						printer.Print("Error accessing %s: %v\n", configFile, err)
						return nil
					}

					// In dry-run mode, just show what would be backed up
					if dryRun {
						printer.Print("Would back up %s to %s", configFile, versionedBackupPath)
						return nil
					}

					// Copy the file
					info, err := fs.Stat(configFile)
					if err != nil {
						printer.Print("Error getting file info: %v", err)
						return err
					}

					if info.IsDir() {
						err = copyDirectory(configFile, versionedBackupPath)
					} else {
						err = copyFile(configFile, versionedBackupPath)
					}

					if err != nil {
						printer.Print("Error backing up %s: %v", configFile, err)
						return err
					}

					printer.Print("Backed up %s to %s", configFile, versionedBackupPath)
					return nil
				})

				if err != nil {
					if appLogger != nil {
						appLogger.Logf("Backup operation failed for %s: %v", configFile, err)
					} else {
						fmt.Printf("Backup operation failed for %s: %v\n", configFile, err)
					}
				}
			} else {
				// Restore operation with recovery
				err := safeExecute("restore operation", func() error {
					// Check if the backup folder exists
					_, err := fs.Stat(latestVersion)
					if err != nil {
						return nil
					}

					// Check if the backup file exists
					_, err = fs.Stat(versionedBackupPath)
					if err != nil {
						printer.Print("Backup file not found: %s", versionedBackupPath)
						return err
					}

					// In dry-run mode, just show what would be restored
					if dryRun {
						printer.Print("Would restore %s to %s", versionedBackupPath, configFile)
						return nil
					}

					// Create the destination directory if it doesn't exist
					err = fs.MkdirAll(fs.Dir(configFile), 0755)
					if err != nil {
						printer.Print("Error creating destination directory: %v", err)
						return err
					}

					// Copy the file
					info, err := fs.Stat(versionedBackupPath)
					if err != nil {
						printer.Print("Error getting file info: %v", err)
						return err
					}

					if info.IsDir() {
						err = copyDirectory(versionedBackupPath, configFile)
					} else {
						err = copyFile(versionedBackupPath, configFile)
					}

					if err != nil {
						printer.Print("Error restoring %s: %v", versionedBackupPath, err)
						return err
					}

					printer.Print("Restored %s to %s", versionedBackupPath, configFile)
					return nil
				})

				if err != nil {
					if appLogger != nil {
						appLogger.Logf("Restore operation failed for %s: %v", configFile, err)
					} else {
						fmt.Printf("Restore operation failed for %s: %v\n", configFile, err)
					}
				}
			}
		}

		if !isBackup && !noCommands {
			for _, restoreCommand := range config.PreRestoreCommands {
				// In dry-run mode, just show what would be executed
				if dryRun {
					printer.Print("Would execute pre-restore command: %s", restoreCommand)
					continue
				}

				// Execute command with recovery
				err := safeExecute("pre-restore command execution", func() error {
					if !executeCommandLine(restoreCommand) {
						return fmt.Errorf("command execution failed")
					}
					return nil
				})

				if err != nil {
					if appLogger != nil {
						appLogger.Logf("Failed to execute pre-restore command: %v", err)
					} else {
						fmt.Printf("Failed to execute pre-restore command: %v\n", err)
					}
				}
			}
			for _, restoreCommand := range config.PostRestoreCommands {
				// In dry-run mode, just show what would be executed
				if dryRun {
					printer.Print("Would execute post-restore command: %s", restoreCommand)
					continue
				}

				// Execute command with recovery
				err := safeExecute("post-restore command execution", func() error {
					if !executeCommandLine(restoreCommand) {
						return fmt.Errorf("command execution failed")
					}
					return nil
				})

				if err != nil {
					if appLogger != nil {
						appLogger.Logf("Failed to execute post-restore command: %v", err)
					} else {
						fmt.Printf("Failed to execute post-restore command: %v\n", err)
					}
				}
			}
		}
	}
	// Cleanup old versions if needed
	if isBackup && versionsToKeep > 0 {
		if dryRun {
			if appLogger != nil {
				appLogger.Logf("Would clean up old versions in %s (keeping %d newest versions)", backupFolder, versionsToKeep)
			} else {
				fmt.Printf("Would clean up old versions in %s (keeping %d newest versions)\n", backupFolder, versionsToKeep)
			}
		} else {
			err := cleanupOldVersions(backupFolder, versionsToKeep)
			if err != nil {
				if appLogger != nil {
					appLogger.Logf("Failed to cleanup old versions: %v", err)
				} else {
					fmt.Printf("Failed to cleanup old versions: %v\n", err)
				}
			}
		}
	}
}

// executeCommandLine executes a command line with recovery
func executeCommandLine(commandLine string) bool {
	if commandLine == "" {
		printer.Print("No command provided")
		return true
	}

	// Define handlers for stdout and stderr
	stdoutHandler := func(line string) {
		printer.Print("  → %s", line)
	}

	stderrHandler := func(line string) {
		printer.Print("  ⚠ %s", line)
	}

	// Execute the command with real-time output handling
	result := cmdExecutor.ExecuteWithCallback(commandLine, stdoutHandler, stderrHandler)

	if !result {
		printer.Print("Error executing command %s", commandLine)
		return false
	}

	// Print the completion message
	printer.Print("Command %s executed", commandLine)

	return true
}

// getEnvWithDefault returns the value of an environment variable or a default value if it is not set
func getEnvWithDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func main() {
	// Parse command-line flags first to get the logFilePath
	logFilePath := flag.String("logfile", "", "Optional: Path to log file. If provided, logs will be written to this file in addition to console output")

	// We need to do a preliminary parse to get the logFilePath
	flag.Parse()

	// Initialize logger early
	var err error
	appLogger, err = logger.NewLogger(*logFilePath)
	if err != nil {
		fmt.Printf("Error initializing logger: %v\n", err)
		os.Exit(1)
	}
	defer appLogger.Close()

	// Reset flags for proper parsing later
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Add panic recovery for the main function
	defer func() {
		if r := recover(); r != nil {
			appLogger.Logf("Panic recovered in main: %v\nStack trace: %s", r, string(debug.Stack()))
			os.Exit(1)
		}
	}()

	appLogger.Logf("SettingsSentry v%s", Version)

	icloud_path, err := get_iCloud_folder_location()
	if err != nil {
		if appLogger != nil {
			appLogger.Logf("Error: iCloud path not found - %v", err)
		} else {
			fmt.Printf("Error: iCloud path not found - %v\n", err)
		}
		return // Add return here to prevent proceeding without a valid path
	}

	icloud_path = filepath.Join(icloud_path, ".settingssentry_backups")

	// Get environment variables with defaults
	envConfigFolder := getEnvWithDefault("SETTINGSSENTRY_CONFIG", "configs")
	envBackupFolder := getEnvWithDefault("SETTINGSSENTRY_BACKUP", icloud_path)
	envAppName := os.Getenv("SETTINGSSENTRY_APP")
	envNoCommands := os.Getenv("SETTINGSSENTRY_NO_COMMANDS") == "true"
	envDryRun := os.Getenv("SETTINGSSENTRY_DRY_RUN") == "true"

	// Define shared command-line flags with environment variable defaults
	configFolder := flag.String("config", envConfigFolder, "Path to the configuration folder (env: SETTINGSSENTRY_CONFIG)")
	backupFolder := flag.String("backup", envBackupFolder, "Path to the backup folder (env: SETTINGSSENTRY_BACKUP)")
	appName := flag.String("app", envAppName, "Optional: Name of the application to process (env: SETTINGSSENTRY_APP)")
	noCommands := flag.Bool("nocommands", envNoCommands, "Optional: Prevent pre-backup/restore commands execution (env: SETTINGSSENTRY_NO_COMMANDS)")
	dryRunFlag := flag.Bool("dry-run", envDryRun, "Optional: Perform a dry run without making any changes (env: SETTINGSSENTRY_DRY_RUN)")
	versionsToKeep := flag.Int("versions", 1, "Number of backup versions to keep")
	// logFilePath is already defined at the beginning of main

	// Define a function to display help information
	showHelp := func() {
		appLogger.Logf("Usage: SettingsSentry <action> [-config=<path>] [-backup=<path>] [-app=<n>] [-nocommands] [-logfile=<path>] [-dry-run]")
		appLogger.Logf("Actions:")
		appLogger.Logf("  backup   - Backup configuration files to the specified backup folder")
		appLogger.Logf("  restore  - Restore the files to their original locations")
		appLogger.Logf("  install  - Install the application as a CRON job that runs at every reboot (you can also provide a valid cron expression as parameter)")
		appLogger.Logf("  remove   - Remove the previously installed CRON job")
		appLogger.Logf("Use -logfile=<path> to enable logging to a file. This will write logs to the specified file in addition to console output.")
		appLogger.Logf("If -logfile is not provided, logs will only be written to the console.")
		appLogger.Logf("Default values:")
		appLogger.Logf("  Configurations: %s", envConfigFolder)
		appLogger.Logf("  Backups: %s", envBackupFolder)
		appLogger.Logf("Documentation available at https://github.com/sstraus/SettingsSentry")

		installed, err := cronjob.IsCronJobInstalled()
		if err != nil {
			appLogger.Logf("Error checking CRON job installation: %v", err)
			return
		}

		if installed {
			appLogger.Logf("CRON job is currently installed - the application will perform backups at every reboot")
		}
	}

	// Override the default flag.Usage function to use our custom help
	flag.Usage = showHelp

	// Check if no arguments or -h flag is provided
	if len(os.Args) < 2 || (len(os.Args) == 2 && (os.Args[1] == "-h" || os.Args[1] == "--help")) {
		showHelp()
		return
	}

	// Check if the second argument is -h or --help
	if len(os.Args) >= 3 && (os.Args[2] == "-h" || os.Args[2] == "--help") {
		showHelp()
		return
	}

	// Parse flags for custom handling based on the specified action
	if err := flag.CommandLine.Parse(os.Args[2:]); err != nil {
		fmt.Printf("Error parsing flags: %v\n", err)
		return
	}

	// Logger is already initialized at the beginning of main

	// Update dry run flag
	dryRun = *dryRunFlag

	action := os.Args[1]

	switch action {
	case "backup":
		processConfiguration(*configFolder, *backupFolder, *appName, true, *noCommands, *versionsToKeep)
	case "restore":
		processConfiguration(*configFolder, *backupFolder, *appName, false, *noCommands, *versionsToKeep)
	case "install":
		// Check if a cron expression is provided
		cronExpression := ""
		if len(os.Args) > 2 && !strings.HasPrefix(os.Args[2], "-") {
			cronExpression = os.Args[2]
		}

		err := cronjob.InstallCronJob(cronExpression)
		if err != nil {
			if appLogger != nil {
				appLogger.Logf("Failed to install cron job: %v", err)
			} else {
				fmt.Printf("Failed to install cron job: %v\n", err)
			}
			os.Exit(1)
		}
		if appLogger != nil {
			appLogger.Logf("CRON job installed successfully")
		} else {
			fmt.Printf("CRON job installed successfully\n")
		}
	case "remove":
		err := cronjob.RemoveCronJob()
		if err != nil {
			if appLogger != nil {
				appLogger.Logf("Failed to remove cron job: %v", err)
			} else {
				fmt.Printf("Failed to remove cron job: %v\n", err)
			}
			os.Exit(1)
		}
		if appLogger != nil {
			appLogger.Logf("CRON job removed successfully")
		} else {
			fmt.Printf("CRON job removed successfully\n")
		}
	default:
		if appLogger != nil {
			appLogger.Logf("Invalid action specified. Please use one of the following: 'backup', 'restore', 'install', or 'remove'")
		} else {
			fmt.Printf("Invalid action specified. Please use one of the following: 'backup', 'restore', 'install', or 'remove'\n")
		}
		os.Exit(1)
	}
}
