package log

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

var (
	_, thisfile, _, _ = runtime.Caller(0)
	repopath          = strings.Replace(thisfile, "pkg/util/log/log.go", "", -1)
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

// RelativeFilePathPrettier changes absolute paths with relative paths
func RelativeFilePathPrettier(f *runtime.Frame) (string, string) {
	filename := strings.Replace(f.File, repopath, "", -1)
	funcname := strings.Replace(f.Function, "github.com/openshift/openshift-azure/", "", -1)
	return fmt.Sprintf("%s()", funcname), fmt.Sprintf("%s:%d", filename, f.Line)
}
