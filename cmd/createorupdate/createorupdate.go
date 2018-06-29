package main

import (
	"io/ioutil"
	"os"

	"github.com/jim-minter/azure-helm/pkg/plugin"
)

func createOrUpdate() error {
	manifestBytes, err := ioutil.ReadFile("_data/manifest.yaml")
	if err != nil {
		return err
	}

	var oldManifestBytes, configBytes []byte
	if _, err := os.Stat("_data/config.yaml"); err == nil {
		// pre-existing config means we're in the update path
		configBytes, err = ioutil.ReadFile("_data/config.yaml")
		if err != nil {
			return err
		}

		// fake this for now
		oldManifestBytes = make([]byte, len(manifestBytes))
		copy(oldManifestBytes, manifestBytes)
	}

	p, err := plugin.NewPlugin(manifestBytes, oldManifestBytes, configBytes)
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

	values, err := p.GenerateHelm()
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("_data/_out/values.yaml", values, 0600)
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
	err = p.(*plugin.Plugin).WriteHelpers()
	if err != nil {
		return err
	}

	return p.HealthCheck()

}

func main() {
	if err := createOrUpdate(); err != nil {
		panic(err)
	}
}
