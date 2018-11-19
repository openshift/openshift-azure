package fakerp

import (
	"bytes"
	"html/template"
	"os"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"

	sdk "github.com/openshift/openshift-azure/pkg/util/azureclient/osa-go-sdk/services/containerservice/mgmt/2018-09-30-preview/containerservice"
)

func readEnv() map[string]string {
	env := make(map[string]string)
	for _, setting := range os.Environ() {
		pair := strings.SplitN(setting, "=", 2)
		env[pair[0]] = pair[1]
	}
	return env
}

// GenerateManifest returns the input manifest using the envirionment
func GenerateManifest(manifestFile string) (*sdk.OpenShiftManagedCluster, error) {
	t, err := template.ParseFiles(manifestFile)
	if err != nil {
		return nil, errors.Wrapf(err, "failed parsing the manifest %s", manifestFile)
	}

	b := &bytes.Buffer{}
	err = t.Execute(b, struct{ Env map[string]string }{Env: readEnv()})
	if err != nil {
		return nil, errors.Wrapf(err, "failed templating the manifest")
	}

	oc := &sdk.OpenShiftManagedCluster{}
	if err = yaml.Unmarshal(b.Bytes(), oc); err != nil {
		return nil, err
	}

	if oc.OpenShiftManagedClusterProperties != nil {
		oc.OpenShiftManagedClusterProperties.ProvisioningState = nil
	}
	return oc, nil
}
