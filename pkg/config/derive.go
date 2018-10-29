package config

import (
	"fmt"
	"strings"

	"github.com/ghodss/yaml"

	"github.com/openshift/openshift-azure/pkg/api"
)

type derived struct{}

var Derived derived

func (derived) SystemReserved(cs *api.OpenShiftManagedCluster, role api.AgentPoolProfileRole) string {
	for _, pool := range cs.Properties.AgentPoolProfiles {
		if pool.Role != role {
			continue
		}
		return api.DefaultVMSizeKubeArguments[pool.VMSize][role][api.SystemReserved]
	}
	return ""
}

func (derived) KubeReserved(cs *api.OpenShiftManagedCluster, role api.AgentPoolProfileRole) string {
	for _, pool := range cs.Properties.AgentPoolProfiles {
		if pool.Role != role {
			continue
		}
		return api.DefaultVMSizeKubeArguments[pool.VMSize][role][api.KubeReserved]
	}
	return ""
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

func (derived) ConsoleBaseAddress(cs *api.OpenShiftManagedCluster) string {
	return fmt.Sprintf("console.%s", cs.Properties.RouterProfiles[0].PublicSubdomain)
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

func (derived) CloudProviderConf(cs *api.OpenShiftManagedCluster) ([]byte, error) {
	return yaml.Marshal(map[string]string{
		"tenantId":            cs.Properties.AzProfile.TenantID,
		"subscriptionId":      cs.Properties.AzProfile.SubscriptionID,
		"aadClientId":         cs.Properties.ServicePrincipalProfile.ClientID,
		"aadClientSecret":     cs.Properties.ServicePrincipalProfile.Secret,
		"resourceGroup":       cs.Properties.AzProfile.ResourceGroup,
		"location":            cs.Location,
		"securityGroupName":   "nsg-compute",
		"primaryScaleSetName": "ss-compute",
		"vmType":              "vmss",
	})
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
