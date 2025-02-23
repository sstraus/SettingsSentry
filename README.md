# SettingsSentry

__Securely archive and reinstate your macOS application configurations, simplifying system recovery processes.__

SettingsSentry is a lightweight and efficient tool written in Go to backup and restore macOS application configurations. It ensures your personalized settings are securely archived and easily reinstated, simplifying system recovery processes.

Inspired by [Mackup](https://github.com/lra/mackup), SettingsSentry was created to address compatibility issues with macOS Sonoma and later versions. Unlike Mackup, which no longer supports symlinked preference files and risks destroying user preferences, SettingsSentry provides a reliable solution.

## Features

- Backup configuration files to iCloud Drive or a specified folder.
- Restore configurations seamlessly to their original locations.
- Install a CRON job that runs at every system reboot.
- Remove the installed CRON job when no longer needed.

## Usage

./SettingsSentry `<action>` `<optional parameters>` [-config=`<path>`] [-backup=`<path>`] [-app=`<name>`] [-nocommands]

### Actions

- backup: Backup configuration files to the specified backup folder.
- restore: Restore the files to their original locations.
- install: Install the application as a CRON job that runs at every reboot.
    You can also provide a valid cron expression as a parameter to customize the schedule (0 9 * * *). Use [cronhub](https://crontab.cronhub.io) to generate a valid one.
- remove: Remove the previously installed CRON job.

### Default Values

Configurations: ./configs
Backups: iCloud Drive/SettingsSentry

#### Options

- `--config` `<path>`: Path to the configuration folder (default: `./configs`).

- `--backup` `<path>`: Path to the backup folder (default: `iCloud Drive/.settingssentry_backups`).

- `--app` `<name>`: Optional name of the application to process.

- `-nocommands`: Prevent command execution during backup or restore.

### Configuration Files

All configuration files are stored in the `./configs` folder. Below is an example of a configuration file named `{name}.cfg`:

```ini

[application]
\# Name of the application to backup
name = Brew

[backup_commands] # this directive is optional
\# Command to execute for backing up installed packages
brew bundle dump --force --file=~/.Brewfile

[restore_commands]
\# Command to execute for restoring packages from backup
brew bundle install --file=~/.Brewfile

[configuration_files]
\# List of configuration files to copy (supports files and folders)
.Brewfile

```

This configuration file specifies the application name, backup and restore commands, as well as the necessary configuration files.

## License

This project is licensed under the MIT License.

## Acknowledgments

Special thanks to Mackup for the inspiration and configuration definitions.
