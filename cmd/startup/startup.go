package main

import (
	"context"
	"encoding/json"
	"flag"
	"net/http"
	"os"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/startup"
	"github.com/openshift/openshift-azure/pkg/util/log"
	"github.com/openshift/openshift-azure/pkg/util/wait"
)

var (
	logLevel  = flag.String("loglevel", "Debug", "valid values are Debug, Info, Warning, Error")
	gitCommit = "unknown"
)

func run(ctx context.Context, log *logrus.Entry) error {
	log.Infof("reading config")
	var cs *api.OpenShiftManagedCluster
	err := wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		resp, err := http.Get(os.Getenv("SASURI"))
		if err != nil {
			log.Info(err)
			return false, nil
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			log.Infof("unexpected status code %d", resp.StatusCode)
			return false, nil
		}
		err = json.NewDecoder(resp.Body).Decode(&cs)
		if err != nil {
			log.Info(err)
			return false, nil
		}
		return true, nil
	}, ctx.Done())
	if err != nil {
		return err
	}

	s, err := startup.New(log, cs)
	if err != nil {
		return err
	}

	log.Info("writing startup files")
	return s.WriteFiles(ctx)
}

func main() {
	flag.Parse()
	logger := logrus.New()
	logger.Formatter = &logrus.TextFormatter{FullTimestamp: true}
	logger.SetLevel(log.SanitizeLogLevel(*logLevel))
	log := logrus.NewEntry(logger)
	log.Infof("startup pod starting, git commit %s", gitCommit)

	if err := run(context.Background(), log); err != nil {
		log.Fatal(err)
	}

	log.Info("all done successfully")
}
