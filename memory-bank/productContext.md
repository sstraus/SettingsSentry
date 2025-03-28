# Product Context

This file provides a high-level overview of the project and the expected product that will be created. Initially it is based upon projectBrief.md (if provided) and all other available project-related information in the working directory. This file is intended to be updated as the project evolves, and should be used to inform all other modes of the project's goals and context.
YYYY-MM-DD HH:MM:SS - Log of updates made will be appended as footnotes to the end of this file.

*   **Main Entry Point (`main.go`):**  Serves as the application's entry point, handling command-line arguments, initializing global components (logger, file system interface, printer), and initiating the backup or restore process.
*   **Cron Job Management (`cron` package):**  Manages scheduled backup tasks using cron jobs, providing functionalities to add, remove, and check the installation status of cron jobs.
*   **Configuration Management (`pkg/config`):**
    *   Parses `.cfg` configuration files to load application-specific backup settings.
    *   Validates loaded configurations to ensure correctness.
    *   Expands environment variables within configuration values for dynamic settings.
    *   Includes utilities to determine standard configuration directories (XDG, iCloud).
*   **Backup and Restore Logic (`pkg/backup`):**
    *   Contains the core logic for backup and restore operations, including versioning and file manipulation.
    *   The `ProcessConfiguration` function orchestrates the entire backup/restore workflow.
    *   Utilizes `copyFile` and `copyDirectory` functions for file and directory copying.
    *   Implements backup versioning by creating timestamped directories for each backup.
*   **Command Execution (`pkg/command`):**
    *   Executes pre and post backup/restore commands as defined in the configuration files.
    *   Provides the `SafeExecute` function to handle potential errors during command execution gracefully.
*   **User Output and Printing (`pkg/printer`):**
    *   Offers formatted console output to communicate backup/restore status and messages to the user.
    *   Used for displaying progress, dry run simulations, and error notifications.
*   **Logging (`logger` package):**
    *   Manages application-wide logging, recording events, errors, and debug information to a log file and optionally to the console.
*   **File System Abstraction (`interfaces` package):**
    *   Defines the `FileSystem` interface, abstracting file system interactions.
    *   Provides `OsFileSystem` as a concrete implementation for standard operating system file system operations.
    *   This abstraction promotes modularity, testability, and potential future support for diverse file systems.
*   **Utility Functions (`pkg/util`):**
    *   Includes general utility functions, such as `GetEnvWithDefault` for retrieving environment variables with default values.
    *   `EmbeddedFallback` (partially implemented) is intended for handling embedded configurations as a fallback mechanism.
    *   `InitGlobals` function is used to initialize global variables, including logger, file system interface, and other dependencies.

## Project Goal

SettingsSentry is a command-line tool designed to simplify the backup and restoration of application configurations. It aims to provide a user-friendly and reliable way to manage configuration files, ensuring that users can easily preserve and restore their application settings.

## Key Features

*   **Configuration Backup and Restore:**  Core functionality to back up and restore application configurations.
*   **Versioned Backups:** Creates timestamped versioned backups, enabling users to revert to previous configurations.
*   **Backup Retention Policy:**  Allows users to specify the number of backup versions to keep, automatically cleaning up older versions.
*   **Pre/Post Backup/Restore Command Execution:** Supports executing custom commands before and after backup/restore operations for application-specific actions (e.g., stopping/starting services).
*   **Dry Run Mode:** Simulates backup and restore processes without modifying actual files, for testing and planning.
*   **Configuration File Driven:** Uses `.cfg` files to define backup configurations for different applications, making it easily extensible.
*   **Flexible File System Interface:** Employs the `interfaces.FileSystem` interface, providing abstraction for file system operations and enhancing testability and adaptability.
*   **Comprehensive Logging:** Utilizes the `logger` package for detailed logging of operations, errors, and debug information to both file and console.
*   **Cron Job Scheduling (Optional):**  Offers the capability to schedule backups using cron jobs via the `cron` package.
*   **Environment Variable Expansion:**  Supports environment variable expansion within configuration files, allowing for flexible path and setting management.
*   **Embedded Configuration Fallback (Potentially):**  Includes logic for a fallback to embedded default configurations if external configuration files are not found (though this feature appears to be partially implemented).

## Overall Architecture

*