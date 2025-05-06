// Package logx provides a standard logger implementation for the gomcp project.
package logx

import (
	"log"
	"os"
	"sync"
)

type LoggingLevel string

const (
	LogLevelDebug LoggingLevel = "debug"
	LogLevelInfo  LoggingLevel = "info"
	LogLevelWarn  LoggingLevel = "warn"
	LogLevelError LoggingLevel = "error"
)

// DefaultLogger provides a basic logger implementation using the standard log package.
type DefaultLogger struct {
	logger *log.Logger
	level  LoggingLevel
	mu     sync.Mutex
}

// NewDefaultLogger creates a new logger writing to stderr with standard flags.
func NewDefaultLogger() *DefaultLogger {
	return &DefaultLogger{
		logger: log.New(os.Stderr, "[DOSync] ", log.LstdFlags|log.Ltime|log.Lmsgprefix),
	}
}

// NewLogger creates a new logger instance based on the configuration.
// Currently only supports "stdout".
func NewLogger(logType string) Logger { // Return the interface type
	// Basic implementation using standard log
	// TODO: Add support for file logging, structured logging (e.g., zerolog, zap)
	prefix := "[Log] " // Example prefix
	return &DefaultLogger{
		logger: log.New(os.Stdout, prefix, log.LstdFlags|log.Lshortfile),
		level:  LogLevelInfo, // Default level
	}
}

func (l *DefaultLogger) Debug(msg string, args ...interface{}) {
	l.logger.Printf("DEBUG: "+msg, args...)
}
func (l *DefaultLogger) Info(msg string, args ...interface{}) { l.logger.Printf("INFO: "+msg, args...) }
func (l *DefaultLogger) Warn(msg string, args ...interface{}) { l.logger.Printf("WARN: "+msg, args...) }
func (l *DefaultLogger) Error(msg string, args ...interface{}) {
	l.logger.Printf("ERROR: "+msg, args...)
}

// Ensure interface compliance
var _ Logger = (*DefaultLogger)(nil)

// Logger defines the interface for logging.
type Logger interface {
	Debug(format string, v ...interface{})
	Info(format string, v ...interface{})
	Warn(format string, v ...interface{})
	Error(format string, v ...interface{})
	SetLevel(level LoggingLevel)
}

// SetLevel updates the logging level for the DefaultLogger.
func (l *DefaultLogger) SetLevel(level LoggingLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	// TODO: Validate level? For now, assume valid levels are passed.
	l.level = level
	l.logger.Printf("[LogX] Log level set to: %s", l.level) // Use internal logger
}

func levelToSeverity(level LoggingLevel) int {
	return 0
}
