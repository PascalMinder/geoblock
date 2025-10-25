package geoblock

import (
	"context"
	"fmt"
	"log"
	"os"
)

func CreateCustomLogTarget(ctx context.Context, logger *log.Logger, name string, logFilePath string) (*os.File, error) {
	logFile, err := initializeLogFile(logFilePath, logger)
	if err != nil {
		return nil, fmt.Errorf("error initializing log file: %v", err)
	}

	// Set up a goroutine to close the file when the context is done
	if logFile != nil {
		go func(logger *log.Logger) {
			<-ctx.Done() // Wait for context cancellation
			logger.SetOutput(os.Stdout)
			logFile.Close()
			logger.Printf("%s: Log file closed for middleware\n", name)
		}(logger)
	}

	logger.Printf("%s: Log file opened for middleware\n", name)
	return logFile, nil
}

func initializeLogFile(logFilePath string, logger *log.Logger) (*os.File, error) {
	if len(logFilePath) == 0 {
		return nil, nil
	}

	path, err := ValidatePersistencePath(logFilePath)
	if err != nil {
		return nil, err
	} else if len(path) == 0 {
		return nil, fmt.Errorf("folder is not writeable (%s)", path)
	}

	logFile, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, filePermissions)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file (%s): %v", path, err)
	}

	logger.SetOutput(logFile)
	return logFile, nil
}
