package geoblock

import (
	"context"
	"fmt"
	"log"
	"os"
)

// CreateCustomLogTarget configures the provided logger to write to a custom
// log file and automatically restores standard output when the given context
// is canceled.
//
// It returns the opened *os.File handle, which is also closed automatically
// when the context is done.
//
// If logFilePath is empty or invalid, the logger remains unchanged and no
// file is created.
//
// The logger output will be reset to stdout when the context is canceled.
func CreateCustomLogTarget(ctx context.Context, logger *log.Logger, name string, logFilePath string) (*os.File, error) {
	logFile, err := openLogFile(logFilePath, logger)
	if err != nil {
		return nil, fmt.Errorf("initialize log file: %w", err)
	}

	if logFile != nil {
		go func(logger *log.Logger) {
			<-ctx.Done()
			logger.SetOutput(os.Stdout)
			if cerr := logFile.Close(); cerr != nil {
				logger.Printf("%s: error closing log file: %v", name, cerr)
			}
			logger.Printf("%s: log file closed for middleware", name)
		}(logger)
	}

	logger.Printf("%s: log file opened for middleware", name)
	return logFile, nil
}

// openLogFile validates the provided path and opens the target log file.
//
// If the path is empty, it returns (nil, nil).
// If the folder is not writable or validation fails, an error is returned.
//
// The logger’s output is redirected to the newly opened file.
// The caller is responsible for closing the returned file when done.
func openLogFile(logFilePath string, logger *log.Logger) (*os.File, error) {
	if len(logFilePath) == 0 {
		return nil, nil
	}

	path, err := ValidatePersistencePath(logFilePath)
	if err != nil {
		return nil, fmt.Errorf("validate persistence path: %w", err)
	} else if len(path) == 0 {
		return nil, fmt.Errorf("folder is not writable: %s", logFilePath)
	}

	const filePermissions = 0o600
	logFile, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, filePermissions)
	if err != nil {
		return nil, fmt.Errorf("open log file %q: %w", path, err)
	}

	logger.SetOutput(logFile)
	return logFile, nil
}
