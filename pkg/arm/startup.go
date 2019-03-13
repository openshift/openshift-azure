package arm

import (
	"io/ioutil"
	"net"
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/template"
)

func WriteTemplatedFiles(log *logrus.Entry, cs *api.OpenShiftManagedCluster) error {
	hostname, _ := os.Hostname()
	cname, _ := net.LookupCNAME(hostname)
	domainname := strings.SplitN(strings.TrimSuffix(cname, "."), ".", 2)[1]

	for _, templateFileName := range AssetNames() {
		if hostname != "master-000000" && templateFileName == "etc/origin/node/pods/sync.yaml" {
			continue
		}
		if templateFileName == "master-startup.sh" || templateFileName == "node-startup.sh" {
			continue
		}
		log.Debugf("processing template %s", templateFileName)
		templateFile, err := Asset(templateFileName)
		if err != nil {
			return errors.Wrapf(err, "Asset(%s)", templateFileName)
		}

		b, err := template.Template(string(templateFile), nil, cs, map[string]interface{}{
			"Hostname":   hostname,
			"DomainName": domainname,
		})
		if err != nil {
			return errors.Wrapf(err, "Template(%s)", templateFileName)
		}
		destination := "/" + templateFileName
		parentDir := path.Dir(destination)
		err = os.MkdirAll(parentDir, 0755)
		if err != nil {
			return errors.Wrapf(err, "MkdirAll(%s)", parentDir)
		}
		var perm os.FileMode = 0666
		if path.Ext(destination) == ".key" ||
			path.Ext(destination) == ".kubeconfig" ||
			path.Base(destination) == "session-secrets.yaml" {
			perm = 0600
		}

		err = ioutil.WriteFile(destination, b, perm)
		if err != nil {
			return errors.Wrapf(err, "WriteFile(%s)", destination)
		}
	}
	return nil
}
