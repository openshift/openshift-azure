package config

import (
	"fmt"
	"strings"

	"github.com/openshift/openshift-azure/pkg/api"
)

// Generator is an interface for sharing the cluster and plugin configs
type Generator interface {
	Generate(cs *api.OpenShiftManagedCluster) error
}

type simpleGenerator struct {
	pluginConfig api.PluginConfig
}

var _ Generator = &simpleGenerator{}

// NewSimpleGenerator creates a struct to hold both the cluster and plugin configs
func NewSimpleGenerator(pluginConfig *api.PluginConfig) Generator {
	return &simpleGenerator{
		pluginConfig: *pluginConfig,
	}
}

// openshiftVersion converts a VM image version (e.g. 310.14.20180101) to an
// openshift container image version (e.g. v3.10.14)
func openShiftVersion(imageVersion string) (string, error) {
	parts := strings.Split(imageVersion, ".")
	if len(parts) != 3 || len(parts[0]) < 2 {
		return "", fmt.Errorf("invalid imageVersion %q", imageVersion)
	}

	return fmt.Sprintf("v%s.%s.%s", parts[0][:1], parts[0][1:], parts[1]), nil
}

func (g *simpleGenerator) selectNodeImage(cs *api.OpenShiftManagedCluster) {
	c := cs.Config
	c.ImagePublisher = "redhat"
	c.ImageOffer = g.pluginConfig.TestConfig.ImageOffer
	if c.ImageOffer == "" {
		c.ImageOffer = "osa"
	}

	c.ImageVersion = g.pluginConfig.TestConfig.ImageVersion
	switch g.pluginConfig.TestConfig.DeployOS {
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

func (g *simpleGenerator) image(component, version string) string {
	image := strings.Replace(g.imageConfigFormat(), "${component}", component, -1)
	return strings.Replace(image, "${version}", version, -1)
}

func (g *simpleGenerator) selectContainerImagesOrigin(cs *api.OpenShiftManagedCluster) error {
	c := cs.Config
	v, err := openShiftVersion(c.ImageVersion)
	if err != nil {
		return err
	}

	switch cs.Properties.OpenShiftVersion {
	case "v3.10":
		c.Images.ControlPlane = g.image("control-plane", v)
		c.Images.Node = g.image("node", v)
		c.Images.ServiceCatalog = g.image("service-catalog", v)
		c.Images.Cli = g.image("cli", v)
		c.Images.TemplateServiceBroker = g.image("template-service-broker", v)
		c.Images.Registry = g.image("docker-registry", v)
		c.Images.Router = g.image("haproxy-router", v)
		c.Images.WebConsole = g.image("web-console", v)

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
		c.Images.Sync = "quay.io/openshift-on-azure/sync:v3.10"
		c.Images.LogBridge = "quay.io/openshift-on-azure/logbridge:latest"
	}

	return nil
}

func (g *simpleGenerator) selectContainerImagesOSA(cs *api.OpenShiftManagedCluster) error {
	c := cs.Config
	v, err := openShiftVersion(c.ImageVersion)
	if err != nil {
		return err
	}

	switch cs.Properties.OpenShiftVersion {
	//TODO: confirm minor version after release
	case "v3.10":
		c.Images.ControlPlane = g.image("control-plane", v)
		c.Images.Node = g.image("node", v)
		c.Images.Cli = g.image("cli", v)
		c.Images.ServiceCatalog = g.image("service-catalog", v)
		c.Images.AnsibleServiceBroker = g.image("ansible-service-broker", v)
		c.Images.TemplateServiceBroker = g.image("template-service-broker", v)
		c.Images.Registry = g.image("docker-registry", v)
		c.Images.Router = g.image("haproxy-router", v)
		c.Images.WebConsole = g.image("web-console", v)

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
		c.Images.Sync = "quay.io/openshift-on-azure/sync:v3.10"
		c.Images.LogBridge = "quay.io/openshift-on-azure/logbridge:latest"
	}

	return nil
}

func (g *simpleGenerator) selectContainerImages(cs *api.OpenShiftManagedCluster) error {
	var err error
	cs.Config.Images.Format = g.imageConfigFormat()
	switch g.pluginConfig.TestConfig.DeployOS {
	case "", "rhel7":
		err = g.selectContainerImagesOSA(cs)
	case "centos7":
		err = g.selectContainerImagesOrigin(cs)
	default:
		err = fmt.Errorf("unrecognised DeployOS value")
	}
	if err != nil {
		return err
	}

	if g.pluginConfig.SyncImage != "" {
		cs.Config.Images.Sync = g.pluginConfig.SyncImage
	}
	if g.pluginConfig.LogBridgeImage != "" {
		cs.Config.Images.LogBridge = g.pluginConfig.LogBridgeImage
	}

	return nil
}

func (g *simpleGenerator) imageConfigFormat() string {
	imageConfigFormat := g.pluginConfig.TestConfig.ORegURL
	if imageConfigFormat != "" {
		return imageConfigFormat
	}

	switch g.pluginConfig.TestConfig.DeployOS {
	case "", "rhel7":
		imageConfigFormat = "registry.access.redhat.com/openshift3/ose-${component}:${version}"
	case "centos7":
		imageConfigFormat = "docker.io/openshift/origin-${component}:${version}"
	}

	return imageConfigFormat
}
