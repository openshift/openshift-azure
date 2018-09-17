package config

import (
	"os"
	"strings"

	acsapi "github.com/openshift/openshift-azure/pkg/api"

	"github.com/ghodss/yaml"
)

type derived struct{}

var Derived derived

func (derived) SystemReserved(cs *acsapi.OpenShiftManagedCluster, role acsapi.AgentPoolProfileRole) string {
	for _, pool := range cs.Properties.AgentPoolProfiles {
		if pool.Role != role {
			continue
		}
		return acsapi.DefaultVMSizeKubeArguments[pool.VMSize]["system-reserved"]
	}
	return ""
}

func (derived) KubeReserved(cs *acsapi.OpenShiftManagedCluster, role acsapi.AgentPoolProfileRole) string {
	for _, pool := range cs.Properties.AgentPoolProfiles {
		if pool.Role != role {
			continue
		}
		return acsapi.DefaultVMSizeKubeArguments[pool.VMSize]["kube-reserved"]
	}
	return ""
}

func (derived) RouterLBCNamePrefix(cs *acsapi.OpenShiftManagedCluster) string {
	return strings.Split(cs.Properties.RouterProfiles[0].FQDN, ".")[0]
}

func (derived) MasterLBCNamePrefix(cs *acsapi.OpenShiftManagedCluster) string {
	return strings.Split(cs.Properties.FQDN, ".")[0]
}

func (derived) CloudProviderConf(cs *acsapi.OpenShiftManagedCluster) ([]byte, error) {
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

func (derived) ImageConfigFormat(cs *acsapi.OpenShiftManagedCluster) string {
	imageConfigFormat := os.Getenv("OREG_URL")
	if imageConfigFormat != "" {
		return imageConfigFormat
	}

	switch os.Getenv("DEPLOY_OS") {
	case "", "rhel7":
		imageConfigFormat = "registry.access.redhat.com/openshift3/ose-${component}:${version}"
	case "centos7":
		imageConfigFormat = "docker.io/openshift/origin-${component}:${version}"
	}

	return imageConfigFormat
}

func (derived) RunningUnderTest() bool {
	return os.Getenv("RUNNING_UNDER_TEST") != ""
}

func (derived) ImageResourceGroup() string {
	return os.Getenv("IMAGE_RESOURCEGROUP")
}

func (derived) ImageResourceName() string {
	return os.Getenv("IMAGE_RESOURCENAME")
}
