package log

import (
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

// Wrapper wraps logrus logger to enable global configuration
type wrapper struct {
	log *logrus.Entry
}

var logger *wrapper
var once sync.Once

// New takes in logrus.Entry and configures global logger
func New(entry *logrus.Entry) {
	once.Do(func() {
		logger = &wrapper{
			log: entry,
		}
	})
}

// WithField adds a single field to the Entry
func WithField(key string, value interface{}) *logrus.Entry {
	return logger.log.WithField(key, value)
}

// Info logs at info level
func Info(args ...interface{}) {
	logger.log.Info(args...)
}

// Infof logs at info level
func Infof(format string, args ...interface{}) {
	logger.log.Infof(format, args...)
}

// Debug logs at debug level
func Debug(args ...interface{}) {
	logger.log.Debug(args...)
}

// Debugf logs at debug level
func Debugf(format string, args ...interface{}) {
	logger.log.Debugf(format, args...)
}

// Fatal logs at fatal level
func Fatal(args ...interface{}) {
	logger.log.Fatal(args...)
}

// Fatalf logs at fatal level
func Fatalf(format string, args ...interface{}) {
	logger.log.Fatalf(format, args...)
}

// Warn logs at warn level
func Warn(args ...interface{}) {
	logger.log.Warn(args...)
}

// Warnf logs at fatal level
func Warnf(format string, args ...interface{}) {
	logger.log.Warnf(format, args...)
}

// WithFields adds a map of fields to the Entry.
func WithFields(fields logrus.Fields) *logrus.Entry {
	return logger.log.WithFields(fields)
}

// WithError adds a single error field to the Entry
func WithError(err error) *logrus.Entry {
	return logger.log.WithError(err)
}

// SanitizeLogLevel checks and sanitizes logLevel input.
func SanitizeLogLevel(lvl string) logrus.Level {
	switch strings.ToLower(lvl) {
	case "debug":
		return logrus.DebugLevel
	case "info":
		return logrus.InfoLevel
	case "warning":
		return logrus.WarnLevel
	case "error":
		return logrus.ErrorLevel
	default:
		// silently default to info
		return logrus.InfoLevel
	}
}
