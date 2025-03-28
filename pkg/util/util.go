package util

import (
	"SettingsSentry/interfaces"
	"SettingsSentry/logger"
	"embed"
	iofs "io/fs"
	"os"
	"path/filepath"
)

var (
	AppLogger   *logger.Logger
	Fs          interfaces.FileSystem
	CmdExecutor interfaces.CommandExecutor
	DryRun      bool
	Version     string = "1.1.5" // Default version, can be overridden by build flags
)

func GetEnvWithDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func EmbeddedFallback(rootEmbedFS embed.FS) iofs.FS {
	fsys, err := iofs.Sub(rootEmbedFS, "configs")
	if err != nil {
		// Log the error but still return the root embed FS as a last resort.
		if AppLogger != nil {
			AppLogger.Logf("Failed to access embedded 'configs' subdirectory: %v. Falling back to root embed FS.", err)
		}
		return rootEmbedFS // Return the original root FS on error
	}
	return fsys
}

func InitGlobals(logger *logger.Logger, fs interfaces.FileSystem, cmdExec interfaces.CommandExecutor, dryRun bool, version string) {
	AppLogger = logger
	Fs = fs
	CmdExecutor = cmdExec
	DryRun = dryRun
	// Allow overriding default version
	if version != "" {
		Version = version
	}
}

func ExtractEmbeddedConfigs(rootEmbedFS embed.FS) error {
	exePath, err := os.Executable()
	if err != nil {
		return AppLogger.LogErrorf("failed to get executable path: %w", err)
	}
	exeDir := filepath.Dir(exePath)
	targetDir := filepath.Join(exeDir, "configs")

	AppLogger.Logf("Extracting embedded configs to: %s", targetDir)

	if _, err := os.Stat(targetDir); !os.IsNotExist(err) {
		AppLogger.Logf("Target directory '%s' already exists. Files might be overwritten.", targetDir)
	} else {
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return AppLogger.LogErrorf("failed to create target directory '%s': %w", targetDir, err)
		}
	}

	embedFS := EmbeddedFallback(rootEmbedFS)

	err = iofs.WalkDir(embedFS, ".", func(path string, d iofs.DirEntry, err error) error {
		if err != nil {
			// Use %v for WalkDir errors as they might not be wrappable
			return AppLogger.LogErrorf("error accessing path %s in embed FS: %v", path, err)
		}
		if path == "." {
			return nil
		}
		if d.IsDir() {
			return nil
		}

		destPath := filepath.Join(targetDir, path)

		content, readErr := iofs.ReadFile(embedFS, path)
		if readErr != nil {
			return AppLogger.LogErrorf("failed to read embedded file '%s': %w", path, readErr)
		}

		writeErr := os.WriteFile(destPath, content, 0644)
		if writeErr != nil {
			return AppLogger.LogErrorf("failed to write file '%s': %w", destPath, writeErr)
		}
		AppLogger.Logf("  Extracted: %s", destPath)
		return nil
	})

	if err != nil {
		return err
	}

	AppLogger.Logf("Configs extracted successfully.")
	return nil
}
