package config

import (
	"os"
	"strings"

	acsapi "github.com/openshift/openshift-azure/pkg/api"
)

func selectNodeImage(cs *acsapi.OpenShiftManagedCluster) {
	c := cs.Config
	c.ImagePublisher = "redhat"
	c.ImageOffer = "osa-preview"
	c.ImageVersion = "latest"

	switch os.Getenv("DEPLOY_OS") {
	case "", "rhel7":
		c.ImageSKU = "osa_" + strings.Replace(cs.Properties.OpenShiftVersion[1:], ".", "", -1)
	case "centos7":
		c.ImageSKU = "origin_" + strings.Replace(cs.Properties.OpenShiftVersion[1:], ".", "", -1)
	}

	c.ImageResourceGroup = os.Getenv("IMAGE_RESOURCEGROUP")
	c.ImageResourceName = os.Getenv("IMAGE_RESOURCENAME")
}

func image(imageConfigFormat, component, version string) string {
	image := strings.Replace(imageConfigFormat, "${component}", component, -1)
	return strings.Replace(image, "${version}", version, -1)
}

func selectContainerImagesOrigin(cs *acsapi.OpenShiftManagedCluster) {
	c := cs.Config
	c.ImageConfigFormat = os.Getenv("OREG_URL")
	if c.ImageConfigFormat == "" {
		c.ImageConfigFormat = "docker.io/openshift/origin-${component}:${version}"
	}

	switch cs.Properties.OpenShiftVersion {
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

		if c.SyncImage == "" {
			c.SyncImage = "quay.io/openshift-on-azure/sync:v3.10"
		}

		c.LogBridgeImage = "quay.io/openshift-on-azure/logbridge:latest"
	}
}

func selectContainerImagesOSA(cs *acsapi.OpenShiftManagedCluster) {
	c := cs.Config
	c.ImageConfigFormat = os.Getenv("OREG_URL")
	if c.ImageConfigFormat == "" {
		c.ImageConfigFormat = "registry.access.redhat.com/openshift3/ose-${component}:${version}"
	}

	switch cs.Properties.OpenShiftVersion {
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

		if c.SyncImage == "" {
			c.SyncImage = "quay.io/openshift-on-azure/sync:v3.10"
		}

		c.LogBridgeImage = "quay.io/openshift-on-azure/logbridge:latest"
	}
}

func selectContainerImages(cs *acsapi.OpenShiftManagedCluster) {
	switch os.Getenv("DEPLOY_OS") {
	case "", "rhel7":
		selectContainerImagesOSA(cs)
	case "centos7":
		selectContainerImagesOrigin(cs)
	}
}
