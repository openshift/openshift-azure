package fakerp

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

func TestLogger(t *testing.T) {
	logger, hook := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	s := Server{
		log: logrus.NewEntry(logger),
	}
	handler := s.logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("User-Agent", "testlogger")

	handler.ServeHTTP(nil, req)

	expectedLogs := fmt.Sprintf("%s %s %s (%s)", req.RemoteAddr, req.Method, req.URL, req.UserAgent())

	if hook.LastEntry() == nil {
		t.Fatalf("no entry logged")
	}

	if hook.LastEntry().Message != expectedLogs {
		t.Fatalf("got entry %q, wanted %q", hook.LastEntry().Message, expectedLogs)
	}
}
