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
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	Version = "1.1.4"
)

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
		fmt.Printf("%s "+format, append([]interface{}{p.printAppName}, args...)...)
		p.firstPrint = false // Reset the state after printing for the first time
	} else {
		// Normal print for subsequent calls
		fmt.Printf(format, args...)
	}
}

func (p *Printer) Reset() {
	p.firstPrint = true
}

// Global printer variable
var printer *Printer

type Config struct {
	AppName         string
	ConfigFiles     []string
	BackupCommands  []string
	RestoreCommands []string
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
		} else if section == "backup_commands" {
			config.BackupCommands = append(config.BackupCommands, line)
		} else if section == "restore_commands" {
			config.RestoreCommands = append(config.RestoreCommands, line)
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
func processConfiguration(configFolder, backupFolder, appName string, isBackup bool, noCommands bool) {
	// Check if configFolder does not contain "/"
	if !strings.Contains(configFolder, string(os.PathSeparator)) {
		exePath, err := os.Executable() // Get the path of the executable
		if err != nil {
			fmt.Println("Error getting executable path:", err)
			return
		}
		configFolder = filepath.Join(filepath.Dir(exePath), configFolder) // Append to the executable directory
	}
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

		printer = NewPrinter(config.AppName)

		if isBackup && !noCommands {
			for _, backupCommand := range config.BackupCommands {
				// Split the command into command and its arguments
				executeCommandLine(backupCommand)
			}
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
					printer.Print("Error accessing %s: %v\n", srcPath, err)
					continue
				}

				if info.IsDir() {
					err = copyDirectory(srcPath, dstPath)
					if err != nil {
						printer.Print("Error backing up directory %s: %v\n", srcPath, err)
					} else {
						printer.Print("Backed up directory %s to %s\n", srcPath, dstPath)
					}
				} else {
					err = copyFile(srcPath, dstPath)
					if err != nil {
						printer.Print("Error backing up file %s: %v\n", srcPath, err)
					} else {
						printer.Print("Backed up file %s to %s\n", srcPath, dstPath)
					}
				}
			} else { // Restore
				info, err := os.Stat(dstPath)
				if os.IsNotExist(err) {
					printer.Print("Skipping non-existent backup: %s\n", dstPath)
					continue
				} else if err != nil {
					printer.Print("Error accessing %s: %v\n", dstPath, err)
					continue
				}

				if info.IsDir() {
					err = copyDirectory(dstPath, srcPath)
					if err != nil {
						printer.Print("Error restoring directory %s: %v\n", dstPath, err)
					} else {
						printer.Print("Restored directory %s to %s\n", dstPath, srcPath)
					}
				} else {
					err = copyFile(dstPath, srcPath)
					if err != nil {
						printer.Print("Error restoring file %s: %v\n", dstPath, err)
					} else {
						fmt.Printf("Restored file %s to %s\n", dstPath, srcPath)
					}
				}
			}
			if !isBackup && !noCommands {
				for _, restoreCommand := range config.RestoreCommands {
					executeCommandLine(restoreCommand)
				}
			}
		}
	}
}

func executeCommandLine(commandLine string) bool {
	parts := strings.Fields(commandLine)

	if len(parts) == 0 {
		printer.Print("No command provided")
		return true
	}

	// The first part is the command, and the rest are arguments
	cmdName := parts[0]
	cmdArgs := parts[1:]

	// Define the command with its arguments
	cmd := exec.Command(cmdName, cmdArgs...) // Use cmdArgs with "..."

	// Set the Stdout and Stderr to os.Stdout and os.Stderr respectively
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start the command and wait for it to finish
	if err := cmd.Run(); err != nil {
		printer.Print("Error executing command %s: %s\n", commandLine, err)
		return false
	}

	// Print the output
	printer.Print("Command %s executed\n", commandLine)

	return true
}

func main() {
	icloud_path, err := get_iCloud_folder_location()
	if err != nil {
		fmt.Printf("Error: iCloud path not found - %v\n", err)
		return // Add return here to prevent proceeding without a valid path
	}

	icloud_path = filepath.Join(icloud_path, constants.DEFAULT_BACKUP_PATH)

	// Define shared command-line flags
	configFolder := flag.String("config", "configs", "Path to the configuration folder")
	backupFolder := flag.String("backup", icloud_path, "Path to the backup folder")
	appName := flag.String("app", "", "Optional: Name of the application to process")
	noCommands := flag.Bool("nocommands", false, "Optional: Prevent pre-backup/restore commands execution")
	fmt.Println("\033[1mSettingsSentry v" + Version + "\033[0m")
	if len(os.Args) < 2 {
		fmt.Println("Securely archive and reinstate your macOS application configurations, simplifying system recovery processes.")
		fmt.Println("")
		fmt.Println("Usage: main <action> [-config=<path>] [-backup=<path>] [-app=<name>] [-nocommands]")
		fmt.Println("")
		fmt.Println("Actions:")
		fmt.Println("  backup   - Backup configuration files to the specified backup folder")
		fmt.Println("  restore  - Restore the files to their original locations")
		fmt.Println("  install  - Install the application as a CRON job that runs at every reboot (you can also provide a valid cron expression as parameter)")
		fmt.Println("  remove   - Remove the previously installed CRON job")
		fmt.Println("")
		fmt.Println("Default values:")
		fmt.Println("  Configurations: configs")
		fmt.Printf("  Backups: iCloud Drive/%s\n", constants.DEFAULT_BACKUP_PATH)
		fmt.Println("")
		fmt.Println("Documentation available at https://github.com/sstraus/SettingsSentry")

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
		processConfiguration(*configFolder, *backupFolder, *appName, true, *noCommands)
	case "restore":
		processConfiguration(*configFolder, *backupFolder, *appName, false, *noCommands)
	case "install":
		when := ""
		if len(os.Args) > 2 {
			when = os.Args[2] // get the argument if provided
		} else {
			when = "@reboot" // Example default cron expression (runs daily at midnight)
		}

		var commandOption string
		if *noCommands {
			commandOption = "-nocommands"
		} else {
			commandOption = ""
		}
		// Pass when as a pointer
		err := cronjob.AddCronJob(&when, commandOption)
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
