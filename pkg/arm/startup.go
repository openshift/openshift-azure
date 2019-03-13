package arm

import (
	"io/ioutil"
	"os"
	"path"

	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/template"
)

func WriteTemplatedFiles(log *logrus.Entry, cs *api.OpenShiftManagedCluster, hostname, domainname string) error {
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
			return err
		}

		b, err := template.Template(string(templateFile), nil, cs, map[string]interface{}{
			"Hostname":   hostname,
			"DomainName": domainname,
		})
		if err != nil {
			return err
		}
		destination := "/" + templateFileName
		parentDir := path.Dir(destination)
		err = os.MkdirAll(parentDir, 0755)
		if err != nil {
			return err
		}
		var perm os.FileMode = 0666
		if path.Ext(destination) == ".key" ||
			path.Ext(destination) == ".kubeconfig" ||
			path.Base(destination) == "session-secrets.yaml" {
			perm = 0600
		}

		err = ioutil.WriteFile(destination, b, perm)
		if err != nil {
			return err
		}
	}
	return nil
}
