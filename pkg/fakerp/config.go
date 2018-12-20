package fakerp

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ghodss/yaml"

	"github.com/openshift/openshift-azure/pkg/api"
	pluginapi "github.com/openshift/openshift-azure/pkg/api/plugin/api"
	"github.com/openshift/openshift-azure/pkg/fakerp/shared"
)

const (
	SecretsDirectory   = "secrets/"
	TemplatesDirectory = "/test/templates/"
)

func GetPluginConfig() (*api.PluginConfig, error) {
	tc := api.TestConfig{
		RunningUnderTest:   os.Getenv("RUNNING_UNDER_TEST") == "true",
		ImageResourceGroup: os.Getenv("IMAGE_RESOURCEGROUP"),
		ImageResourceName:  os.Getenv("IMAGE_RESOURCENAME"),
		DeployOS:           os.Getenv("DEPLOY_OS"),
	}

	return &api.PluginConfig{
		AcceptLanguages: []string{"en-us"},
		TestConfig:      tc,
	}, nil
}

func GetPluginTemplate() (*pluginapi.Config, error) {
	artifactDir, err := shared.FindDirectory(TemplatesDirectory)
	if err != nil {
		return nil, err
	}
	data, err := readFile(filepath.Join(artifactDir, "template.yaml"))
	if err != nil {
		return nil, err
	}
	var template *pluginapi.Config
	if err := yaml.Unmarshal(data, &template); err != nil {
		return nil, err
	}

	return template, nil
}

func overridePluginTemplate(template *pluginapi.Config) {
	if os.Getenv("SYNC_IMAGE") != "" {
		template.Images.Sync = os.Getenv("SYNC_IMAGE")
	}
	if os.Getenv("METRICSBRIDGE_IMAGE") != "" {
		template.Images.MetricsBridge = os.Getenv("METRICSBRIDGE_IMAGE")
	}
	if os.Getenv("ETCDBACKUP_IMAGE") != "" {
		template.Images.EtcdBackup = os.Getenv("ETCDBACKUP_IMAGE")
	}
	if os.Getenv("AZURE_CONTROLLERS_IMAGE") != "" {
		template.Images.AzureControllers = os.Getenv("AZURE_CONTROLLERS_IMAGE")
	}
	if os.Getenv("OREG_URL") != "" {
		template.Images.Format = os.Getenv("OREG_URL")
	}
	if os.Getenv("IMAGE_VERSION") != "" {
		template.ImageVersion = os.Getenv("IMAGE_VERSION")
	}
	if os.Getenv("IMAGE_OFFER") != "" {
		template.ImageOffer = os.Getenv("IMAGE_OFFER")
	}
}

func readFile(path string) ([]byte, error) {
	if fileExist(path) {
		return ioutil.ReadFile(path)
	}
	return []byte{}, fmt.Errorf("file %s does not exist", path)
}

func fileExist(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
