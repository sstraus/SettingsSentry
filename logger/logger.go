package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

// Logger struct
type Logger struct {
	logFile *os.File
	logger  *log.Logger
	enabled bool
}

// NewLogger creates a new logger instance
func NewLogger(logFilePath string) (*Logger, error) {
	var file *os.File
	var err error

	if logFilePath != "" {
		// Create directory if it doesn't exist
		logDir := filepath.Dir(logFilePath)
		if _, err := os.Stat(logDir); os.IsNotExist(err) {
			err := os.MkdirAll(logDir, 0755)
			if err != nil {
				return nil, fmt.Errorf("error creating log directory: %w", err)
			}
		}

		// Try to create the file directly to see if there are any issues
		file, err = os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, fmt.Errorf("error creating log file: %w", err)
		}
	}

	var multiWriter io.Writer
	if file != nil {
		multiWriter = io.MultiWriter(os.Stdout, file)
	} else {
		multiWriter = os.Stdout
	}

	logger := log.New(multiWriter, "", log.LstdFlags)

	return &Logger{
		logFile: file,
		logger:  logger,
		enabled: logFilePath != "",
	}, nil
}

// Logf logs a formatted message
func (l *Logger) Logf(format string, v ...interface{}) {
	l.logger.Printf(format, v...)
}

// Log logs a message
func (l *Logger) Log(v ...interface{}) {
	l.logger.Println(v...)
}

// LogError logs a formatted error message and returns the error
func (l *Logger) LogErrorf(format string, v ...interface{}) error {
	logMessage := "Error: " + format
	l.Logf(logMessage, v...)
	return fmt.Errorf(logMessage, v...)
}

// Close closes the log file
func (l *Logger) Close() {
	if l.logFile != nil {
		l.logFile.Close()
	}
}
