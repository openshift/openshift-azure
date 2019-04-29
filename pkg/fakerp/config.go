package fakerp

import (
	"crypto/rsa"
	"crypto/x509"
	"io/ioutil"
	"os"

	"github.com/ghodss/yaml"

	"github.com/openshift/openshift-azure/pkg/api"
	pluginapi "github.com/openshift/openshift-azure/pkg/api/plugin"
	"github.com/openshift/openshift-azure/pkg/util/tls"
)

const (
	ContainerService                api.ContextKey = "ContainerService"
	ContextKeyGraphClientAuthorizer api.ContextKey = "GraphClientAuthorizer"
)

func GetTestConfig() api.TestConfig {
	return api.TestConfig{
		RunningUnderTest:   os.Getenv("RUNNING_UNDER_TEST") == "true",
		DebugHashFunctions: os.Getenv("DEBUG_HASH_FUNCTIONS") == "true",
		ImageResourceGroup: os.Getenv("IMAGE_RESOURCEGROUP"),
		ImageResourceName:  os.Getenv("IMAGE_RESOURCENAME"),
		ArtifactDir:        os.Getenv("ARTIFACTS"),
	}
}

func GetPluginTemplate() (*pluginapi.Config, error) {
	// read template file without secrets
	data, err := ioutil.ReadFile("pluginconfig/pluginconfig-311.yaml")
	if err != nil {
		return nil, err
	}
	var template *pluginapi.Config
	if err := yaml.Unmarshal(data, &template); err != nil {
		return nil, err
	}

	// enrich template with secrets
	logCert, err := readCert("secrets/logging-int.cert")
	if err != nil {
		return nil, err
	}
	logKey, err := readKey("secrets/logging-int.key")
	if err != nil {
		return nil, err
	}
	metCert, err := readCert("secrets/metrics-int.cert")
	if err != nil {
		return nil, err
	}
	metKey, err := readKey("secrets/metrics-int.key")
	if err != nil {
		return nil, err
	}
	pullSecret, err := ioutil.ReadFile("secrets/.dockerconfigjson")
	if err != nil {
		return nil, err
	}
	imagePullSecret, err := ioutil.ReadFile("secrets/system-docker-config.json")
	if err != nil {
		return nil, err
	}
	template.Certificates.GenevaLogging.Cert = logCert
	template.Certificates.GenevaLogging.Key = logKey
	template.Certificates.GenevaMetrics.Cert = metCert
	template.Certificates.GenevaMetrics.Key = metKey
	template.GenevaImagePullSecret = pullSecret
	template.ImagePullSecret = imagePullSecret

	return template, nil
}

func overridePluginTemplate(template *pluginapi.Config) {
	v := template.Versions[template.PluginVersion]

	if os.Getenv("AZURE_IMAGE") != "" {
		v.Images.Sync = os.Getenv("AZURE_IMAGE")
		v.Images.MetricsBridge = os.Getenv("AZURE_IMAGE")
		v.Images.EtcdBackup = os.Getenv("AZURE_IMAGE")
		v.Images.TLSProxy = os.Getenv("AZURE_IMAGE")
		v.Images.Canary = os.Getenv("AZURE_IMAGE")
		v.Images.AzureControllers = os.Getenv("AZURE_IMAGE")
		v.Images.Startup = os.Getenv("AZURE_IMAGE")
	}
	if os.Getenv("OREG_URL") != "" {
		v.Images.Format = os.Getenv("OREG_URL")
	}
	if os.Getenv("IMAGE_VERSION") != "" {
		v.ImageVersion = os.Getenv("IMAGE_VERSION")
	}
	if os.Getenv("IMAGE_OFFER") != "" {
		v.ImageOffer = os.Getenv("IMAGE_OFFER")
	}

	template.Versions[template.PluginVersion] = v
}

func readCert(path string) (*x509.Certificate, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return tls.ParseCert(b)
}

func readKey(path string) (*rsa.PrivateKey, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return tls.ParsePrivateKey(b)
}
