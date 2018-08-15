package log

import (
	"sync"

	"github.com/sirupsen/logrus"
)

// Wrapper wrapper logrus logger to enable global configuration
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

// Debug logs at debug level
func Debug(args ...interface{}) {
	logger.log.Debug(args...)
}

// Fatal logs at fatal level
func Fatal(args ...interface{}) {
	logger.log.Fatal(args...)
}

// WithFields adds a map of fields to the Entry.
func WithFields(fields logrus.Fields) *logrus.Entry {
	return logger.log.WithFields(fields)
}

// WithError adds a single error field to the Entry
func WithError(err error) *logrus.Entry {
	return logger.log.WithError(err)
}
