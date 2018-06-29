package osa

import (
	"io/ioutil"

	"github.com/ghodss/yaml"

	"github.com/jim-minter/azure-helm/pkg/api"
	"github.com/jim-minter/azure-helm/pkg/config"
	"github.com/jim-minter/azure-helm/pkg/tls"
)

func readManifest(path string) (*api.Manifest, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var m *api.Manifest
	err = yaml.Unmarshal(b, &m)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func readConfig() (*config.Config, error) {
	b, err := ioutil.ReadFile("_data/oonfig.yaml")
	if err != nil {
		return nil, err
	}

	var c *config.Config
	err = yaml.Unmarshal(b, &c)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func writeConfig(b []byte) error {
	return ioutil.WriteFile("_data/config.yaml", b, 0600)
}

func writeHelpers(b []byte) error {
	var c *config.Config
	err := yaml.Unmarshal(b, &c)
	if err != nil {
		return err
	}

	out, err := tls.PrivateKeyAsBytes(c.SSHPrivateKey)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("_data/_out/id_rsa", out, 0600)
	if err != nil {
		return err
	}

	out, err = yaml.Marshal(c.AdminKubeconfig)
	if err != nil {
		return err
	}
	return ioutil.WriteFile("_data/_out/admin.kubeconfig", out, 0600)
}
