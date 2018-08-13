package main

import (
	"flag"
	"io/ioutil"
	"time"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/util/errors"

	"github.com/openshift/openshift-azure/pkg/addons"
	acsapi "github.com/openshift/openshift-azure/pkg/api"
	validation "github.com/openshift/openshift-azure/pkg/validate"
)

var (
	dryRun   = flag.Bool("dry-run", false, "Print resources to be synced instead of mutating cluster state.")
	once     = flag.Bool("run-once", false, "If true, run only once then quit.")
	interval = flag.Duration("interval", 3*time.Minute, "How often the sync process going to be rerun.")
	config   = flag.String("config", "", "Path to the config to use for syncing a cluster.")
)

func validate() error {
	if *config == "" {
		return errors.New("--config is required")
	}
	return nil
}

func sync() error {
	log.Print("Sync process started")

	b, err := ioutil.ReadFile(*config)
	if err != nil {
		return errors.Wrapf(err, "cannot read %s", *config)
	}

	var cs *acsapi.OpenShiftManagedCluster
	if err := yaml.Unmarshal(b, &cs); err != nil {
		return errors.Wrapf(err, "cannot unmarshal %s", *config)
	}

	if errs := validation.Validate(cs, nil, false); len(errs) > 0 {
		return errors.Wrap(kerrors.NewAggregate(errs), "cannot validate user request")
	}

	if err := addons.Main(cs, *dryRun); err != nil {
		return errors.Wrap(err, "cannot sync cluster config")
	}

	log.Print("Sync process complete")
	return nil
}

func main() {
	flag.Parse()
	log.SetLevel(log.DebugLevel)

	if err := validate(); err != nil {
		log.Fatal(err)
	}

	for {
		if err := sync(); err != nil {
			log.Printf("Error while syncing: %v", err)
		}
		if *once {
			return
		}
		<-time.After(*interval)
	}
}
