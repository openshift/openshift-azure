package log

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestInfo(t *testing.T) {
	os.Setenv("RESOURCEGROUP", "test-rg")
	// dummy writer
	buf := new(bytes.Buffer)
	// configure logger
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	logger.SetOutput(buf)
	entry := logrus.NewEntry(logger)
	entry = entry.WithFields(logrus.Fields{"resourceGroup": os.Getenv("RESOURCEGROUP")})
	New(entry)
	Info("info message")
	if !strings.Contains(buf.String(), "level=info msg=\"info message\" resourceGroup=test-rg") {
		t.Fatalf("test %s failed. Message [%v] does not contain expected output", "info", buf.String())
	}
}
