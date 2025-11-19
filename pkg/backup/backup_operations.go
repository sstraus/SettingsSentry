package backup

import (
	"SettingsSentry/interfaces"
	"SettingsSentry/logger"
	"SettingsSentry/pkg/command"
	"SettingsSentry/pkg/config"
	"SettingsSentry/pkg/printer"
	"archive/zip"
	"fmt"
	iofs "io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// BackupContext holds the context for backup/restore operations
type BackupContext struct {
	ConfigFolder   string
	BackupFolder   string
	AppNames       []string
	IsBackup       bool
	Commands       bool
	VersionsToKeep int
	ZipBackup      bool
	Password       string
	HomeDir        string
	Timestamp      string
	StagingDir     string
	Logger         *logger.Logger
	FS             interfaces.FileSystem
	Printer        *printer.Printer
}

// NewBackupContext creates a new backup context with validated paths
func NewBackupContext(configFolder, backupFolder string, appNames []string, isBackup bool, commands bool, versionsToKeep int, zipBackup bool, password string) (*BackupContext, error) {
	configFolder = config.ExpandEnvVars(configFolder)
	backupFolder = config.ExpandEnvVars(backupFolder)

	// Resolve relative config folder path
	if !strings.Contains(configFolder, string(os.PathSeparator)) {
		exePath, err := os.Executable()
		if err != nil {
			return nil, fmt.Errorf("error getting executable path: %w", err)
		}
		configFolder = Fs.Join(Fs.Dir(exePath), configFolder)
	}

	homeDir, err := config.GetHomeDirectory()
	if err != nil {
		return nil, fmt.Errorf("error getting home directory: %w", err)
	}

	timestamp := time.Now().Format("20060102-150405")

	ctx := &BackupContext{
		ConfigFolder:   configFolder,
		BackupFolder:   backupFolder,
		AppNames:       appNames,
		IsBackup:       isBackup,
		Commands:       commands,
		VersionsToKeep: versionsToKeep,
		ZipBackup:      zipBackup,
		Password:       password,
		HomeDir:        homeDir,
		Timestamp:      timestamp,
		Logger:         AppLogger,
		FS:             Fs,
		Printer:        Printer,
	}

	return ctx, nil
}

// SetupBackupDirectory creates backup directory or validates restore directory
func (ctx *BackupContext) SetupBackupDirectory() error {
	if ctx.IsBackup {
		if DryRun {
			ctx.Logger.Logf("Would create backup folder: %s", ctx.BackupFolder)
		} else {
			err := ctx.FS.MkdirAll(ctx.BackupFolder, 0755)
			if err != nil {
				return fmt.Errorf("failed to create backup folder: %w", err)
			}
		}
	} else {
		_, err := ctx.FS.Stat(ctx.BackupFolder)
		if err != nil {
			return fmt.Errorf("backup folder does not exist or is not accessible: %w", err)
		}
	}

	// Setup staging directory for zip backups
	if ctx.IsBackup && ctx.ZipBackup {
		stagingDir, err := os.MkdirTemp("", "settingssentry-zip-")
		if err != nil {
			return fmt.Errorf("failed to create temporary staging directory: %w", err)
		}
		ctx.StagingDir = stagingDir
		ctx.Logger.Logf("Using staging directory for zip backup: %s", stagingDir)
	}

	return nil
}

// LoadConfigFiles loads configuration files from the config folder
func (ctx *BackupContext) LoadConfigFiles() (iofs.FS, []iofs.DirEntry, error) {
	_, err := ctx.FS.Stat(ctx.ConfigFolder)
	if err != nil {
		return nil, nil, fmt.Errorf("config folder '%s' not found or inaccessible: %w", ctx.ConfigFolder, err)
	}

	ctx.Logger.Logf("Using config folder: %s", ctx.ConfigFolder)

	currentFS := os.DirFS(ctx.ConfigFolder)
	files, err := iofs.ReadDir(currentFS, ".")
	if err != nil {
		return nil, nil, fmt.Errorf("error reading config folder '%s': %w", ctx.ConfigFolder, err)
	}

	if len(files) == 0 {
		return nil, nil, fmt.Errorf("no configuration files found in '%s'", ctx.ConfigFolder)
	}

	return currentFS, files, nil
}

// FilterConfigFiles filters config files based on app names
func (ctx *BackupContext) FilterConfigFiles(files []iofs.DirEntry) []iofs.DirEntry {
	if len(ctx.AppNames) == 0 {
		return files
	}

	var filtered []iofs.DirEntry
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".cfg") {
			continue
		}

		currentFileNameLower := strings.ToLower(file.Name())
		for _, requestedAppName := range ctx.AppNames {
			if currentFileNameLower == strings.ToLower(requestedAppName)+".cfg" {
				filtered = append(filtered, file)
				break
			}
		}
	}
	return filtered
}

// LoadZipFileMap loads the file map from a zip archive for restore operations
func (ctx *BackupContext) LoadZipFileMap(zipPath string) (map[string]bool, *zip.ReadCloser, error) {
	zipReader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open zip backup '%s': %w", zipPath, err)
	}

	zipFileMap := make(map[string]bool)
	for _, f := range zipReader.File {
		zipFileMap[f.Name] = true
	}

	return zipFileMap, zipReader, nil
}

// ResolveConfigFilePath resolves the config file path relative to home directory
func (ctx *BackupContext) ResolveConfigFilePath(configFile string) string {
	if strings.HasPrefix(configFile, "~/") {
		return ctx.FS.Join(ctx.HomeDir, configFile[2:])
	} else if !strings.HasPrefix(configFile, "/") && !strings.HasPrefix(configFile, ".") {
		return ctx.FS.Join(ctx.HomeDir, configFile)
	} else if strings.HasPrefix(configFile, ".") {
		return ctx.FS.Join(ctx.HomeDir, configFile)
	}
	return configFile
}

// BackupFile backs up a single file
func (ctx *BackupContext) BackupFile(configFile, targetPath string) error {
	info, err := ctx.FS.Stat(configFile)
	if os.IsNotExist(err) {
		if DryRun {
			ctx.Printer.Print("Would skip backup of %s (doesn't exist)", configFile)
		}
		return nil
	} else if err != nil {
		return fmt.Errorf("error accessing %s: %w", configFile, err)
	}

	if DryRun {
		finalBackupPath := ctx.FS.Join(ctx.BackupFolder, ctx.Timestamp, ctx.FS.Base(targetPath))
		if ctx.ZipBackup {
			finalBackupPath = ctx.FS.Join(ctx.BackupFolder, ctx.Timestamp+".zip")
		}
		ctx.Printer.Print("Would back up %s to %s", configFile, finalBackupPath)
		return nil
	}

	if info.IsDir() {
		err = copyDirectory(configFile, targetPath)
	} else {
		err = copyFile(configFile, targetPath)
	}

	if err != nil {
		return fmt.Errorf("error backing up %s: %w", configFile, err)
	}

	finalBackupPath := ctx.FS.Join(ctx.BackupFolder, ctx.Timestamp, ctx.FS.Base(targetPath))
	if ctx.ZipBackup {
		finalBackupPath = ctx.FS.Join(ctx.BackupFolder, ctx.Timestamp+".zip", ctx.FS.Base(targetPath))
	}
	ctx.Printer.Print("Backed up %s to %s", configFile, finalBackupPath)
	return nil
}

// RestoreFile restores a single file
func (ctx *BackupContext) RestoreFile(versionedBackupPath, configFile, latestVersion string, latestIsZip bool) error {
	if DryRun {
		ctx.Printer.Print("Would restore %s to %s", versionedBackupPath, configFile)
		return nil
	}

	err := ctx.FS.MkdirAll(ctx.FS.Dir(configFile), 0755)
	if err != nil {
		return fmt.Errorf("error creating destination directory '%s': %w", ctx.FS.Dir(configFile), err)
	}

	if latestIsZip {
		err = extractFromZip(latestVersion, versionedBackupPath, configFile)
	} else {
		backupSourceInfo, statErr := ctx.FS.Stat(versionedBackupPath)
		if statErr != nil {
			return nil // Skip if source doesn't exist
		}
		if backupSourceInfo.IsDir() {
			err = copyDirectory(versionedBackupPath, configFile)
		} else {
			err = copyFile(versionedBackupPath, configFile)
		}
	}

	if err != nil {
		return fmt.Errorf("error restoring %s: %w", versionedBackupPath, err)
	}

	ctx.Printer.Print("Restored %s to %s", versionedBackupPath, configFile)
	return nil
}

// EncryptFile encrypts a file for backup
func (ctx *BackupContext) EncryptFile(configFile, targetPath string) error {
	plaintext, err := ctx.FS.ReadFile(configFile)
	if os.IsNotExist(err) {
		if DryRun {
			ctx.Printer.Print("Would skip encryption of %s (doesn't exist)", configFile)
		}
		return nil
	} else if err != nil {
		return fmt.Errorf("error reading source file %s for encryption: %w", configFile, err)
	}

	encryptedData, err := encrypt(plaintext, ctx.Password)
	if err != nil {
		return fmt.Errorf("error encrypting %s: %w", configFile, err)
	}

	encryptedTargetPath := targetPath + ".encrypted"

	if DryRun {
		ctx.Printer.Print("Would encrypt %s to %s", configFile, encryptedTargetPath)
		return nil
	}

	if err := ctx.FS.MkdirAll(ctx.FS.Dir(encryptedTargetPath), 0755); err != nil {
		return fmt.Errorf("failed to create target directory '%s' for encrypted file: %w", ctx.FS.Dir(encryptedTargetPath), err)
	}

	if err := ctx.FS.WriteFile(encryptedTargetPath, encryptedData, 0644); err != nil {
		return fmt.Errorf("error writing encrypted file %s: %w", encryptedTargetPath, err)
	}

	ctx.Printer.Print("Encrypted %s to %s", configFile, encryptedTargetPath)
	return nil
}

// DecryptFile decrypts a file for restore
func (ctx *BackupContext) DecryptFile(encryptedSourcePath, configFile, appName string, latestIsZip bool, latestVersion string) error {
	var encryptedData []byte
	var err error

	if latestIsZip {
		encryptedSourcePathInZip := filepath.ToSlash(ctx.FS.Join(appName, ctx.FS.Base(configFile))) + ".encrypted"
		tempFile, tempErr := os.CreateTemp("", "settingssentry-decrypt-")
		if tempErr != nil {
			return fmt.Errorf("failed to create temp file for zip extraction: %w", tempErr)
		}
		tempFilePath := tempFile.Name()
		if closeErr := tempFile.Close(); closeErr != nil {
			_ = os.Remove(tempFilePath)
			return fmt.Errorf("failed to close temp file handle: %w", closeErr)
		}
		defer func() {
			if removeErr := os.Remove(tempFilePath); removeErr != nil {
				ctx.Logger.Logf("Warning: Failed to remove temp decryption file %s: %v", tempFilePath, removeErr)
			}
		}()

		err = extractFromZip(latestVersion, encryptedSourcePathInZip, tempFilePath)
		if err != nil {
			return nil // Skip if not found in zip
		}
		encryptedData, err = os.ReadFile(tempFilePath)
	} else {
		encryptedData, err = ctx.FS.ReadFile(encryptedSourcePath)
	}

	if err != nil {
		return fmt.Errorf("error reading encrypted file %s: %w", encryptedSourcePath, err)
	}

	plaintext, err := decrypt(encryptedData, ctx.Password)
	if err != nil {
		return fmt.Errorf("error decrypting %s (wrong password or corrupt data?): %w", encryptedSourcePath, err)
	}

	if DryRun {
		ctx.Printer.Print("Would restore (decrypted) %s to %s", encryptedSourcePath, configFile)
		return nil
	}

	if err := ctx.FS.MkdirAll(ctx.FS.Dir(configFile), 0755); err != nil {
		return fmt.Errorf("error creating destination directory '%s': %w", ctx.FS.Dir(configFile), err)
	}

	if err := ctx.FS.WriteFile(configFile, plaintext, 0644); err != nil {
		return fmt.Errorf("error writing decrypted file %s: %w", configFile, err)
	}

	ctx.Printer.Print("Restored (decrypted) %s to %s", encryptedSourcePath, configFile)
	return nil
}

// ExecuteCommands executes pre/post backup or restore commands
func (ctx *BackupContext) ExecuteCommands(commands []string, commandType string) {
	for _, cmd := range commands {
		if DryRun {
			ctx.Printer.Print("Would execute %s command: %s", commandType, cmd)
			continue
		}
		err := command.SafeExecute(commandType+" command execution", func() error {
			if !command.ExecuteCommandLine(cmd) {
				return fmt.Errorf("command execution failed")
			}
			return nil
		})
		if err != nil {
			ctx.Logger.Logf("Failed to execute %s command: %v", commandType, err)
		}
	}
}

// FinalizeBackup finalizes the backup by creating zip archive and cleaning up old versions
func (ctx *BackupContext) FinalizeBackup() error {
	if ctx.IsBackup && ctx.ZipBackup {
		if ctx.StagingDir == "" {
			return fmt.Errorf("staging directory path is empty, cannot create zip archive")
		}
		targetZipPath := ctx.FS.Join(ctx.BackupFolder, ctx.Timestamp+".zip")
		ctx.Logger.Logf("Creating zip archive: %s", targetZipPath)
		if err := createZipArchive(ctx.StagingDir, targetZipPath); err != nil {
			return fmt.Errorf("failed to create zip archive: %w", err)
		}
		ctx.Logger.Logf("Successfully created zip archive: %s", targetZipPath)

		// Cleanup staging directory
		ctx.Logger.Logf("Cleaning up staging directory: %s", ctx.StagingDir)
		if err := ctx.FS.RemoveAll(ctx.StagingDir); err != nil {
			ctx.Logger.Logf("Error removing staging directory %s: %v", ctx.StagingDir, err)
		}
	}

	if ctx.IsBackup && ctx.VersionsToKeep > 0 {
		err := CleanupOldVersions(ctx.BackupFolder, ctx.VersionsToKeep)
		if err != nil {
			return fmt.Errorf("failed to cleanup old versions: %w", err)
		}
	}

	return nil
}