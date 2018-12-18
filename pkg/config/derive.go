package config

import (
	"fmt"
	"strings"

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

func (derived) CloudProviderConf(cs *api.OpenShiftManagedCluster) ([]byte, error) {
	return yaml.Marshal(cloudprovider.Config{
		TenantID:            cs.Properties.AzProfile.TenantID,
		SubscriptionID:      cs.Properties.AzProfile.SubscriptionID,
		AadClientID:         cs.Properties.ServicePrincipalProfile.ClientID,
		AadClientSecret:     cs.Properties.ServicePrincipalProfile.Secret,
		ResourceGroup:       cs.Properties.AzProfile.ResourceGroup,
		Location:            cs.Location,
		LoadBalancerSku:     "standard",
		SecurityGroupName:   GetSecurityGroupName(cs, api.AgentPoolProfileRoleCompute),
		PrimaryScaleSetName: GetScalesetName(cs, api.AgentPoolProfileRoleCompute),
		VMType:              "vmss",
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
