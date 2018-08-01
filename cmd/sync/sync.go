package main

import (
	"flag"
	"io/ioutil"
	"log"
	"time"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	kerrors "k8s.io/apimachinery/pkg/util/errors"

	"github.com/openshift/openshift-azure/pkg/addons"
	acsapi "github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/api/v1"
	"github.com/openshift/openshift-azure/pkg/config"
	"github.com/openshift/openshift-azure/pkg/validate"
)

var (
	dryRun   = flag.Bool("dry-run", false, "Print resources to be synced instead of mutating cluster state.")
	once     = flag.Bool("run-once", false, "If true, run only once then quit.")
	interval = flag.Duration("interval", 3*time.Minute, "How often the sync process going to be rerun.")
)

func sync() error {
	log.Print("Sync process started!")

	b, err := ioutil.ReadFile("_data/manifest.yaml")
	if err != nil {
		return errors.Wrap(err, "cannot read _data/manifest.yaml")
	}
	var ext *v1.OpenShiftCluster
	if err := yaml.Unmarshal(b, &ext); err != nil {
		return errors.Wrap(err, "cannot unmarshal _data/manifest.yaml")
	}
	if errs := validate.OpenShiftCluster(ext); len(errs) > 0 {
		return errors.Wrap(kerrors.NewAggregate(errs), "cannot validate _data/manifest.yaml")
	}
	cs := acsapi.ConvertVLabsOpenShiftClusterToContainerService(ext)

	b, err = ioutil.ReadFile("_data/config.yaml")
	if err != nil {
		return errors.Wrap(err, "cannot read _data/config.yaml")
	}

	var c *config.Config
	if err = yaml.Unmarshal(b, &c); err != nil {
		return errors.Wrap(err, "cannot unmarshal _data/config.yaml")
	}

	if err := addons.Main(cs, c, *dryRun); err != nil {
		return errors.Wrap(err, "cannot sync cluster config")
	}

	log.Print("Sync process complete!")
	return nil
}

func main() {
	flag.Parse()

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
