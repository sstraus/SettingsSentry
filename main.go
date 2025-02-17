package main

import (
	"SettingsSentry/constants"
	cronjob "SettingsSentry/cron"
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	AppName     string
	ConfigFiles []string
}

// getHomeDirectory returns the user's home directory.
func getHomeDirectory() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return homeDir, nil
}

func get_iCloud_folder_location() (string, error) {
	icloud_path, err := getHomeDirectory()
	if err != nil {
		return "", errors.New(constants.ERROR_UNABLE_TO_FIND_HOME_DIR)
	}
	icloud_path = filepath.Join(icloud_path, "Library", "Mobile Documents")

	icloud_home, err := filepath.EvalSymlinks(icloud_path)
	if err != nil {
		return "", errors.New(constants.ERROR_UNABLE_TO_FIND_STORAGE + " (" + strings.ToLower(os.Getenv("USER")) + "/iCloud Drive)")
	}

	if !filepath.IsAbs(icloud_home) {
		return "", errors.New(constants.ERROR_UNABLE_TO_FIND_STORAGE + " (" + strings.ToLower(os.Getenv("USER")) + "/iCloud Drive)")
	}

	// On some systems, the iCloud path might be a symlink to ~/Library/Mobile Documents/
	icloud_home, err = filepath.EvalSymlinks(filepath.Join(icloud_path, "com~apple~CloudDocs/"))
	if err == nil {
		return icloud_home, nil
	}

	return icloud_home, nil
}

// parseConfig reads and parses the content of a .cfg file into a Config struct.
func parseConfig(filePath string) (Config, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return Config{}, err
	}
	defer file.Close()

	var config Config
	scanner := bufio.NewScanner(file)
	section := ""

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.Trim(line, "[]")
		} else if section == "application" && strings.HasPrefix(line, "name =") {
			config.AppName = strings.TrimSpace(strings.SplitN(line, "=", 2)[1])
		} else if section == "configuration_files" {
			config.ConfigFiles = append(config.ConfigFiles, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return Config{}, err
	}
	return config, nil
}

// copyFile copies a single file from src to dst.
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	err = os.MkdirAll(filepath.Dir(dst), 0755)
	if err != nil {
		return err
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// copyDirectory recursively copies a directory from src to dst.
func copyDirectory(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, 0755)
		} else {
			return copyFile(path, dstPath)
		}
	})
}

// processConfiguration processes configuration files for backup or restore.
func processConfiguration(configFolder, backupFolder, appName string, isBackup bool) {
	files, err := os.ReadDir(configFolder)
	if err != nil {
		fmt.Printf("Error reading config folder: %v\n", err)
		return
	}

	homeDir, err := getHomeDirectory()
	if err != nil {
		fmt.Printf("Error getting home directory: %v\n", err)
		return
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".cfg" {
			continue
		}

		configPath := filepath.Join(configFolder, file.Name())
		config, err := parseConfig(configPath)
		if err != nil {
			fmt.Printf("Error parsing config file %s: %v\n", file.Name(), err)
			continue
		}

		// If an app name is specified, skip other apps
		if appName != "" && config.AppName != appName {
			continue
		}

		for _, configFile := range config.ConfigFiles {
			srcPath := filepath.Join(homeDir, configFile)
			dstPath := filepath.Join(backupFolder, config.AppName, filepath.Base(configFile))

			if isBackup {
				info, err := os.Stat(srcPath)
				if os.IsNotExist(err) {
					//fmt.Printf("Skipping non-existent path: %s\n", srcPath)
					continue
				} else if err != nil {
					fmt.Printf("Error accessing %s: %v\n", srcPath, err)
					continue
				}

				if info.IsDir() {
					err = copyDirectory(srcPath, dstPath)
					if err != nil {
						fmt.Printf("Error backing up directory %s: %v\n", srcPath, err)
					} else {
						fmt.Printf("Backed up directory %s to %s\n", srcPath, dstPath)
					}
				} else {
					err = copyFile(srcPath, dstPath)
					if err != nil {
						fmt.Printf("Error backing up file %s: %v\n", srcPath, err)
					} else {
						fmt.Printf("Backed up file %s to %s\n", srcPath, dstPath)
					}
				}
			} else { // Restore
				info, err := os.Stat(dstPath)
				if os.IsNotExist(err) {
					fmt.Printf("Skipping non-existent backup: %s\n", dstPath)
					continue
				} else if err != nil {
					fmt.Printf("Error accessing %s: %v\n", dstPath, err)
					continue
				}

				if info.IsDir() {
					err = copyDirectory(dstPath, srcPath)
					if err != nil {
						fmt.Printf("Error restoring directory %s: %v\n", dstPath, err)
					} else {
						fmt.Printf("Restored directory %s to %s\n", dstPath, srcPath)
					}
				} else {
					err = copyFile(dstPath, srcPath)
					if err != nil {
						fmt.Printf("Error restoring file %s: %v\n", dstPath, err)
					} else {
						fmt.Printf("Restored file %s to %s\n", dstPath, srcPath)
					}
				}
			}
		}
	}
}

func main() {
	icloud_path, err := get_iCloud_folder_location()
	if err != nil {
		fmt.Printf("Error: iCloud path not found - %v\n", err)
		return // Add return here to prevent proceeding without a valid path
	}

	icloud_path = filepath.Join(icloud_path, constants.DEFAULT_BACKUP_PATH)

	// Define shared command-line flags
	configFolder := flag.String("config", "./configs", "Path to the configuration folder")
	backupFolder := flag.String("backup", icloud_path, "Path to the backup folder")
	appName := flag.String("app", "", "Optional: Name of the application to process")
	fmt.Println("SettingsSentry")
	if len(os.Args) < 2 {
		fmt.Println("Securely archive and reinstate your macOS application configurations, simplifying system recovery processes.")
		fmt.Println("")
		fmt.Println("Usage: main <action> [-config=<path>] [-backup=<path>] [-app=<name>]")
		fmt.Println("")
		fmt.Println("Actions:")
		fmt.Println("  backup   - Backup configuration files to the specified backup folder")
		fmt.Println("  restore  - Restore the files to their original locations")
		fmt.Println("  install  - Install the application as a CRON job that runs at every reboot")
		fmt.Println("  remove   - Remove the previously installed CRON job")
		fmt.Println("")
		fmt.Println("Default values:")
		fmt.Println("  Configurations: ./configs")
		fmt.Printf("  Backups: iCloud Drive/%s\n", constants.DEFAULT_BACKUP_PATH)

		installed, err := cronjob.IsCronJobInstalled()
		if err != nil {
			fmt.Printf("Error checking CRON job installation: %v\n", err)
			return
		}

		if installed {
			fmt.Println("CRON job is currently installed - the application will perform backups at every reboot")
		}
		return
	}

	// Parse flags for custom handling based on the specified action
	flag.CommandLine.Parse(os.Args[2:])

	action := os.Args[1]
	switch action {
	case "backup":
		processConfiguration(*configFolder, *backupFolder, *appName, true)
	case "restore":
		processConfiguration(*configFolder, *backupFolder, *appName, false)
	case "install":
		err := cronjob.AddCronJob()
		if err != nil {
			fmt.Printf("Error while installing CRON job: %v\n", err)
		}
	case "remove":
		err := cronjob.RemoveCronJob()
		if err != nil {
			fmt.Printf("Error while removing CRON job: %v\n", err)
		}
	default:
		fmt.Println("Invalid action specified. Please use one of the following: 'backup', 'restore', 'install', or 'remove'")
	}
}
