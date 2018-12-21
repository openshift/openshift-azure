package client

import (
	"bytes"
	"html/template"
	"io/ioutil"
	"os"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"

	v20180930preview "github.com/openshift/openshift-azure/pkg/api/2018-09-30-preview/api"
)

func readEnv() map[string]string {
	env := make(map[string]string)
	for _, setting := range os.Environ() {
		pair := strings.SplitN(setting, "=", 2)
		env[pair[0]] = pair[1]
	}
	return env
}

// WriteClusterConfigToManifest write to file
func WriteClusterConfigToManifest(oc *v20180930preview.OpenShiftManagedCluster, manifestFile string) error {
	out, err := yaml.Marshal(oc)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(manifestFile, out, 0666)
}

// GenerateManifest accepts a manifest file and returns a typed OSA
// v20180930preview struct that can be used to request OSA creates
// and updates. If the provided manifest is templatized, it will be
// parsed appropriately.
func GenerateManifest(manifestFile string) (*v20180930preview.OpenShiftManagedCluster, error) {
	t, err := template.ParseFiles(manifestFile)
	if err != nil {
		return nil, errors.Wrapf(err, "failed parsing the manifest %s", manifestFile)
	}

	b := &bytes.Buffer{}
	err = t.Execute(b, struct{ Env map[string]string }{Env: readEnv()})
	if err != nil {
		return nil, errors.Wrapf(err, "failed templating the manifest")
	}

	oc := &v20180930preview.OpenShiftManagedCluster{}
	if err = yaml.Unmarshal(b.Bytes(), oc); err != nil {
		return nil, err
	}

	if oc.Properties != nil {
		oc.Properties.ProvisioningState = nil
	}
	return oc, nil
}
