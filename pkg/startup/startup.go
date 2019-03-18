package startup

//go:generate go get github.com/go-bindata/go-bindata/go-bindata
//go:generate go-bindata -nometadata -pkg $GOPACKAGE -prefix data data/...
//go:generate gofmt -s -l -w bindata.go

import (
	"os"
	"path"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/template"
	"github.com/openshift/openshift-azure/pkg/util/writers"
)

func WriteStartupFiles(log *logrus.Entry, cs *api.OpenShiftManagedCluster, role api.AgentPoolProfileRole, w writers.Writer, hostname, domainname string) error {
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

		b, err := template.Template(tmpl, nil, cs, map[string]interface{}{
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
