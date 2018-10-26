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

func (g *simpleGenerator) selectContainerImagesOSA(cs *api.OpenShiftManagedCluster) error {
	c := cs.Config
	v, err := openShiftVersion(c.ImageVersion)
	if err != nil {
		return err
	}

	switch cs.Properties.OpenShiftVersion {
	//TODO: confirm minor version after release
	case "v3.11":

		// Operators
		c.Images.ClusterMonitoringOperator = g.image("cluster-monitoring-operator", v)

		// Operators base images
		c.Images.PrometheusOperatorBase = "registry.redhat.io/openshift3/ose-prometheus-operator"
		c.Images.PrometheusConfigReloaderBase = "registry.redhat.io/openshift3/ose-prometheus-config-reloader"
		c.Images.ConfigReloaderBase = "registry.redhat.io/openshift3/ose-configmap-reloader"
		c.Images.PrometheusBase = "registry.redhat.io/openshift3/prometheus"
		c.Images.AlertManagerBase = "registry.redhat.io/openshift3/prometheus-alertmanager"
		c.Images.NodeExporterBase = "registry.redhat.io/openshift3/prometheus-node-exporter"
		c.Images.GrafanaBase = "registry.redhat.io/openshift3/grafana"
		c.Images.KubeStateMetricsBase = "registry.redhat.io/openshift3/ose-kube-state-metrics"
		c.Images.KubeRbacProxyBase = "registry.redhat.io/openshift3/ose-kube-rbac-proxy"
		c.Images.OAuthProxyBase = "registry.redhat.io/openshift3/oauth-proxy"

		// other images
		c.Images.ControlPlane = g.image("control-plane", v)
		c.Images.Node = g.image("node", v)
		c.Images.Router = g.image("haproxy-router", v)
		c.Images.MasterEtcd = "registry.redhat.io/rhel7/etcd:3.2.22"

		c.Images.WebConsole = g.image("web-console", v)
		c.Images.Console = g.image("console", v)
		c.Images.RegistryConsole = "registry.redhat.io/openshift3/registry-console:" + v
		c.Images.ServiceCatalog = g.image("service-catalog", v)
		c.Images.Recycler = g.image("recycler", v)
		c.Images.ServiceCatalog = g.image("service-catalog", v)
		c.Images.AnsibleServiceBroker = g.image("ansible-service-broker", v)
		c.Images.TemplateServiceBroker = g.image("template-service-broker", v)
		c.Images.Registry = g.image("docker-registry", v)

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
		imageConfigFormat = "registry.redhat.io/openshift3/ose-${component}:${version}"
	case "centos7":
		imageConfigFormat = "docker.io/openshift/origin-${component}:${version}"
	}

	return imageConfigFormat
}
