package config

import (
	"encoding/base64"
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

func (derived) ConsoleBaseAddress(cs *api.OpenShiftManagedCluster) string {
	return fmt.Sprintf("https://console.%s", cs.Properties.RouterProfiles[0].PublicSubdomain)
}

func (derived) RegistrySecret(cs *api.OpenShiftManagedCluster) string {
	format := "{\"auths\":{\"registry.redhat.io\":{\"auth\":\"%s\"}}}"
	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", cs.Config.RHUsername, cs.Config.RHPasswd)))
	return fmt.Sprintf(format, auth)
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
