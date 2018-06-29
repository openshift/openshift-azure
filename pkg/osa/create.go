package osa

import (
	"io/ioutil"
	"os"
)

func Create() error {
	osa, err := NewOSAByPath("_data/manifest.yaml", "")
	if err != nil {
		return err
	}

	if errs := osa.Validate(); errs != nil {
		return err
	}

	c, err := osa.GenerateConfig()
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

	values, err := osa.GenerateHelm()
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("_data/_out/values.yaml", values, 0600)
	if err != nil {
		return err
	}

	azuredeploy, err := osa.GenerateARM()
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("_data/_out/azuredeploy.json", azuredeploy, 0600)
	if err != nil {
		return err
	}

	return writeHelpers(c)
}
