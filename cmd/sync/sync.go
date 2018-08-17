package main

import (
	"flag"
	"io/ioutil"
	"time"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/util/errors"

	"github.com/openshift/openshift-azure/pkg/addons"
	acsapi "github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/log"
	"github.com/openshift/openshift-azure/pkg/validate"
)

var (
	dryRun   = flag.Bool("dry-run", false, "Print resources to be synced instead of mutating cluster state.")
	once     = flag.Bool("run-once", false, "If true, run only once then quit.")
	interval = flag.Duration("interval", 3*time.Minute, "How often the sync process going to be rerun.")
	logLevel = flag.String("loglevel", "Debug", "valid values are Debug, Info, Warning, Error")
)

func sync() error {
	logrus.Print("Sync process started")

	b, err := ioutil.ReadFile("_data/containerservice.yaml")
	if err != nil {
		return errors.Wrap(err, "cannot read _data/containerservice.yaml")
	}

	var cs *acsapi.OpenShiftManagedCluster
	if err := yaml.Unmarshal(b, &cs); err != nil {
		return errors.Wrap(err, "cannot unmarshal _data/containerservice.yaml")
	}

	if errs := validate.Validate(cs, nil, false); len(errs) > 0 {
		return errors.Wrap(kerrors.NewAggregate(errs), "cannot validate _data/manifest.yaml")
	}

	if err := addons.Main(cs, *dryRun); err != nil {
		return errors.Wrap(err, "cannot sync cluster config")
	}

	logrus.Print("Sync process complete")
	return nil
}

func main() {
	flag.Parse()
	logrus.SetLevel(log.SanitizeLogLevel(*logLevel))

	for {
		if err := sync(); err != nil {
			logrus.Printf("Error while syncing: %v", err)
		}
		if *once {
			return
		}
		<-time.After(*interval)
	}
}
