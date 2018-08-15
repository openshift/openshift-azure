package config

import (
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"os"
	"strings"

	acsapi "github.com/openshift/openshift-azure/pkg/api"
	"golang.org/x/crypto/bcrypt"
)

func selectNodeImage(cs *acsapi.ContainerService) {
	c := cs.Config
	c.ImagePublisher = "redhat"
	c.ImageOffer = "osa-preview"
	c.ImageVersion = "latest"

	switch os.Getenv("DEPLOY_OS") {
	case "":
		c.ImageSKU = "osa_" + strings.Replace(cs.Properties.OrchestratorProfile.OrchestratorVersion[1:], ".", "", -1)
	case "centos7":
		c.ImageSKU = "okd_" + strings.Replace(cs.Properties.OrchestratorProfile.OrchestratorVersion[1:], ".", "", -1)
	}

	c.ImageResourceGroup = os.Getenv("IMAGE_RESOURCEGROUP")
	c.ImageResourceName = os.Getenv("IMAGE_RESOURCENAME")
}

func image(imageConfigFormat, component, version string) string {
	image := strings.Replace(imageConfigFormat, "${component}", component, -1)
	return strings.Replace(image, "${version}", version, -1)
}

func selectContainerImagesOrigin(cs *acsapi.ContainerService) {
	c := cs.Config
	c.ImageConfigFormat = os.Getenv("OREG_URL")
	if c.ImageConfigFormat == "" {
		c.ImageConfigFormat = "docker.io/openshift/origin-${component}:${version}"
	}

	switch cs.Properties.OrchestratorProfile.OrchestratorVersion {
	case "v3.10":
		v := "v3.10.0"
		c.ControlPlaneImage = image(c.ImageConfigFormat, "control-plane", v)
		c.NodeImage = image(c.ImageConfigFormat, "node", v)
		c.ServiceCatalogImage = image(c.ImageConfigFormat, "service-catalog", v)
		c.TemplateServiceBrokerImage = image(c.ImageConfigFormat, "template-service-broker", v)
		c.RegistryImage = image(c.ImageConfigFormat, "docker-registry", v)
		c.RouterImage = image(c.ImageConfigFormat, "haproxy-router", v)
		c.WebConsoleImage = image(c.ImageConfigFormat, "web-console", v)

		c.MasterEtcdImage = "quay.io/coreos/etcd:v3.2.15"

		c.OAuthProxyImage = "docker.io/openshift/oauth-proxy:v1.0.0"
		c.PrometheusImage = "docker.io/openshift/prometheus:v2.2.1"
		c.PrometheusAlertBufferImage = "docker.io/openshift/prometheus-alert-buffer:v0.0.2"
		c.PrometheusAlertManagerImage = "docker.io/openshift/prometheus-alertmanager:v0.14.0"
		c.PrometheusNodeExporterImage = "docker.io/openshift/prometheus-node-exporter:v0.15.2"

		c.AnsibleServiceBrokerImage = "docker.io/ansibleplaybookbundle/origin-ansible-service-broker:latest"

		c.RegistryConsoleImage = "docker.io/cockpit/kubernetes:latest"

		c.AzureCLIImage = "docker.io/microsoft/azure-cli:latest"

		c.SyncImage = "quay.io/openshift-on-azure/sync:latest"
	}
}

func selectContainerImagesOSA(cs *acsapi.ContainerService) {
	c := cs.Config
	c.ImageConfigFormat = os.Getenv("OREG_URL")
	if c.ImageConfigFormat == "" {
		c.ImageConfigFormat = "registry.access.redhat.com/openshift3/ose-${component}:${version}"
	}

	switch cs.Properties.OrchestratorProfile.OrchestratorVersion {
	//TODO: confirm minor version after release
	case "v3.10":
		v := "v3.10"
		c.ControlPlaneImage = image(c.ImageConfigFormat, "control-plane", v)
		c.NodeImage = image(c.ImageConfigFormat, "node", v)
		c.ServiceCatalogImage = image(c.ImageConfigFormat, "service-catalog", v)
		c.AnsibleServiceBrokerImage = image(c.ImageConfigFormat, "ansible-service-broker", v)
		c.TemplateServiceBrokerImage = image(c.ImageConfigFormat, "template-service-broker", v)
		c.RegistryImage = image(c.ImageConfigFormat, "docker-registry", v)
		c.RouterImage = image(c.ImageConfigFormat, "haproxy-router", v)
		c.WebConsoleImage = image(c.ImageConfigFormat, "web-console", v)

		c.MasterEtcdImage = "registry.access.redhat.com/rhel7/etcd:3.2.22"

		c.OAuthProxyImage = "registry.access.redhat.com/openshift3/oauth-proxy:" + v
		c.PrometheusImage = "registry.access.redhat.com/openshift3/prometheus:" + v
		c.PrometheusAlertBufferImage = "registry.access.redhat.com/openshift3/prometheus-alert-buffer:" + v
		c.PrometheusAlertManagerImage = "registry.access.redhat.com/openshift3/prometheus-alertmanager:" + v
		c.PrometheusNodeExporterImage = "registry.access.redhat.com/openshift3/prometheus-node-exporter:" + v

		c.RegistryConsoleImage = "registry.access.redhat.com/openshift3/registry-console:" + v

		c.AzureCLIImage = "docker.io/microsoft/azure-cli:latest" //TODO: create mapping for OSA release to any other image we use

		c.SyncImage = "quay.io/openshift-on-azure/sync:latest"
	}
}

func selectContainerImages(cs *acsapi.ContainerService) {
	switch os.Getenv("DEPLOY_OS") {
	case "":
		selectContainerImagesOSA(cs)
	case "centos7":
		selectContainerImagesOrigin(cs)
	}
}

func makeHtPasswd(username, password string) ([]byte, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	return append([]byte(username+":"), b...), nil
}

func randomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := io.ReadAtLeast(rand.Reader, b, n); err != nil {
		return nil, err
	}
	return b, nil
}

func randomStorageAccountName() (string, error) {
	const letterBytes = "abcdefghijklmnopqrstuvwxyz0123456789"

	b := make([]byte, 24)
	for i := range b {
		o, err := rand.Int(rand.Reader, big.NewInt(int64(len(letterBytes))))
		if err != nil {
			return "", err
		}
		b[i] = letterBytes[o.Int64()]
	}

	return string(b), nil
}

func randomString(length int) (string, error) {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	b := make([]byte, length)
	for i := range b {
		o, err := rand.Int(rand.Reader, big.NewInt(int64(len(letterBytes))))
		if err != nil {
			return "", err
		}
		b[i] = letterBytes[o.Int64()]
	}

	return string(b), nil
}

func selectDNSNames(cs *acsapi.ContainerService) {

	// Prefix values used to set arm and router k8s service dns annotations
	cs.Config.RouterLBCNamePrefix = strings.Split(cs.Properties.OrchestratorProfile.OpenShiftConfig.RouterProfiles[0].FQDN, ".")[0]
	cs.Config.MasterLBCNamePrefix = strings.Split(cs.Properties.FQDN, ".")[0]

	// Set PublicHostname to FQDN values if not specified
	if cs.Properties.OrchestratorProfile.OpenShiftConfig.PublicHostname == "" {
		cs.Properties.OrchestratorProfile.OpenShiftConfig.PublicHostname = cs.Properties.FQDN
	}
	if cs.Properties.OrchestratorProfile.OpenShiftConfig.RouterProfiles[0].PublicSubdomain == "" {
		cs.Properties.OrchestratorProfile.OpenShiftConfig.RouterProfiles[0].PublicSubdomain = cs.Properties.OrchestratorProfile.OpenShiftConfig.RouterProfiles[0].FQDN
	}
}

func getMasterDNSNames(cs *acsapi.ContainerService) []string {
	dnsNames := []string{}
	for _, app := range cs.Properties.AgentPoolProfiles {
		switch app.Role {
		case acsapi.AgentPoolProfileRoleMaster:
			for i := 0; i < app.Count; i++ {
				dnsNames = append(dnsNames, fmt.Sprintf("master-%06d", i))
			}
		}
	}
	return dnsNames
}
