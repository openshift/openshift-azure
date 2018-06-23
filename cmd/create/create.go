package main

import (
	"io/ioutil"
	"os"

	"github.com/ghodss/yaml"
	"github.com/jim-minter/azure-helm/pkg/api"
	"github.com/jim-minter/azure-helm/pkg/arm"
	"github.com/jim-minter/azure-helm/pkg/config"
	"github.com/jim-minter/azure-helm/pkg/helm"
	"github.com/jim-minter/azure-helm/pkg/tls"
)

func create() error {
	b, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return err
	}

	var m *api.Manifest
	err = yaml.Unmarshal(b, &m)
	if err != nil {
		return err
	}

	c, err := config.Generate(m)
	if err != nil {
		return err
	}
	config, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("_out/config", config, 0600)
	if err != nil {
		return err
	}

	values, err := helm.Generate(m, c)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("_out/values.yaml", values, 0600)
	if err != nil {
		return err
	}

	azuredeploy, err := arm.Generate(m, c)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("_out/azuredeploy.json", azuredeploy, 0600)
	if err != nil {
		return err
	}

	sshprivatekey, err := tls.PrivateKeyAsBytes(c.SSHPrivateKey)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("_out/id_rsa", sshprivatekey, 0600)
	if err != nil {
		return err
	}

	adminkubeconfig, err := yaml.Marshal(c.AdminKubeconfig)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("_out/admin.kubeconfig", adminkubeconfig, 0600)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	if err := create(); err != nil {
		panic(err)
	}
}
