package sync

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
	"text/template"

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

// TODO: remove once old router architecture no longer exists
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

func (derivedType) MDSDConfig(cs *api.OpenShiftManagedCluster) (string, error) {
	var tmpl = `<?xml version="1.0" encoding="utf-8"?>
    <MonitoringManagement version="1.0" namespace="{{ .Namespace | Escape }}" eventVersion="1" timestamp="2017-08-01T00:00:00.000Z">
        <Accounts>
            <Account moniker="{{ .Account | Escape }}" isDefault="true" autoKey="false"/>
        </Accounts>
        <Management eventVolume="Large" defaultRetentionInDays="90">
            <Identity tenantNameAlias="ResourceName" roleNameAlias="ResourceGroupName" roleInstanceNameAlias="SubscriptionId">
                <IdentityComponent name="Region">{{ .Region | Escape }}</IdentityComponent>
                <IdentityComponent name="SubscriptionId">{{ .SubscriptionId | Escape }}</IdentityComponent>
                <IdentityComponent name="ResourceGroupName">{{ .ResourceGroupName | Escape }}</IdentityComponent>
                <IdentityComponent name="ResourceName">{{ .ResourceName | Escape }}</IdentityComponent>
            </Identity>
            <AgentResourceUsage diskQuotaInMB="50000"/>
        </Management>
        <Sources>
            <Source name="journald" dynamic_schema="true"/>
            <Source name="audit" dynamic_schema="true"/>
        </Sources>
        <Events>
            <MdsdEvents>
                <MdsdEventSource source="journald">
                    <RouteEvent eventName="CustomerSyslogEvents" storeType="CentralBond" priority="High"/>
                </MdsdEventSource>
                <MdsdEventSource source="audit">
                    <RouteEvent eventName="CustomerAuditLogEvents" storeType="CentralBond" priority="High"/>
                </MdsdEventSource>
            </MdsdEvents>
        </Events>
	</MonitoringManagement>`

	t := template.Must(template.New("").Funcs(map[string]interface{}{
		"Escape": func(s string) (string, error) {
			var b bytes.Buffer
			err := xml.EscapeText(&b, []byte(s))
			return b.String(), err
		},
	}).Parse(tmpl))

	b := &bytes.Buffer{}

	err := t.Execute(b, map[string]string{
		"Namespace":         cs.Config.GenevaLoggingNamespace,
		"Account":           cs.Config.GenevaLoggingAccount,
		"Region":            cs.Location,
		"SubscriptionId":    cs.Properties.AzProfile.SubscriptionID,
		"ResourceName":      cs.Name,
		"ResourceGroupName": cs.Properties.AzProfile.ResourceGroup,
	})
	if err != nil {
		return "", err
	}

	return string(b.Bytes()), nil
}
