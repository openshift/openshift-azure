package fakerp

import (
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/fakerp/shared"
	"github.com/openshift/openshift-azure/pkg/tls"
)

const (
	LoggingSecretsDirectory = "secrets/"
)

func GetPluginConfig() (*api.PluginConfig, error) {
	tc := api.TestConfig{
		RunningUnderTest:      os.Getenv("RUNNING_UNDER_TEST") == "true",
		ImageResourceGroup:    os.Getenv("IMAGE_RESOURCEGROUP"),
		ImageResourceName:     os.Getenv("IMAGE_RESOURCENAME"),
		DeployOS:              os.Getenv("DEPLOY_OS"),
		ImageOffer:            os.Getenv("IMAGE_OFFER"),
		ImageVersion:          os.Getenv("IMAGE_VERSION"),
		ORegURL:               os.Getenv("OREG_URL"),
		EtcdBackupImage:       os.Getenv("ETCDBACKUP_IMAGE"),
		AzureControllersImage: os.Getenv("AZURE_CONTROLLERS_IMAGE"),
	}

	// populate geneva artifacts
	artifactDir, err := shared.FindDirectory(LoggingSecretsDirectory)
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

	var syncImage, metricsBridge string
	if os.Getenv("SYNC_IMAGE") == "" {
		syncImage = "quay.io/openshift-on-azure/sync:latest"
	} else {
		syncImage = os.Getenv("SYNC_IMAGE")
	}
	if os.Getenv("METRICSBRIDGE_IMAGE") == "" {
		metricsBridge = "quay.io/openshift-on-azure/metricsbridge:latest"
	} else {
		metricsBridge = os.Getenv("METRICSBRIDGE_IMAGE")
	}

	genevaConfig := api.GenevaConfig{
		ImagePullSecret: pullSecret,

		LoggingCert:                logCert,
		LoggingKey:                 logKey,
		LoggingSector:              "US-Test",
		LoggingAccount:             "ccpopenshiftdiag",
		LoggingNamespace:           "CCPOpenShift",
		LoggingControlPlaneAccount: "RPOpenShiftAccount",
		LoggingImage:               "osarpint.azurecr.io/acs/mdsd:12051806",
		TDAgentImage:               "osarpint.azurecr.io/acs/td-agent:latest",

		MetricsCert:     metCert,
		MetricsKey:      metKey,
		MetricsBridge:   metricsBridge,
		StatsdImage:     "osarpint.azurecr.io/acs/mdm:git-a909a2e76",
		MetricsAccount:  "RPOpenShift",
		MetricsEndpoint: "https://az-int.metrics.nsatc.net/",
	}
	return &api.PluginConfig{
		SyncImage:       syncImage,
		AcceptLanguages: []string{"en-us"},
		TestConfig:      tc,
		GenevaConfig:    genevaConfig,
	}, nil
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
