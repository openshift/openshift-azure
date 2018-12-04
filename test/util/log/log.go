package log

import (
	. "github.com/onsi/ginkgo"

	"github.com/sirupsen/logrus"
)

func GetTestLogger() *logrus.Entry {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	logrus.SetOutput(GinkgoWriter)
	return logrus.NewEntry(logrus.StandardLogger())
}
