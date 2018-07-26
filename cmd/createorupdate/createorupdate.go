package main

import (
	"io/ioutil"
	"os"

	acsapi "github.com/Azure/acs-engine/pkg/api"
	"github.com/Azure/acs-engine/pkg/api/osa/vlabs"
	"github.com/ghodss/yaml"

	"github.com/openshift/openshift-azure/pkg/plugin"
)

func createOrUpdate() error {
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

	var oldCs *acsapi.ContainerService
	var configBytes []byte
	if _, err := os.Stat("_data/config.yaml"); err == nil {
		// pre-existing config means we're in the update path
		configBytes, err = ioutil.ReadFile("_data/config.yaml")
		if err != nil {
			return err
		}

		// fake this for now
		oldCs = cs
	}

	p, err := plugin.NewPlugin(cs, oldCs, configBytes)
	if err != nil {
		return err
	}

	err = p.Validate()
	if err != nil {
		return err
	}

	configBytes, err = p.GenerateConfig()
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("_data/config.yaml", configBytes, 0600)
	if err != nil {
		return err
	}

	err = os.MkdirAll("_data/_out", 0777)
	if err != nil {
		return err
	}

	azuredeploy, err := p.GenerateARM()
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("_data/_out/azuredeploy.json", azuredeploy, 0600)
	if err != nil {
		return err
	}

	// WriteHelpers is for development - not part of the external API
	return p.(*plugin.Plugin).WriteHelpers()
}

func main() {
	if err := createOrUpdate(); err != nil {
		panic(err)
	}
}
