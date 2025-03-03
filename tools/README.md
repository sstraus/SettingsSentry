# SettingsSentry Tools

This directory contains various tools used for SettingsSentry development and maintenance.

## Config Updater

The Config Updater tool has been moved to a separate Git repository to isolate it from the main project.

You can find it at: `/Users/stefano.straus/Gits/SettingsSentryConfigUpdater`

To use the Config Updater:

1. Clone the SettingsSentryConfigUpdater repository at the same level as the SettingsSentry repository
2. Run the update_configs.sh script from the SettingsSentryConfigUpdater directory

```bash
cd /path/to/SettingsSentryConfigUpdater
./update_configs.sh --compare-only
```

The tool expects the SettingsSentry repository to be located at `../SettingsSentry` relative to the SettingsSentryConfigUpdater directory.
