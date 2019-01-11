package fakerp

import (
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ghodss/yaml"

	"github.com/openshift/openshift-azure/pkg/api"
	pluginapi "github.com/openshift/openshift-azure/pkg/api/plugin/api"
	"github.com/openshift/openshift-azure/pkg/fakerp/shared"
	"github.com/openshift/openshift-azure/pkg/tls"
)

const (
	SecretsDirectory      = "secrets/"
	PluginConfigDirectory = "pluginconfig/"
	TemplatesDirectory    = "/test/templates/"
)

func GetPluginConfig() (*api.PluginConfig, error) {
	tc := api.TestConfig{
		RunningUnderTest:   os.Getenv("RUNNING_UNDER_TEST") == "true",
		ImageResourceGroup: os.Getenv("IMAGE_RESOURCEGROUP"),
		ImageResourceName:  os.Getenv("IMAGE_RESOURCENAME"),
	}

	return &api.PluginConfig{
		AcceptLanguages: []string{"en-us"},
		TestConfig:      tc,
	}, nil
}

func GetPluginTemplate() (*pluginapi.Config, error) {
	// read template file without secrets
	artifactDir, err := shared.FindDirectory(PluginConfigDirectory)
	if err != nil {
		return nil, err
	}
	data, err := readFile(filepath.Join(artifactDir, "pluginconfig-311.yaml"))
	if err != nil {
		return nil, err
	}
	var template *pluginapi.Config
	if err := yaml.Unmarshal(data, &template); err != nil {
		return nil, err
	}

	// enrich template with secrets
	artifactDir, err = shared.FindDirectory(SecretsDirectory)
	if err != nil {
		return nil, err
	}
	logCert, err := readCert(filepath.Join(artifactDir, "logging-int.cert"))
	if err != nil {
		return nil, err
	}
	logKey, err := readKey(filepath.Join(artifactDir, "logging-int.key"))
	if err != nil {
		return nil, err
	}
	metCert, err := readCert(filepath.Join(artifactDir, "metrics-int.cert"))
	if err != nil {
		return nil, err
	}
	metKey, err := readKey(filepath.Join(artifactDir, "metrics-int.key"))
	if err != nil {
		return nil, err
	}
	pullSecret, err := readFile(filepath.Join(artifactDir, ".dockerconfigjson"))
	if err != nil {
		return nil, err
	}
	imagePullSecret, err := readFile(filepath.Join(artifactDir, "system-docker-config.json"))
	if err != nil {
		return nil, err
	}
	template.Certificates.GenevaLogging.Cert = logCert
	template.Certificates.GenevaLogging.Key = logKey
	template.Certificates.GenevaMetrics.Cert = metCert
	template.Certificates.GenevaMetrics.Key = metKey
	template.Images.GenevaImagePullSecret = pullSecret
	template.Images.ImagePullSecret = imagePullSecret

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

func readCert(path string) (*x509.Certificate, error) {
	b, err := readFile(path)
	if err != nil {
		return nil, err
	}
	return tls.ParseCert(b)
}

func readKey(path string) (*rsa.PrivateKey, error) {
	b, err := readFile(path)
	if err != nil {
		return nil, err
	}
	return tls.ParsePrivateKey(b)
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
