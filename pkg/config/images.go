package config

import (
	"os"
	"strings"

	acsapi "github.com/openshift/openshift-azure/pkg/api"
)

func selectNodeImage(cs *acsapi.OpenShiftManagedCluster, deployOS string) {
	c := cs.Config
	c.ImagePublisher = "redhat"
	c.ImageOffer = os.Getenv("IMAGE_OFFER")
	if c.ImageOffer == "" {
		c.ImageOffer = "osa"
	}

	c.ImageVersion = os.Getenv("IMAGE_VERSION")
	switch deployOS {
	case "", "rhel7":
		c.ImageSKU = "osa_" + strings.Replace(cs.Properties.OpenShiftVersion[1:], ".", "", -1)
		if c.ImageVersion == "" {
			c.ImageVersion = "310.34.20180913"
		}
	case "centos7":
		c.ImageSKU = "origin_" + strings.Replace(cs.Properties.OpenShiftVersion[1:], ".", "", -1)
		if c.ImageVersion == "" {
			c.ImageVersion = "310.0.20180913"
		}
	}
}

func image(cs *acsapi.OpenShiftManagedCluster, component, version string) string {
	image := strings.Replace(Derived.ImageConfigFormat(cs), "${component}", component, -1)
	return strings.Replace(image, "${version}", version, -1)
}

func selectContainerImagesOrigin(cs *acsapi.OpenShiftManagedCluster) {
	c := cs.Config

	switch cs.Properties.OpenShiftVersion {
	case "v3.10":
		v := "v3.10.0" // TODO: perhaps we should calculate this from c.ImageVersion
		c.ControlPlaneImage = image(cs, "control-plane", v)
		c.NodeImage = image(cs, "node", v)
		c.ServiceCatalogImage = image(cs, "service-catalog", v)
		c.TemplateServiceBrokerImage = image(cs, "template-service-broker", v)
		c.RegistryImage = image(cs, "docker-registry", v)
		c.RouterImage = image(cs, "haproxy-router", v)
		c.WebConsoleImage = image(cs, "web-console", v)

		c.MasterEtcdImage = "quay.io/coreos/etcd:v3.2.15"
		c.EtcdOperatorImage = "quay.io/coreos/etcd-operator:v0.9.2"

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

	switch cs.Properties.OpenShiftVersion {
	//TODO: confirm minor version after release
	case "v3.10":
		v := "v3.10.14" // TODO: perhaps we should calculate this from c.ImageVersion
		c.ControlPlaneImage = image(cs, "control-plane", v)
		c.NodeImage = image(cs, "node", v)
		c.ServiceCatalogImage = image(cs, "service-catalog", v)
		c.AnsibleServiceBrokerImage = image(cs, "ansible-service-broker", v)
		c.TemplateServiceBrokerImage = image(cs, "template-service-broker", v)
		c.RegistryImage = image(cs, "docker-registry", v)
		c.RouterImage = image(cs, "haproxy-router", v)
		c.WebConsoleImage = image(cs, "web-console", v)

		c.MasterEtcdImage = "registry.access.redhat.com/rhel7/etcd:3.2.22"
		c.EtcdOperatorImage = "quay.io/coreos/etcd-operator:v0.9.2"

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
