package startup

//go:generate go get github.com/go-bindata/go-bindata/go-bindata
//go:generate go-bindata -nometadata -pkg $GOPACKAGE -prefix data data/...
//go:generate gofmt -s -l -w bindata.go

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/arm/constants"
	"github.com/openshift/openshift-azure/pkg/cluster/names"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/keyvault"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/network"
	"github.com/openshift/openshift-azure/pkg/util/enrich"
	"github.com/openshift/openshift-azure/pkg/util/template"
	"github.com/openshift/openshift-azure/pkg/util/writers"
)

type startup struct {
	log        *logrus.Entry
	cs         *api.OpenShiftManagedCluster
	testConfig api.TestConfig
}

// New returns a new startup entrypoint
func New(log *logrus.Entry, cs *api.OpenShiftManagedCluster, testConfig api.TestConfig) *startup {
	return &startup{log: log, cs: cs, testConfig: testConfig}
}

func (s *startup) GetWorkerCs() *api.OpenShiftManagedCluster {
	workerCS := &api.OpenShiftManagedCluster{
		ID:   s.cs.ID,
		Name: s.cs.Name,
		Properties: api.Properties{
			PrivateAPIServer: s.cs.Properties.PrivateAPIServer,
			NetworkProfile: api.NetworkProfile{
				Nameservers: s.cs.Properties.NetworkProfile.Nameservers,
			},
			WorkerServicePrincipalProfile: api.ServicePrincipalProfile{
				ClientID: s.cs.Properties.WorkerServicePrincipalProfile.ClientID,
				Secret:   s.cs.Properties.WorkerServicePrincipalProfile.Secret,
			},
			AzProfile: api.AzProfile{
				TenantID:       s.cs.Properties.AzProfile.TenantID,
				SubscriptionID: s.cs.Properties.AzProfile.SubscriptionID,
				ResourceGroup:  s.cs.Properties.AzProfile.ResourceGroup,
			},
		},
		Location: s.cs.Location,
		Config: api.Config{
			PluginVersion: s.cs.Config.PluginVersion,
			ComponentLogLevel: api.ComponentLogLevel{
				Node: s.cs.Config.ComponentLogLevel.Node,
			},
			Certificates: api.CertificateConfig{
				Ca: api.CertKeyPair{
					Cert: s.cs.Config.Certificates.Ca.Cert,
				},
				GenevaLogging: s.cs.Config.Certificates.GenevaLogging,
				NodeBootstrap: s.cs.Config.Certificates.NodeBootstrap,
			},
			Images: api.ImageConfig{
				Format: s.cs.Config.Images.Format,
				Node:   s.cs.Config.Images.Node,
			},
			NodeBootstrapKubeconfig:              s.cs.Config.NodeBootstrapKubeconfig,
			SDNKubeconfig:                        s.cs.Config.SDNKubeconfig,
			GenevaLoggingAccount:                 s.cs.Config.GenevaLoggingAccount,
			GenevaLoggingNamespace:               s.cs.Config.GenevaLoggingNamespace,
			GenevaLoggingControlPlaneEnvironment: s.cs.Config.GenevaLoggingControlPlaneEnvironment,
			GenevaLoggingControlPlaneAccount:     s.cs.Config.GenevaLoggingControlPlaneAccount,
			GenevaLoggingControlPlaneRegion:      s.cs.Config.GenevaLoggingControlPlaneRegion,
		},
	}
	for _, app := range s.cs.Properties.AgentPoolProfiles {
		workerCS.Properties.AgentPoolProfiles = append(workerCS.Properties.AgentPoolProfiles, api.AgentPoolProfile{
			Role:   app.Role,
			VMSize: app.VMSize,
		})
	}
	return workerCS
}

func (s *startup) WriteFiles(ctx context.Context) error {
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}

	cname, err := net.LookupCNAME(hostname)
	if err != nil {
		return err
	}

	domainname := strings.SplitN(strings.TrimSuffix(cname, "."), ".", 2)[1]

	role := names.GetAgentRole(hostname)

	spp := &s.cs.Properties.WorkerServicePrincipalProfile
	if role == api.AgentPoolProfileRoleMaster {
		spp = &s.cs.Properties.MasterServicePrincipalProfile

		s.log.Info("creating clients")
		vaultauthorizer, err := azureclient.NewAuthorizer(spp.ClientID, spp.Secret, s.cs.Properties.AzProfile.TenantID, azureclient.KeyVaultEndpoint)
		if err != nil {
			return err
		}

		kvc := keyvault.NewKeyVaultClient(ctx, s.log, vaultauthorizer)

		s.log.Info("enriching config")
		err = enrich.CertificatesFromVault(ctx, kvc, s.cs)
		if err != nil {
			return err
		}
	}

	return s.writeFiles(role, writers.NewFilesystemWriter(), hostname, domainname)
}

func (s *startup) Hash(role api.AgentPoolProfileRole) ([]byte, error) {
	hash := sha256.New()

	err := s.writeFiles(role, writers.NewTarWriter(hash), "", "")
	if err != nil {
		return nil, err
	}

	if s.testConfig.DebugHashFunctions {
		buf := &bytes.Buffer{}
		err = s.writeFiles(role, writers.NewTarWriter(buf), "", "")
		if err != nil {
			return nil, err
		}
		err = ioutil.WriteFile(fmt.Sprintf("startup-%s-%d.tar", role, time.Now().UnixNano()), buf.Bytes(), 0666)
		if err != nil {
			return nil, err
		}
	}

	return hash.Sum(nil), nil
}

func (s *startup) realFilePathAndContents(assetPath string, role api.AgentPoolProfileRole) (string, string, error) {
	prefix := strings.Split(assetPath, "/")[0]
	if prefix != "common" {
		switch role {
		case api.AgentPoolProfileRoleMaster:
			if prefix != "master" {
				return "", "", nil // skip
			}
		default:
			if prefix != "worker" {
				return "", "", nil // skip
			}
		}
	}
	b, err := Asset(assetPath)
	if err != nil {
		return "", "", err
	}
	return strings.TrimPrefix(assetPath, prefix), string(b), nil
}

func (s *startup) filePermissions(filepath string) os.FileMode {
	var perm os.FileMode
	switch {
	case strings.HasSuffix(filepath, ".key"),
		strings.HasSuffix(filepath, ".kubeconfig"),
		filepath == "/etc/origin/cloudprovider/azure.conf",
		filepath == "/etc/origin/master/client.secret",
		filepath == "/etc/origin/master/session-secrets.yaml",
		filepath == "/var/lib/origin/.docker/config.json",
		filepath == "/root/.kube/config":
		perm = 0600
	default:
		perm = 0644
	}
	return perm
}

func (s *startup) writeFiles(role api.AgentPoolProfileRole, w writers.Writer, hostname, domainname string) error {
	assetNames := AssetNames()
	sort.Strings(assetNames)
	var filesToWrite = map[string]string{}
	var fileKeys = []string{}

	// load all files into a map, common/ will be first (as it is sorted) and later
	// role-specific files will overwrite the common ones
	for _, assetPath := range assetNames {
		filepath, fileContent, err := s.realFilePathAndContents(assetPath, role)
		if err != nil {
			return err
		}
		if filepath != "" {
			filesToWrite[filepath] = fileContent
			fileKeys = append(fileKeys, filepath)
		}
	}

	// write the final map file to disk using fileKeys slice to guarantee order
	for _, filepath := range fileKeys {
		var tmpl = filesToWrite[filepath]
		b, err := template.Template(filepath, tmpl,
			map[string]interface{}{
				"Deref": func(pi *int) int { return *pi },
			}, struct {
				ContainerService *api.OpenShiftManagedCluster
				Config           *api.Config
				Derived          *derivedType
				Role             api.AgentPoolProfileRole
				Hostname         string
				DomainName       string
			}{
				ContainerService: s.cs,
				Config:           &s.cs.Config,
				Derived:          derived,
				Role:             role,
				Hostname:         hostname,
				DomainName:       domainname,
			})
		if err != nil {
			return err
		}

		perm := s.filePermissions(filepath)
		filepath = "/host" + filepath

		parentDir := path.Dir(filepath)
		err = w.MkdirAll(parentDir, 0755)
		if err != nil {
			return err
		}

		err = w.WriteFile(filepath, b, perm)
		if err != nil {
			return err
		}
	}

	return w.Close()
}

// WriteSearchDomain queries for the search domain and writes to
// /etc/dhcp/dhclient-eth0.conf.  This is for private api clusters only
func (s *startup) WriteSearchDomain(ctx context.Context, log *logrus.Entry) error {
	spp := &s.cs.Properties.WorkerServicePrincipalProfile

	s.log.Info("creating clients")
	authorizer, err := azureclient.NewAuthorizer(spp.ClientID, spp.Secret, s.cs.Properties.AzProfile.TenantID, "")
	if err != nil {
		return err
	}

	ncli := network.NewInterfacesClient(ctx, log, s.cs.Properties.AzProfile.SubscriptionID, authorizer)
	var dnsString string
	nic, err := ncli.GetVirtualMachineScaleSetNetworkInterface(ctx, s.cs.Properties.AzProfile.ResourceGroup, names.MasterScalesetName, "0", "nic", "")
	if err != nil {
		return err
	}
	dnsString = to.String(nic.DNSSettings.InternalDomainNameSuffix)

	s.log.Infof("writing custom dns %s", dnsString)

	domainSettings := []string{
		"interface \"eth0\" {\n",
		fmt.Sprintf("    supersede domain-name \"%s\";\n", dnsString),
		fmt.Sprintf("    supersede domain-search \"%s\";\n", dnsString),
		"}\n",
	}

	err = writeContentsToFile("/host/etc/dhcp/dhclient-eth0.conf", domainSettings)
	if err != nil {
		return err
	}

	err = writeContentsToFile("/host/etc/dnsmasq.conf", []string{fmt.Sprintf("server=/%s/%s\n", dnsString, constants.AzureNameserver)})
	return err
}

func writeContentsToFile(filename string, contents []string) error {
	fd, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	for _, c := range contents {
		fd.WriteString(c)
	}
	defer fd.Close()

	return nil
}
