# SettingsSentry

__Securely archive and reinstate your macOS application configurations, simplifying system recovery processes.__

SettingsSentry is a lightweight and efficient tool written in Go to backup and restore macOS application configurations. It ensures your personalized settings are securely archived and easily reinstated, simplifying system recovery processes.

Inspired by [Mackup](https://github.com/lra/mackup), SettingsSentry was created to address compatibility issues with macOS Sonoma and later versions. Unlike Mackup, which no longer supports symlinked preference files and risks destroying user preferences, SettingsSentry provides a reliable solution.

## Features

- Backup configuration files to iCloud Drive or a specified folder.
- Restore configurations seamlessly to their original locations.
- Install a CRON job that runs at every system reboot.
- Remove the installed CRON job when no longer needed.
- Support for environment variables in configuration paths and values.
- Configuration validation to ensure all required fields are present.
- Versioned backups with timestamp-based directories.
- Dry-run mode to preview operations without making changes.
- Interactive wizard for creating new application configurations.
- Automatic detection of installed applications and their configuration files.
- Pre-defined templates for common macOS applications.

## Usage

./SettingsSentry `<action>` `<optional parameters>` [-config=`<path>`] [-backup=`<path>`] [-app=`<n>`] [-nocommands] [-dry-run] [-versions=`<n>`]

### Actions

- backup: Backup configuration files to the specified backup folder.
- restore: Restore the files to their original locations.
- install: Install the application as a CRON job that runs at every reboot.
    You can also provide a valid cron expression as a parameter to customize the schedule (0 9 * * *). Use [cronhub](https://crontab.cronhub.io) to generate a valid one.
- remove: Remove the previously installed CRON job.
- wizard: Start the configuration wizard to create a new application configuration.
- detect: Detect installed applications and create configurations for them.
- template: Create configuration from a template (use 'template list' to see available templates).

### Default Values

Configurations: configs
Backups: iCloud Drive/SettingsSentry

#### Options

- `--config` `<path>`: Path to the configuration folder (default: `configs`).

- `--backup` `<path>`: Path to the backup folder (default: `iCloud Drive/.settingssentry_backups`).

- `--app` `<n>`: Optional name of the application to process.

- `-nocommands`: Prevent command execution during backup or restore.

- `--dry-run`: Perform a dry run without making any changes.

- `--versions` `<n>`: Number of backup versions to keep (default: 1, 0 = keep all).

### Environment Variables

SettingsSentry supports the following environment variables:

- `SETTINGS_SENTRY_CONFIG_PATH`: Path to the configuration folder.
- `SETTINGS_SENTRY_BACKUP_PATH`: Path to the backup folder.
- `SETTINGS_SENTRY_APP_NAME`: Optional name of the application to process.
- `SETTINGS_SENTRY_DRY_RUN`: Set to 'true' to perform a dry run without making any changes.

### Configuration Files

All configuration files are stored in the `configs` folder. Below is an example of a configuration file named `{name}.cfg`:

```ini

[application]
# Name of the application to backup
name = Brew

[backup_commands] # This directive is optional
# Command to execute for backing up installed packages
brew bundle dump --force --file=~/.Brewfile

[restore_commands]
# Command to execute for restoring packages from the backup
brew bundle install --file=~/.Brewfile

[configuration_files]
# List of configuration files to copy (supports files and folders)
.Brewfile

```

This configuration file specifies the application name, backup and restore commands, as well as the necessary configuration files.

#### Environment Variables in Configuration Files

You can use environment variables in your configuration files using the `${VAR_NAME}` syntax:

```ini
[application]
name = ${APP_NAME}

[configuration_files]
${CONFIG_DIR}/.config
~/Library/${APP_NAME}/settings.json
```

Environment variables will be expanded when the configuration is loaded, making it easy to reuse the same configuration across different environments or users.

### Configuration Wizard

The configuration wizard makes it easy to create new application configurations without having to manually edit configuration files. To start the wizard:

```bash
./SettingsSentry wizard
```

The wizard will guide you through:
- Specifying the application name
- Adding configuration file/directory paths
- Adding backup commands (optional)
- Adding restore commands (optional)

### Application Detection

SettingsSentry can automatically detect installed applications and help you create configurations for them:

```bash
./SettingsSentry detect
```

This will:
1. Scan your system for installed applications
2. Show a list of applications without existing configuration files
3. Help you create a configuration file for a selected application by suggesting potential configuration paths
4. For common applications, it will offer to use a pre-defined template

### Application Templates

SettingsSentry includes pre-defined templates for common macOS applications:

```bash
# List all available templates
./SettingsSentry template list

# Create a configuration file from a specific template
./SettingsSentry template vscode
```

Templates include common configuration paths and appropriate backup/restore commands for popular applications like VSCode, Chrome, Firefox, iTerm2, Homebrew, Git, and many more.

### Versioned Backups

SettingsSentry creates versioned backups using timestamp-based directories (format: YYYYMMDD-HHMMSS). This allows you to:

1. Keep multiple versions of your configuration backups
2. Restore from the latest version automatically
3. Limit the number of versions to keep using the `--versions` command-line argument

When restoring, SettingsSentry automatically uses the most recent backup version available.

### Dry Run Mode

The dry-run mode allows you to preview what would happen during backup or restore operations without making any actual changes to your system. This is useful for:

1. Testing new configurations
2. Verifying what files would be backed up or restored
3. Checking which commands would be executed

To use dry-run mode, add the `--dry-run` flag to your command:

```
./SettingsSentry backup --dry-run
```

## Tools

### Config Updater

SettingsSentry includes a tool to keep configuration definitions up-to-date with the [Mackup](https://github.com/lra/mackup) project. This tool can:

- Compare local configuration files with Mackup's repository
- Identify new applications that could be supported
- Find differences in existing configurations
- Automatically update or add new configurations

To use the config updater:

```bash
cd tools/config_updater
./update_configs.sh
```

For more information, see the [Config Updater README](tools/config_updater/README.md).

## Development

### Dependency Management

SettingsSentry uses Go modules for dependency management. The project dependencies are defined in the `go.mod` file and are vendored for reproducible builds.

To update dependencies to their latest versions:

```bash
go get -u ./...
go mod tidy
go mod vendor
```

To build using vendored dependencies:

```bash
go build -mod=vendor
```

## License

This project is licensed under the MIT License.

## Acknowledgments

Special thanks to Mackup team for the inspiration and configuration definitions.
