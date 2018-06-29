package main

import (
	"io/ioutil"

	"github.com/jim-minter/azure-helm/pkg/plugin"
)

func healthCheck() error {
	manifestBytes, err := ioutil.ReadFile("_data/manifest.yaml")
	if err != nil {
		return err
	}
	configBytes, err := ioutil.ReadFile("_data/config.yaml")
	if err != nil {
		return err
	}

	p, err := plugin.NewPlugin(manifestBytes, nil, configBytes)
	if err != nil {
		return err
	}

	err = p.Validate()
	if err != nil {
		return err
	}

	return p.HealthCheck()
}

func main() {
	if err := healthCheck(); err != nil {
		panic(err)
	}
}
