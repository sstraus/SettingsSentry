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
	logFile   *os.File
	logger    *log.Logger
	cliLogger *log.Logger
	enabled   bool
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

	// Create a logger for file output with timestamps
	var fileLogger *log.Logger
	if file != nil {
		fileLogger = log.New(file, "", log.LstdFlags)
	}

	// Create a logger for CLI output without timestamps
	cliLogger := log.New(os.Stdout, "", 0)

	return &Logger{
		logFile:   file,
		logger:    fileLogger,
		cliLogger: cliLogger,
		enabled:   logFilePath != "",
	}, nil
}

// GetCliLoggerWriter returns the cliLogger's writer.
func (l *Logger) GetCliLoggerWriter() io.Writer {
	return l.cliLogger.Writer()
}

// SetCliLoggerOutput sets the output writer for the CLI logger.
func (l *Logger) SetCliLoggerOutput(writer io.Writer) {
	l.cliLogger.SetOutput(writer)
}

// Logf logs a formatted message
func (l *Logger) Logf(format string, v ...interface{}) {
	// Log to CLI without timestamp
	l.cliLogger.Printf(format, v...)

	// Log to file with timestamp if enabled
	if l.enabled && l.logger != nil {
		l.logger.Printf(format, v...)
	}
}

// Log logs a message
func (l *Logger) Log(v ...interface{}) {
	// Log to CLI without timestamp
	l.cliLogger.Println(v...)

	// Log to file with timestamp if enabled
	if l.enabled && l.logger != nil {
		l.logger.Println(v...)
	}
}

// LogError logs a formatted error message and returns the error
func (l *Logger) LogErrorf(format string, v ...interface{}) error {
	logMessage := "Error: " + format

	// For CLI, print in red
	redErrorFormat := "\033[31mError: " + format + "\033[0m"
	l.cliLogger.Printf(redErrorFormat, v...)

	// For file logging, use normal format
	if l.enabled && l.logger != nil {
		l.logger.Printf(logMessage, v...)
	}

	return fmt.Errorf(logMessage, v...)
}

// Close closes the log file
func (l *Logger) Close() {
	if l.logFile != nil {
		l.logFile.Close()
	}
}
