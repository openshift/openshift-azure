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

	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster/names"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/keyvault"
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

func (s *startup) writeFiles(role api.AgentPoolProfileRole, w writers.Writer, hostname, domainname string) error {
	assetNames := AssetNames()
	sort.Strings(assetNames)
	var filesToWrite = map[string]string{}
	var fileKeys = []string{}

	// load all files into a map, common/ will be first (as it is sorted) and later
	// role-specific files will overwrite the common ones
	for _, filepath := range assetNames {
		var tmpl string

		if strings.HasPrefix(filepath, "common/") {
			b, err := Asset(filepath)
			if err != nil {
				return err
			}

			filepath = strings.TrimPrefix(filepath, "common")

			filesToWrite[filepath] = string(b)

		} else {

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

			filesToWrite[filepath] = tmpl
			fileKeys = append(fileKeys, filepath)
		}
	}

	// write the final map file to disk using fileKeys slice to guarantee order
	for _, filepath := range fileKeys {
		var tmpl = filesToWrite[filepath]
		b, err := template.Template(filepath, tmpl,
			map[string]interface{}{
				"Deref": func(pi *int) int { return *pi },
			}, map[string]interface{}{
				"ContainerService": s.cs,
				"Config":           &s.cs.Config,
				"Derived":          derived,
				"Role":             role,
				"Hostname":         hostname,
				"DomainName":       domainname,
			})
		if err != nil {
			return err
		}

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
