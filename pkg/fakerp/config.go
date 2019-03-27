package fakerp

import (
	"crypto/rsa"
	"crypto/x509"
	"io/ioutil"
	"os"

	"github.com/ghodss/yaml"

	"github.com/openshift/openshift-azure/pkg/api"
	pluginapi "github.com/openshift/openshift-azure/pkg/api/plugin/api"
	"github.com/openshift/openshift-azure/pkg/util/tls"
)

func GetTestConfig() api.TestConfig {
	return api.TestConfig{
		RunningUnderTest:   os.Getenv("RUNNING_UNDER_TEST") == "true",
		ImageResourceGroup: os.Getenv("IMAGE_RESOURCEGROUP"),
		ImageResourceName:  os.Getenv("IMAGE_RESOURCENAME"),
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

func overridePluginTemplate(template *pluginapi.Config) error {
	v := template.Versions[template.PluginVersion]

	// read plugin template override
	data, err := ioutil.ReadFile("pluginconfig/override.yaml")
	if err != nil {
		return err
	}
	var override *pluginapi.Config
	if err := yaml.Unmarshal(data, &override); err != nil {
		return err
	}
	o := override.Versions[override.PluginVersion]

	if o.Images.Sync != "" {
		v.Images.Sync = o.Images.Sync
	}
	if o.Images.MetricsBridge != "" {
		v.Images.MetricsBridge = o.Images.MetricsBridge
	}
	if o.Images.EtcdBackup != "" {
		v.Images.EtcdBackup = o.Images.EtcdBackup
	}
	if o.Images.TLSProxy != "" {
		v.Images.TLSProxy = o.Images.TLSProxy
	}
	if o.Images.Canary != "" {
		v.Images.Canary = o.Images.Canary
	}
	if o.Images.AzureControllers != "" {
		v.Images.AzureControllers = o.Images.AzureControllers
	}
	if o.Images.Startup != "" {
		v.Images.Startup = o.Images.Startup
	}
	if o.Images.Format != "" {
		v.Images.Format = o.Images.Format
	}
	if o.ImageVersion != "" {
		v.ImageVersion = o.ImageVersion
	}
	if o.ImageOffer != "" {
		v.ImageOffer = o.ImageOffer
	}
	template.Versions[template.PluginVersion] = v

	if override.ComponentLogLevel.APIServer >= 0 {
		template.ComponentLogLevel.APIServer = override.ComponentLogLevel.APIServer
	}
	if override.ComponentLogLevel.ControllerManager >= 0 {
		template.ComponentLogLevel.ControllerManager = override.ComponentLogLevel.ControllerManager
	}
	if override.ComponentLogLevel.Node >= 0 {
		template.ComponentLogLevel.Node = override.ComponentLogLevel.Node
	}

	return nil
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
