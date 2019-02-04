package main

import (
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/util/errors"

	"github.com/openshift/openshift-azure/pkg/addons"
	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/api/validate"
	"github.com/openshift/openshift-azure/pkg/util/log"
)

var (
	dryRun    = flag.Bool("dry-run", false, "Print resources to be synced instead of mutating cluster state.")
	once      = flag.Bool("run-once", false, "If true, run only once then quit.")
	interval  = flag.Duration("interval", 3*time.Minute, "How often the sync process going to be rerun.")
	logLevel  = flag.String("loglevel", "Debug", "valid values are Debug, Info, Warning, Error")
	gitCommit = "unknown"
)

type sync struct {
	log *logrus.Entry
}

// sync syncs the current state of the cluster with the
// desired state that is kept in a file in local storage.
func (s *sync) sync(ctx context.Context) error {
	s.log.Print("Sync process started")

	s.log.Print("reading config")
	b, err := ioutil.ReadFile("_data/_out/containerservice.json")
	if err != nil {
		return err
	}

	var cs *api.OpenShiftManagedCluster
	if err := json.Unmarshal(b, &cs); err != nil {
		return err
	}

	v := validate.NewAPIValidator(cs.Config.RunningUnderTest)
	if errs := v.Validate(cs, nil, false); len(errs) > 0 {
		return errors.Wrap(kerrors.NewAggregate(errs), "cannot validate _data/manifest.yaml")
	}

	if err := addons.Main(ctx, s.log, cs, *dryRun); err != nil {
		return errors.Wrap(err, "cannot sync cluster config")
	}

	s.log.Print("Sync process complete")
	return nil
}

func main() {
	flag.Parse()
	logger := logrus.New()
	logger.Formatter = &logrus.TextFormatter{FullTimestamp: true}
	logger.SetLevel(log.SanitizeLogLevel(*logLevel))
	log := logrus.NewEntry(logger)
	log.Printf("sync pod starting, git commit %s", gitCommit)

	s := new(sync)
	s.log = log
	ctx := context.Background()

	for {
		err := s.sync(ctx)
		if err != nil {
			log.Printf("Error while syncing: %v", err)
		}
		if *once {
			return
		}
		<-time.After(*interval)
	}
}
