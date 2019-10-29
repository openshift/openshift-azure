package startup

import (
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"path"
	"strings"

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/openshift/openshift-azure/pkg/api"
	derivedpkg "github.com/openshift/openshift-azure/pkg/util/derived"
	"github.com/openshift/openshift-azure/pkg/util/tls"
)

type derivedType struct {
	root string
}

var _ = &derivedType{}

func isSmallVM(vmSize api.VMSize) bool {
	// TODO: we should only be allowing StandardD2sV3 for test
	return vmSize == api.StandardD2sV3
}

func (d *derivedType) SystemReserved(cs *api.OpenShiftManagedCluster, role api.AgentPoolProfileRole) (string, error) {
	for _, app := range cs.Properties.AgentPoolProfiles {
		if app.Role != role {
			continue
		}

		if isSmallVM(app.VMSize) {
			if role == api.AgentPoolProfileRoleMaster {
				return "cpu=500m,memory=1Gi", nil
			}

			return "cpu=200m,memory=512Mi", nil
		}

		if role == api.AgentPoolProfileRoleMaster {
			return "cpu=1000m,memory=1Gi", nil
		}

		return "cpu=500m,memory=512Mi", nil
	}

	return "", fmt.Errorf("role %s not found", role)
}

func (d *derivedType) KubeReserved(cs *api.OpenShiftManagedCluster, role api.AgentPoolProfileRole) (string, error) {
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

func (d *derivedType) MasterCloudProviderConf(cs *api.OpenShiftManagedCluster) ([]byte, error) {
	return derivedpkg.MasterCloudProviderConf(cs, false)
}

func (d *derivedType) WorkerCloudProviderConf(cs *api.OpenShiftManagedCluster) ([]byte, error) {
	return derivedpkg.WorkerCloudProviderConf(cs, false)
}

func getServerFromDNSConf(content string) ([]string, error) {
	//  cat /etc/dnsmasq.d/origin-upstream-dns.conf
	//  server=168.63.129.16
	servers := []string{}
	for _, line := range strings.Split(content, "\n") {
		if !strings.HasPrefix(line, "server=") {
			continue
		}
		servers = append(servers, strings.Split(line, "=")[1])
	}
	if len(servers) == 0 {
		return nil, fmt.Errorf("no servers found in origin-upstream-dns.conf")
	}
	return servers, nil
}

func (d *derivedType) NameServers() ([]string, error) {
	dnsConfFile := path.Join(d.root, "/etc/dnsmasq.d/origin-upstream-dns.conf")
	b, err := ioutil.ReadFile(dnsConfFile)
	if err != nil {
		return nil, err
	}
	return getServerFromDNSConf(string(b))
}

func (d *derivedType) CustomerResourceGroup(ID string) (string, error) {
	res, err := azure.ParseResourceID(ID)
	return res.ResourceGroup, err
}

// MaxDataDisksPerVM is a stopgap until k8s 1.12.  It requires that a cluster
// has only one compute AgentPoolProfile and that no infra VM will require more
// mounted disks than the maximum number allowed by the compute agent pool.
// https://docs.microsoft.com/en-us/azure/virtual-machines/windows/sizes
func (d *derivedType) MaxDataDisksPerVM(cs *api.OpenShiftManagedCluster) (string, error) {
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
	// General purpose VMs
	case api.StandardD2sV3:
		return "4", nil
	case api.StandardD4sV3:
		return "8", nil
	case api.StandardD8sV3:
		return "16", nil
	case api.StandardD16sV3, api.StandardD32sV3:
		return "32", nil

	// Memory optimized VMs
	case api.StandardE4sV3:
		return "8", nil
	case api.StandardE8sV3:
		return "16", nil
	case api.StandardE16sV3, api.StandardE32sV3:
		return "32", nil

	// Compute optimized VMs
	case api.StandardF8sV2:
		return "16", nil
	case api.StandardF16sV2, api.StandardF32sV2:
		return "32", nil
	}

	return "", fmt.Errorf("unknown VMSize %q", app.VMSize)
}

// CaBundle created ca-bundle which includes
// CA and any external certificates we trust
func (d *derivedType) CaBundle(cs *api.OpenShiftManagedCluster) ([]*x509.Certificate, error) {
	caBundle := []*x509.Certificate{cs.Config.Certificates.Ca.Cert}

	// we take only root certificate from the chain (last)
	certs := cs.Config.Certificates.OpenShiftConsole.Certs
	caBundle = append(caBundle, certs[len(certs)-1])

	certs = cs.Config.Certificates.Router.Certs
	caBundle = append(caBundle, certs[len(certs)-1])

	return tls.UniqueCert(caBundle), nil
}
