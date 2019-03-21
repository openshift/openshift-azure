package startup

//go:generate go get github.com/go-bindata/go-bindata/go-bindata
//go:generate go-bindata -nometadata -pkg $GOPACKAGE -prefix data data/...
//go:generate gofmt -s -l -w bindata.go

import (
	"context"
	"crypto/sha256"
	"net"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster/names"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/template"
	"github.com/openshift/openshift-azure/pkg/util/vault"
	"github.com/openshift/openshift-azure/pkg/util/writers"
)

type startup struct {
	log *logrus.Entry
	cs  *api.OpenShiftManagedCluster
}

func New(log *logrus.Entry, cs *api.OpenShiftManagedCluster) *startup {
	return &startup{log: log, cs: cs}
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

		kvc := azureclient.NewKeyVaultClient(ctx, vaultauthorizer)

		s.log.Info("enriching config")
		err = vault.EnrichCSFromVault(ctx, kvc, s.cs)
		if err != nil {
			return err
		}
	}

	return s.writeFiles(role, writers.NewFilesystemWriter(), hostname, domainname)
}

func (s *startup) Hash(role api.AgentPoolProfileRole) ([]byte, error) {
	hash := sha256.New()

	err := s.writeFiles(role, writers.NewTarWriter(hash), "", "domainname")
	if err != nil {
		return nil, err
	}

	return hash.Sum(nil), nil
}

func (s *startup) writeFiles(role api.AgentPoolProfileRole, w writers.Writer, hostname, domainname string) error {
	assetNames := AssetNames()
	sort.Strings(assetNames)

	for _, filepath := range assetNames {
		var tmpl string

		switch role {
		case api.AgentPoolProfileRoleMaster:
			if !strings.HasPrefix(filepath, "master/") {
				continue
			}

			b, err := Asset(filepath)
			if err != nil {
				return err
			}
			tmpl = string(b)

			filepath = strings.TrimPrefix(filepath, "master")

		default:
			if !strings.HasPrefix(filepath, "worker/") {
				continue
			}

			b, err := Asset(filepath)
			if err != nil {
				return err
			}
			tmpl = string(b)

			filepath = strings.TrimPrefix(filepath, "worker")
		}

		b, err := template.Template(filepath, tmpl, nil, s.cs, map[string]interface{}{
			"Role":       role,
			"Hostname":   hostname,
			"DomainName": domainname,
		})
		if err != nil {
			return err
		}

		var perm os.FileMode
		switch {
		case strings.HasSuffix(filepath, ".key"),
			strings.HasSuffix(filepath, ".kubeconfig"),
			filepath == "/etc/origin/cloudprovider/azure.conf",
			filepath == "/etc/origin/master/session-secrets.yaml",
			filepath == "/var/lib/origin/.docker/config.json",
			filepath == "/root/.kube/config":
			perm = 0600
		default:
			perm = 0644
		}

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

	return nil
}
