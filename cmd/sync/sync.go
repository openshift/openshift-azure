package main

import (
	"io/ioutil"

	"github.com/ghodss/yaml"

	"github.com/jim-minter/azure-helm/pkg/addons"
	"github.com/jim-minter/azure-helm/pkg/api"
	"github.com/jim-minter/azure-helm/pkg/config"
)

func main() {
	b, err := ioutil.ReadFile("_data/manifest.yaml")
	if err != nil {
		panic(err)
	}

	var m *api.Manifest
	err = yaml.Unmarshal(b, &m)
	if err != nil {
		panic(err)
	}

	b, err = ioutil.ReadFile("_data/config.yaml")
	if err != nil {
		panic(err)
	}

	var c *config.Config
	err = yaml.Unmarshal(b, &c)
	if err != nil {
		panic(err)
	}

	if err := addons.Main(m, c); err != nil {
		panic(err)
	}
}
