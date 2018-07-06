package main

import (
	"context"
	"io/ioutil"

	"github.com/Azure/acs-engine/pkg/api/osa/vlabs"
	"github.com/ghodss/yaml"

	"github.com/jim-minter/azure-helm/pkg/plugin"
)

func healthCheck() error {
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

	configBytes, err := ioutil.ReadFile("_data/config.yaml")
	if err != nil {
		return err
	}

	p, err := plugin.NewPlugin(cs, nil, configBytes)
	if err != nil {
		return err
	}

	err = p.Validate()
	if err != nil {
		return err
	}

	return p.HealthCheck(context.Background())
}

func main() {
	if err := healthCheck(); err != nil {
		panic(err)
	}
}
