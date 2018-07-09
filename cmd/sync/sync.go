package main

import (
	"flag"
	"io/ioutil"

	"github.com/Azure/acs-engine/pkg/api/osa/vlabs"
	"github.com/ghodss/yaml"

	"github.com/jim-minter/azure-helm/pkg/addons"
	"github.com/jim-minter/azure-helm/pkg/config"
	"github.com/jim-minter/azure-helm/pkg/plugin"
)

var dryRun = flag.Bool("dry-run", false, "Print resources to be synced instead of mutating cluster state")

func sync() error {
	b, err := ioutil.ReadFile("_data/manifest.yaml")
	if err != nil {
		return err
	}
	var ext *vlabs.OpenShiftCluster
	err = yaml.Unmarshal(b, &ext)
	if err != nil {
		return err
	}
	err = ext.Validate()
	if err != nil {
		return err
	}
	cs := ext.AsContainerService()
	err = plugin.Enrich(cs)
	if err != nil {
		return err
	}

	b, err = ioutil.ReadFile("_data/config.yaml")
	if err != nil {
		return err
	}

	var c *config.Config
	err = yaml.Unmarshal(b, &c)
	if err != nil {
		return err
	}

	return addons.Main(cs, c, *dryRun)
}

func main() {
	flag.Parse()

	if err := sync(); err != nil {
		panic(err)
	}
}
