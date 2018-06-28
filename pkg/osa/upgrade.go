package osa

import (
	"io/ioutil"
	"os"

	"github.com/jim-minter/azure-helm/pkg/arm"
	"github.com/jim-minter/azure-helm/pkg/config"
	"github.com/jim-minter/azure-helm/pkg/helm"
)

func Upgrade() error {
	m, err := readManifest("_data/manifest.yaml")
	if err != nil {
		return err
	}

	c, err := readConfig()
	if err != nil {
		return err
	}

	c, err = config.Upgrade(m, c)
	if err != nil {
		return err
	}

	err = writeConfig(c)
	if err != nil {
		return err
	}

	err = os.MkdirAll("_data/_out", 0777)
	if err != nil {
		return err
	}

	values, err := helm.Generate(m, c)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("_data/_out/values.yaml", values, 0600)
	if err != nil {
		return err
	}

	azuredeploy, err := arm.Generate(m, c)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("_data/_out/azuredeploy.json", azuredeploy, 0600)
	if err != nil {
		return err
	}

	return writeHelpers(c)
}
