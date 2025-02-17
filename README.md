# SettingsSentry

__Securely archive and reinstate your macOS application configurations, simplifying system recovery processes.__

SettingsSentry is a lightweight and efficient tool written in Go to backup and restore macOS application configurations. It ensures your personalized settings are securely archived and easily reinstated, simplifying system recovery processes.

Inspired by [Mackup](https://github.com/lra/mackup), SettingsSentry was created to address compatibility issues with macOS Sonoma and later versions. Unlike Mackup, which no longer supports symlinked preference files and risks destroying user preferences, SettingsSentry provides a reliable solution.

## Features

- Backup configuration files to iCloud Drive or a specified folder.
- Restore configurations seamlessly to their original locations.
- Install as a CRON job that runs at every system reboot.
- Remove the installed CRON job when no longer needed.

## Usage

./SettingsSentry `<action>` [-config=`<path>`] [-backup=`<path>`] [-app=`<name>`]

### Actions

- backup: Backup configuration files to the specified backup folder.
- restore: Restore the files to their original locations.
- install: Install the application as a CRON job that runs at every reboot.
- remove: Remove the previously installed CRON job.

### Default Values

Configurations: ./configs
Backups: iCloud Drive/SettingsSentry

#### Options

-config=`<path>`: Path to the configuration folder (default: ./configs).
-backup=`<path>`: Path to the backup folder (default: iCloud Drive/SettingsSentry).
-app=`<name>`: Optional name of the application to process.

## License

This project is licensed under the MIT License.

## Acknowledgments

Special thanks to Mackup for the inspiration and configuration definitions.
