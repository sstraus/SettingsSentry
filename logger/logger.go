package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

type Logger struct {
	logFile   *os.File
	logger    *log.Logger
	cliLogger *log.Logger
	enabled   bool
}

func NewLogger(logFilePath string) (*Logger, error) {
	var file *os.File
	var err error

	if logFilePath != "" {
		logDir := filepath.Dir(logFilePath)
		if _, err := os.Stat(logDir); os.IsNotExist(err) {
			err := os.MkdirAll(logDir, 0755)
			if err != nil {
				return nil, fmt.Errorf("error creating log directory: %w", err)
			}
		}

		file, err = os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, fmt.Errorf("error creating log file: %w", err)
		}
	}

	var fileLogger *log.Logger
	if file != nil {
		fileLogger = log.New(file, "", log.LstdFlags)
	}

	cliLogger := log.New(os.Stdout, "", 0)

	return &Logger{
		logFile:   file,
		logger:    fileLogger,
		cliLogger: cliLogger,
		enabled:   logFilePath != "",
	}, nil
}

func (l *Logger) GetCliLoggerWriter() io.Writer {
	return l.cliLogger.Writer()
}

func (l *Logger) SetCliLoggerOutput(writer io.Writer) {
	l.cliLogger.SetOutput(writer)
}

func (l *Logger) Logf(format string, v ...interface{}) {
	l.cliLogger.Printf(format, v...)

	if l.enabled && l.logger != nil {
		l.logger.Printf(format, v...)
	}
}

func (l *Logger) Log(v ...interface{}) {
	l.cliLogger.Println(v...)

	if l.enabled && l.logger != nil {
		l.logger.Println(v...)
	}
}

func (l *Logger) LogErrorf(format string, v ...interface{}) error {
	logMessage := "Error: " + format

	// For CLI, print in red
	redErrorFormat := "\033[31mError: " + format + "\033[0m"
	l.cliLogger.Printf(redErrorFormat, v...)

	if l.enabled && l.logger != nil {
		l.logger.Printf(logMessage, v...)
	}

	return fmt.Errorf(logMessage, v...)
}

func (l *Logger) Close() {
	if l.logFile != nil {
		// Explicitly ignore close error, as logging it might be problematic
		_ = l.logFile.Close()
	}
}
