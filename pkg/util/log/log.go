package log

import (
	"strings"

	"github.com/sirupsen/logrus"
)

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
