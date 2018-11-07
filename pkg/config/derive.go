package config

import (
	"fmt"
	"os"
	"strings"

	acsapi "github.com/openshift/openshift-azure/pkg/api"

	"github.com/ghodss/yaml"
)

type derived struct{}

var Derived derived

func isSmallVM(vmSize acsapi.VMSize) bool {
	// TODO: we should only be allowing StandardD2sV3 for test
	return vmSize == acsapi.StandardD2sV3
}

func (derived) SystemReserved(cs *acsapi.OpenShiftManagedCluster, role acsapi.AgentPoolProfileRole) (string, error) {
	if role == acsapi.AgentPoolProfileRoleMaster {
		return "", fmt.Errorf("systemreserved not defined for role %s", role)
	}

	for _, pool := range cs.Properties.AgentPoolProfiles {
		if pool.Role != role {
			continue
		}

		if isSmallVM(pool.VMSize) {
			return "cpu=200m,memory=512Mi", nil
		} else {
			return "cpu=500m,memory=512Mi", nil
		}
	}

	return "", fmt.Errorf("role %s not found", role)
}

func (derived) KubeReserved(cs *acsapi.OpenShiftManagedCluster, role acsapi.AgentPoolProfileRole) (string, error) {
	if role == acsapi.AgentPoolProfileRoleMaster {
		return "", fmt.Errorf("kubereserved not defined for role %s", role)
	}

	for _, pool := range cs.Properties.AgentPoolProfiles {
		if pool.Role != role {
			continue
		}

		if isSmallVM(pool.VMSize) {
			return "cpu=200m,memory=512Mi", nil
		} else {
			return "cpu=500m,memory=512Mi", nil
		}
	}

	return "", fmt.Errorf("role %s not found", role)
}

func (derived) PublicHostname(cs *acsapi.OpenShiftManagedCluster) string {
	if cs.Properties.PublicHostname != "" {
		return cs.Properties.PublicHostname
	}
	return cs.Properties.FQDN
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
