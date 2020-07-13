package sync

import (
	"fmt"
	"strings"

	"github.com/openshift/openshift-azure/pkg/api"
	derivedpkg "github.com/openshift/openshift-azure/pkg/util/derived"
)

type derivedType struct{}

var derived = &derivedType{}

func (derivedType) RegistryURL(cs *api.OpenShiftManagedCluster) string {
	return strings.Split(cs.Config.Images.Format, "/")[0]
}

func (derivedType) OpenShiftVersionTag(cs *api.OpenShiftManagedCluster) (string, error) {
	parts := strings.Split(cs.Config.ImageVersion, ".")
	if len(parts) != 3 || len(parts[0]) < 2 {
		return "", fmt.Errorf("invalid imageVersion %q", cs.Config.ImageVersion)
	}

	return fmt.Sprintf("v%s.%s.%s", parts[0][:1], parts[0][1:], parts[1]), nil
}

// OpenShiftClientVersion gets the version out of the following registry.access.redhat.com/openshift3/ose-console:v3.11.135
func (derivedType) OpenShiftClientVersion(cs *api.OpenShiftManagedCluster) (string, error) {
	verIndex := strings.LastIndex(cs.Config.Images.Console, ":v")
	if verIndex < 2 {
		return "", fmt.Errorf("could not get version from %s", cs.Config.Images.Console)
	}
	return cs.Config.Images.Console[verIndex+2:], nil
}

func (derivedType) RouterLBCNamePrefix(cs *api.OpenShiftManagedCluster) string {
	return strings.Split(cs.Properties.RouterProfiles[0].FQDN, ".")[0]
}

func (derivedType) AadGroupSyncConf(cs *api.OpenShiftManagedCluster) ([]byte, error) {
	return derivedpkg.AadGroupSyncConf(cs)
}

func (derivedType) ClusterMonitoringOperatorArgs(cs *api.OpenShiftManagedCluster) ([]interface{}, error) {
	return []interface{}{
		"-namespace=openshift-monitoring",
		"-configmap=cluster-monitoring-config",
		"-logtostderr=true",
		"-v=4",
		fmt.Sprintf("-tags=prometheus-operator=%s", strings.Split(cs.Config.Images.PrometheusOperator, ":")[1]),
		fmt.Sprintf("-tags=prometheus-config-reloader=%s", strings.Split(cs.Config.Images.PrometheusConfigReloader, ":")[1]),
		fmt.Sprintf("-tags=config-reloader=%s", strings.Split(cs.Config.Images.ConfigReloader, ":")[1]),
		fmt.Sprintf("-tags=prometheus=%s", strings.Split(cs.Config.Images.Prometheus, ":")[1]),
		fmt.Sprintf("-tags=alertmanager=%s", strings.Split(cs.Config.Images.AlertManager, ":")[1]),
		fmt.Sprintf("-tags=grafana=%s", strings.Split(cs.Config.Images.Grafana, ":")[1]),
		fmt.Sprintf("-tags=oauth-proxy=%s", strings.Split(cs.Config.Images.OAuthProxy, ":")[1]),
		fmt.Sprintf("-tags=node-exporter=%s", strings.Split(cs.Config.Images.NodeExporter, ":")[1]),
		fmt.Sprintf("-tags=kube-state-metrics=%s", strings.Split(cs.Config.Images.KubeStateMetrics, ":")[1]),
		fmt.Sprintf("-tags=kube-rbac-proxy=%s", strings.Split(cs.Config.Images.KubeRbacProxy, ":")[1]),
	}, nil
}

func (derivedType) StatsdArgs(cs *api.OpenShiftManagedCluster) ([]interface{}, error) {
	return []interface{}{
		// "-Dbg", // enable debugging
		"-StopEvent", "MDMEvent",
		"-FrontEndUrl", cs.Config.GenevaMetricsEndpoint,
		"-MonitoringAccount", cs.Config.GenevaMetricsAccount,
		"-CertFile", "/mdm/certs/cert.pem",
		"-Input", "statsd_local",
		"-PrivateKeyFile", "/mdm/certs/key.pem",
		"-ConfigOverrides", `{"internalMetricProductionLevel":3,"enableDimensionTrimming":false}`,
		"-SourceIdentity", cs.Location,
		"-SourceRole", "OSA",
		"-SourceRoleInstance", "OSA",
	}, nil
}
