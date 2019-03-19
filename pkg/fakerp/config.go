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
	if os.Getenv("TLSPROXY_IMAGE") != "" {
		template.Images.TLSProxy = os.Getenv("TLSPROXY_IMAGE")
	}
	if os.Getenv("AZURE_CONTROLLERS_IMAGE") != "" {
		template.Images.AzureControllers = os.Getenv("AZURE_CONTROLLERS_IMAGE")
	}
	if os.Getenv("STARTUP_IMAGE") != "" {
		template.Images.Startup = os.Getenv("STARTUP_IMAGE")
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
