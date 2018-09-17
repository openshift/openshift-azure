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
		v := "v3.10.0" // TODO: perhaps we should calculate this from ImageVersion
		c.Images.ControlPlane = image(cs, "control-plane", v)
		c.Images.Node = image(cs, "node", v)
		c.Images.ServiceCatalog = image(cs, "service-catalog", v)
		c.Images.TemplateServiceBroker = image(cs, "template-service-broker", v)
		c.Images.Registry = image(cs, "docker-registry", v)
		c.Images.Router = image(cs, "haproxy-router", v)
		c.Images.WebConsole = image(cs, "web-console", v)

		c.Images.MasterEtcd = "quay.io/coreos/etcd:v3.2.15"
		c.Images.EtcdOperator = "quay.io/coreos/etcd-operator:v0.9.2"
		c.Images.KubeStateMetrics = "quay.io/coreos/kube-state-metrics:v1.4.0"
		c.Images.AddonsResizer = "k8s.gcr.io/addon-resizer:1.7"

		c.Images.OAuthProxy = "docker.io/openshift/oauth-proxy:v1.0.0"
		c.Images.Prometheus = "docker.io/openshift/prometheus:v2.2.1"
		c.Images.PrometheusAlertBuffer = "docker.io/openshift/prometheus-alert-buffer:v0.0.2"
		c.Images.PrometheusAlertManager = "docker.io/openshift/prometheus-alertmanager:v0.14.0"
		c.Images.PrometheusNodeExporter = "docker.io/openshift/prometheus-node-exporter:v0.15.2"

		c.Images.AnsibleServiceBroker = "docker.io/ansibleplaybookbundle/origin-ansible-service-broker:latest"

		c.Images.RegistryConsole = "docker.io/cockpit/kubernetes:latest"

		if c.Images.Sync == "" {
			c.Images.Sync = "quay.io/openshift-on-azure/sync:v3.10"
		}

		c.Images.LogBridge = "quay.io/openshift-on-azure/logbridge:latest"
	}
}

func selectContainerImagesOSA(cs *acsapi.OpenShiftManagedCluster) {
	c := cs.Config

	switch cs.Properties.OpenShiftVersion {
	//TODO: confirm minor version after release
	case "v3.10":
		v := "v3.10.14" // TODO: perhaps we should calculate this from c.Images.ImageVersion
		c.Images.ControlPlane = image(cs, "control-plane", v)
		c.Images.Node = image(cs, "node", v)
		c.Images.ServiceCatalog = image(cs, "service-catalog", v)
		c.Images.AnsibleServiceBroker = image(cs, "ansible-service-broker", v)
		c.Images.TemplateServiceBroker = image(cs, "template-service-broker", v)
		c.Images.Registry = image(cs, "docker-registry", v)
		c.Images.Router = image(cs, "haproxy-router", v)
		c.Images.WebConsole = image(cs, "web-console", v)

		c.Images.MasterEtcd = "registry.access.redhat.com/rhel7/etcd:3.2.22"
		c.Images.EtcdOperator = "quay.io/coreos/etcd-operator:v0.9.2"
		c.Images.KubeStateMetrics = "quay.io/coreos/kube-state-metrics:v1.4.0"
		c.Images.AddonsResizer = "k8s.gcr.io/addon-resizer:1.7"

		c.Images.OAuthProxy = "registry.access.redhat.com/openshift3/oauth-proxy:" + v
		c.Images.Prometheus = "registry.access.redhat.com/openshift3/prometheus:" + v
		c.Images.PrometheusAlertBuffer = "registry.access.redhat.com/openshift3/prometheus-alert-buffer:" + v
		c.Images.PrometheusAlertManager = "registry.access.redhat.com/openshift3/prometheus-alertmanager:" + v
		c.Images.PrometheusNodeExporter = "registry.access.redhat.com/openshift3/prometheus-node-exporter:" + v

		c.Images.RegistryConsole = "registry.access.redhat.com/openshift3/registry-console:" + v

		if c.Images.Sync == "" {
			c.Images.Sync = "quay.io/openshift-on-azure/sync:v3.10"
		}

		c.Images.LogBridge = "quay.io/openshift-on-azure/logbridge:latest"
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
