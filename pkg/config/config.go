package config

import (
	"SettingsSentry/interfaces"
	"SettingsSentry/logger"
	"bufio"
	"errors"
	"fmt"
	iofs "io/fs"
	"os"
	"path/filepath"
	"strings"
)

var (
	AppLogger *logger.Logger
	Fs        interfaces.FileSystem
)

var GetHomeDirectory = func() (string, error) {
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		return "", errors.New("unable to find home directory. HOME environment variable may not be set")
	}
	return homeDir, nil
}

type Config struct {
	Name                string
	Files               []string
	PreBackupCommands   []string
	PostBackupCommands  []string
	PreRestoreCommands  []string
	PostRestoreCommands []string
}

func GetXDGConfigHome() (string, error) {
	homeDir, err := GetHomeDirectory()
	if err != nil {
		return "", err
	}

	xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfigHome == "" {
		xdgConfigHome = filepath.Join(homeDir, ".config")
	}

	if !strings.HasPrefix(xdgConfigHome, homeDir) {
		if AppLogger != nil {
			return "", AppLogger.LogErrorf("$XDG_CONFIG_HOME: %s must be somewhere within your home directory: %s", xdgConfigHome, homeDir)
		}
		return "", errors.New("$XDG_CONFIG_HOME must be within home directory")
	}

	return xdgConfigHome, nil
}

func GetICloudFolderLocation() (string, error) {
	if Fs == nil {
		return "", errors.New("filesystem interface (Fs) is not initialized")
	}
	homeDir, err := GetHomeDirectory()
	if err != nil {
		return "", err
	}

	iCloudPath := Fs.Join(homeDir, "Library", "Mobile Documents", "com~apple~CloudDocs")

	_, err = Fs.Stat(iCloudPath)
	if err != nil {
		if AppLogger != nil {
			return "", AppLogger.LogErrorf("iCloud Drive folder not found: %w", err)
		}
		return "", errors.New("iCloud Drive folder not found")
	}

	resolvedPath, err := Fs.EvalSymlinks(iCloudPath)
	if err != nil {
		if AppLogger != nil {
			return "", AppLogger.LogErrorf("failed to resolve iCloud Drive path: %w", err)
		}
		return "", errors.New("failed to resolve iCloud Drive path")
	}

	return resolvedPath, nil
}

func ExpandEnvVars(value string) string {
	result := os.Expand(value, func(key string) string {
		return os.Getenv(key)
	})

	return result
}

func ValidateConfig(config Config) error {
	if Fs == nil {
		return errors.New("filesystem interface (Fs) is not initialized")
	}
	if config.Name == "" {
		return errors.New("application name is required in configuration")
	}

	if len(config.Files) == 0 {
		return errors.New("at least one configuration file must be specified")
	}

	for _, file := range config.Files {
		if strings.TrimSpace(file) == "" {
			return errors.New("empty configuration file path specified")
		}

		if strings.HasPrefix(file, "~") {
			homeDir, err := GetHomeDirectory()
			if err != nil {
				if AppLogger != nil {
					return AppLogger.LogErrorf("failed to expand ~ in path %s: %w", file, err)
				}
				return errors.New("failed to get home directory for path expansion")
			}

			expandedPath := strings.Replace(file, "~", homeDir, 1)
			if strings.Contains(expandedPath, "*") || strings.Contains(expandedPath, "?") {
				dir := filepath.Dir(expandedPath)
				if _, err := Fs.Stat(dir); err != nil {
					if AppLogger != nil {
						return AppLogger.LogErrorf("directory for glob pattern %s does not exist: %w", file, err)
					}
					return errors.New("directory for glob pattern does not exist")
				}
			}
		}
	}

	return nil
}

func ParseConfig(filesystem iofs.FS, filePath string) (Config, error) {
	var config Config

	data, err := iofs.ReadFile(filesystem, filePath)
	if err != nil {
		if AppLogger != nil {
			return config, AppLogger.LogErrorf("failed to read config file '%s': %w", filePath, err)
		}
		return config, errors.New("failed to read config file")
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	var section string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(line[1 : len(line)-1])
			continue
		}

		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			key := strings.TrimSpace(strings.ToLower(parts[0]))
			value := strings.TrimSpace(parts[1])

			value = ExpandEnvVars(value)

			if section == "application" && key == "name" {
				config.Name = value
				continue
			}
		}

		switch section {
		case "app":
			config.Name = ExpandEnvVars(line)
		case "application":
			if !strings.Contains(line, "=") {
				config.Name = ExpandEnvVars(line)
			}
		case "files", "configuration_files":
			config.Files = append(config.Files, ExpandEnvVars(line))

		case "xdg_configuration_files":
			path := ExpandEnvVars(line)
			if strings.HasPrefix(path, "/") {
				if AppLogger != nil {
					return config, AppLogger.LogErrorf("unsupported absolute path in xdg_configuration_files: %s", path)
				}
				return config, errors.New("unsupported absolute path in xdg_configuration_files")
			}

			xdgConfigHome, err := GetXDGConfigHome()
			if err != nil {
				if AppLogger != nil {
					return config, AppLogger.LogErrorf("error getting XDG_CONFIG_HOME: %w", err)
				}
				return config, errors.New("error getting XDG_CONFIG_HOME")
			}

			fullPath := filepath.Join(xdgConfigHome, path)

			homeDir, err := GetHomeDirectory()
			if err != nil {
				if AppLogger != nil {
					return config, AppLogger.LogErrorf("error getting home directory: %w", err)
				}
				return config, errors.New("error getting home directory")
			}

			relativePath := strings.Replace(fullPath, homeDir+"/", "", 1)

			config.Files = append(config.Files, relativePath)
		case "backup", "backup_commands", "pre_backup_commands":
			config.PreBackupCommands = append(config.PreBackupCommands, ExpandEnvVars(line))
		case "post_backup_commands":
			config.PostBackupCommands = append(config.PostBackupCommands, ExpandEnvVars(line))
		case "restore", "restore_commands", "pre_restore_commands":
			config.PreRestoreCommands = append(config.PreRestoreCommands, ExpandEnvVars(line))
		case "post_restore_commands":
			config.PostRestoreCommands = append(config.PostRestoreCommands, ExpandEnvVars(line))
		}
	}

	if err := scanner.Err(); err != nil {
		if AppLogger != nil {
			return config, AppLogger.LogErrorf("error scanning config file: %w", err)
		}
		return config, errors.New("error scanning config file")
	}

	if validationErr := ValidateConfig(config); validationErr != nil {
		detailedError := fmt.Errorf("invalid configuration in '%s': %w", filePath, validationErr)
		if AppLogger != nil {
			return config, AppLogger.LogErrorf(detailedError.Error())
		}
		return config, detailedError
	}

	return config, nil
}
