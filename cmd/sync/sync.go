package main

import (
	"io/ioutil"

	"github.com/ghodss/yaml"
	"github.com/jim-minter/azure-helm/pkg/addons"
	"github.com/jim-minter/azure-helm/pkg/config"
)

func main() {
	b, err := ioutil.ReadFile("_out/config")
	if err != nil {
		panic(err)
	}

	var c *config.Config
	err = yaml.Unmarshal(b, &c)
	if err != nil {
		panic(err)
	}

	if err := addons.Main(c); err != nil {
		panic(err)
	}
}
