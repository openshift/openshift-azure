package config

import (
	"os"
	"strings"

	acsapi "github.com/openshift/openshift-azure/pkg/api"
	"k8s.io/client-go/tools/clientcmd/api/v1"

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

func (derived) AdminKubeconfig(cs *acsapi.OpenShiftManagedCluster) (*v1.Config, error) {
	return makeKubeConfig(
		cs.Config.Certificates.Admin.Key,
		cs.Config.Certificates.Admin.Cert,
		cs.Config.Certificates.Ca.Cert,
		cs.Properties.FQDN,
		"system:admin",
		"default",
	)
}

func (derived) MasterKubeconfig(cs *acsapi.OpenShiftManagedCluster) (*v1.Config, error) {
	return makeKubeConfig(
		cs.Config.Certificates.OpenShiftMaster.Key,
		cs.Config.Certificates.OpenShiftMaster.Cert,
		cs.Config.Certificates.Ca.Cert,
		cs.Properties.FQDN,
		"system:openshift-master",
		"default",
	)
}

func (derived) NodeBootstrapKubeconfig(cs *acsapi.OpenShiftManagedCluster) (*v1.Config, error) {
	return makeKubeConfig(
		cs.Config.Certificates.NodeBootstrap.Key,
		cs.Config.Certificates.NodeBootstrap.Cert,
		cs.Config.Certificates.Ca.Cert,
		cs.Properties.FQDN,
		"system:serviceaccount:openshift-infra:node-bootstrapper",
		"openshift-infra",
	)
}

func (derived) AzureClusterReaderKubeconfig(cs *acsapi.OpenShiftManagedCluster) (*v1.Config, error) {
	return makeKubeConfig(
		cs.Config.Certificates.AzureClusterReader.Key,
		cs.Config.Certificates.AzureClusterReader.Cert,
		cs.Config.Certificates.Ca.Cert,
		cs.Properties.FQDN,
		"system:serviceaccount:openshift-azure:azure-cluster-reader",
		"openshift-azure",
	)
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
