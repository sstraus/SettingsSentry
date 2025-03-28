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

## Usage

./SettingsSentry `<action>` `<optional parameters>` [-config=`<path>`] [-backup=`<path>`] [-app=`<n>`] [-commands] [-dry-run] [-versions=`<n>`] [-logfile=`<path>`]

### Actions

- `backup`: Backup configuration files to the specified backup folder.
- `restore`: Restore the files to their original locations.
- `install`: Install the application as a CRON job that runs at every reboot.
    You can also provide a valid cron expression as a parameter to customize the schedule (0 9 \* \* \*). Use [cronhub](https://crontab.cronhub.io) to generate a valid one.
- `remove`: Remove the previously installed CRON job.
- `configsinit`: Extract embedded default configurations to a 'configs' directory located next to the executable. This allows for customization of the configurations and provides a way to view the default settings.

### Default Values

Configurations: configs
Backups: iCloud Drive/settingssentry_backups

#### Options

- `--config` `<path>`: Path to the configuration folder (default: `configs`).

- `--backup` `<path>`: Path to the backup folder (default: `iCloud Drive/settingssentry_backups`).

- `--app` `<n>`: Optional name of the application to process.

- `-commands`: Executes pre and post commands during backup or restore where available.

- `--dry-run`: Perform a dry run without making any changes.

- `--versions` `<n>`: Number of backup versions to keep (default: 1, 0 = keep all).

- `--logfile` `<path>`: Path to log file. If provided, logs will be written to this file in addition to console output.

### Environment Variables

SettingsSentry supports the following environment variables:

- `SETTINGSSENTRY_CONFIG`: Path to the configuration folder.
- `SETTINGSSENTRY_BACKUP`: Path to the backup folder.
- `SETTINGSSENTRY_APP`: Optional name of the application to process.
- `SETTINGSSENTRY_COMMANDS`: Set to 'true' to perform command execution during backup or restore.
- `SETTINGSSENTRY_DRY_RUN`: Set to 'true' to perform a dry run without making any changes.

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

```sh
./SettingsSentry backup --dry-run
```

## License

This project is licensed under the MIT License.
(C) 2025 Stefano Straus

## Acknowledgments

Special thanks to Mackup team for the inspiration and configuration definitions.
