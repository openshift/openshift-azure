package config

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"
	"text/template"

	"github.com/ghodss/yaml"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/cloudprovider"
)

type derived struct{}

var Derived derived

func isSmallVM(vmSize api.VMSize) bool {
	// TODO: we should only be allowing StandardD2sV3 for test
	return vmSize == api.StandardD2sV3
}

func (derived) SystemReserved(cs *api.OpenShiftManagedCluster, role api.AgentPoolProfileRole) (string, error) {
	for _, app := range cs.Properties.AgentPoolProfiles {
		if app.Role != role {
			continue
		}

		if isSmallVM(app.VMSize) {
			if role == api.AgentPoolProfileRoleMaster {
				return "cpu=500m,memory=1Gi", nil
			} else {
				return "cpu=200m,memory=512Mi", nil
			}

		} else {
			if role == api.AgentPoolProfileRoleMaster {
				return "cpu=1000m,memory=1Gi", nil
			} else {
				return "cpu=500m,memory=512Mi", nil
			}
		}
	}

	return "", fmt.Errorf("role %s not found", role)
}

func (derived) KubeReserved(cs *api.OpenShiftManagedCluster, role api.AgentPoolProfileRole) (string, error) {
	if role == api.AgentPoolProfileRoleMaster {
		return "", fmt.Errorf("kubereserved not defined for role %s", role)
	}

	for _, app := range cs.Properties.AgentPoolProfiles {
		if app.Role != role {
			continue
		}

		if isSmallVM(app.VMSize) {
			return "cpu=200m,memory=512Mi", nil
		}
		return "cpu=500m,memory=512Mi", nil
	}

	return "", fmt.Errorf("role %s not found", role)
}

func (derived) PublicHostname(cs *api.OpenShiftManagedCluster) string {
	if cs.Properties.PublicHostname != "" {
		return cs.Properties.PublicHostname
	}
	return cs.Properties.FQDN
}

func (derived) RegistryURL(cs *api.OpenShiftManagedCluster) string {
	return strings.Split(cs.Config.Images.Format, "/")[0]
}

func (derived) OpenShiftVersionTag(cs *api.OpenShiftManagedCluster) (string, error) {
	parts := strings.Split(cs.Config.ImageVersion, ".")
	if len(parts) != 3 || len(parts[0]) < 2 {
		return "", fmt.Errorf("invalid imageVersion %q", cs.Config.ImageVersion)
	}

	return fmt.Sprintf("v%s.%s.%s", parts[0][:1], parts[0][1:], parts[1]), nil
}

func (derived) RouterLBCNamePrefix(cs *api.OpenShiftManagedCluster) string {
	return strings.Split(cs.Properties.RouterProfiles[0].FQDN, ".")[0]
}

func (derived) MasterLBCNamePrefix(cs *api.OpenShiftManagedCluster) string {
	return strings.Split(cs.Properties.FQDN, ".")[0]
}

func (derived) ClusterConfig(cs *api.OpenShiftManagedCluster) ([]byte, error) {
	return json.Marshal(cs)
}

func (derived) CloudProviderConf(cs *api.OpenShiftManagedCluster) ([]byte, error) {
	return yaml.Marshal(cloudprovider.Config{
		TenantID:          cs.Properties.AzProfile.TenantID,
		SubscriptionID:    cs.Properties.AzProfile.SubscriptionID,
		AadClientID:       cs.Properties.ServicePrincipalProfile.ClientID,
		AadClientSecret:   cs.Properties.ServicePrincipalProfile.Secret,
		ResourceGroup:     cs.Properties.AzProfile.ResourceGroup,
		LoadBalancerSku:   "standard",
		Location:          cs.Location,
		SecurityGroupName: "nsg-worker",
		VMType:            "vmss",
		SubnetName:        "default",
		VnetName:          "vnet",
	})
}

func (derived) AadGroupSyncConf(cs *api.OpenShiftManagedCluster) ([]byte, error) {
	provider := cs.Properties.AuthProfile.IdentityProviders[0].Provider.(*api.AADIdentityProvider)
	return yaml.Marshal(provider)
}

func (derived) ClusterMonitoringOperatorArgs(cs *api.OpenShiftManagedCluster) ([]interface{}, error) {
	return []interface{}{
		"-namespace=openshift-monitoring",
		"-configmap=cluster-monitoring-config",
		"-logtostderr=true",
		"-v=4",
		fmt.Sprintf("-tags=prometheus-operator=%s", cs.Properties.OpenShiftVersion),
		fmt.Sprintf("-tags=prometheus-config-reloader=%s", cs.Properties.OpenShiftVersion),
		fmt.Sprintf("-tags=config-reloader=%s", cs.Properties.OpenShiftVersion),
		fmt.Sprintf("-tags=prometheus=%s", cs.Properties.OpenShiftVersion),
		fmt.Sprintf("-tags=alertmanager=%s", cs.Properties.OpenShiftVersion),
		fmt.Sprintf("-tags=grafana=%s", cs.Properties.OpenShiftVersion),
		fmt.Sprintf("-tags=oauth-proxy=%s", cs.Properties.OpenShiftVersion),
		fmt.Sprintf("-tags=node-exporter=%s", cs.Properties.OpenShiftVersion),
		fmt.Sprintf("-tags=kube-state-metrics=%s", cs.Properties.OpenShiftVersion),
		fmt.Sprintf("-tags=kube-rbac-proxy=%s", cs.Properties.OpenShiftVersion),
	}, nil
}

func (derived) StatsdArgs(cs *api.OpenShiftManagedCluster) ([]interface{}, error) {
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

// MaxDataDisksPerVM is a stopgap until k8s 1.12.  It requires that a cluster
// has only one compute AgentPoolProfile and that no infra VM will require more
// mounted disks than the maximum number allowed by the compute agent pool.
// https://docs.microsoft.com/en-us/azure/virtual-machines/windows/sizes
func (derived) MaxDataDisksPerVM(cs *api.OpenShiftManagedCluster) (string, error) {
	var app *api.AgentPoolProfile
	for i := range cs.Properties.AgentPoolProfiles {
		if cs.Properties.AgentPoolProfiles[i].Role != api.AgentPoolProfileRoleCompute {
			continue
		}

		if app != nil {
			return "", fmt.Errorf("found multiple compute agentPoolProfiles")
		}

		app = &cs.Properties.AgentPoolProfiles[i]
	}

	if app == nil {
		return "", fmt.Errorf("couldn't find compute agentPoolProfile")
	}

	switch app.VMSize {
	case api.StandardD2sV3:
		return "4", nil
	case api.StandardD4sV3:
		return "8", nil
	case api.StandardD8sV3:
		return "16", nil
	case api.StandardD16sV3, api.StandardD32sV3, api.StandardD64sV3:
		return "32", nil

	case api.StandardDS4V2:
		return "32", nil
	case api.StandardDS5V2:
		return "64", nil

	// Compute optimized VMs
	case api.StandardF8sV2:
		return "16", nil
	case api.StandardF16sV2, api.StandardF32sV2, api.StandardF64sV2,
		api.StandardF72sV2:
		return "32", nil

	case api.StandardF8s:
		return "32", nil
	case api.StandardF16s:
		return "64", nil

	// Memory optimized VMs
	case api.StandardE4sV3:
		return "8", nil
	case api.StandardE8sV3:
		return "16", nil
	case api.StandardE16sV3, api.StandardE20sV3, api.StandardE32sV3,
		api.StandardE64sV3:
		return "32", nil

	case api.StandardGS2:
		return "16", nil
	case api.StandardGS3:
		return "32", nil
	case api.StandardGS4, api.StandardGS5:
		return "64", nil

	case api.StandardDS12V2:
		return "16", nil
	case api.StandardDS13V2:
		return "32", nil
	case api.StandardDS14V2, api.StandardDS15V2:
		return "64", nil

	// Storage optimized VMs
	case api.StandardL4s:
		return "16", nil
	case api.StandardL8s:
		return "32", nil
	case api.StandardL16s, api.StandardL32s:
		return "64", nil
	}

	return "", fmt.Errorf("unknown VMSize %q", app.VMSize)
}

func (derived) MDSDConfig(cs *api.OpenShiftManagedCluster) (string, error) {
	var tmpl = `<?xml version="1.0" encoding="utf-8"?>
    <MonitoringManagement version="1.0" namespace="{{ .Namespace | Escape }}" eventVersion="1" timestamp="2017-08-01T00:00:00.000Z">
        <Accounts>
            <Account moniker="{{ .Account | Escape }}" isDefault="true" autoKey="false" />
        </Accounts>
        <Management eventVolume="Large" defaultRetentionInDays="90" >
            <Identity tenantNameAlias="ResourceName" roleNameAlias="ResourceGroupName" roleInstanceNameAlias="SubscriptionId">
                <IdentityComponent name="Region">{{ .Region | Escape }}</IdentityComponent>
                <IdentityComponent name="SubscriptionId">{{ .SubscriptionId | Escape }}</IdentityComponent>
                <IdentityComponent name="ResourceGroupName">{{ .ResourceGroupName | Escape }}</IdentityComponent>
                <IdentityComponent name="ResourceName">{{ .ResourceName | Escape }}</IdentityComponent>
            </Identity>
            <AgentResourceUsage diskQuotaInMB="50000" />
        </Management>
        <Sources>
            <Source name="journald" dynamic_schema="true" />
            <Source name="audit" dynamic_schema="true" />
        </Sources>
        <Events>
            <MdsdEvents>
                <MdsdEventSource source="journald">
                    <RouteEvent eventName="CustomerSyslogEvents" storeType="CentralBond" priority="Normal"/>
                </MdsdEventSource>
                <MdsdEventSource source="audit">
                <RouteEvent eventName="CustomerAuditLogEvents" storeType="CentralBond" priority="Normal"/>
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
