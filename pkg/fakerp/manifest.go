package fakerp

import (
	"bytes"
	"html/template"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

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

// LoadClusterConfigFromManifest reads (and potentially template) the mainifest
func LoadClusterConfigFromManifest(log *logrus.Entry, manifestTemplate string, conf *Config) (*sdk.OpenShiftManagedCluster, error) {
	if IsUpdate() && conf.Manifest == "" && manifestTemplate == "" {
		defaultManifestFile := path.Join(DataDirectory, "manifest.yaml")
		log.Debugf("using manifest from %q", defaultManifestFile)
		return loadManifestFromFile(defaultManifestFile)
	}
	if manifestTemplate == "" {
		manifestTemplate = conf.Manifest
	}
	if manifestTemplate == "" {
		manifestTemplate = "test/manifests/normal/create.yaml"
	}
	log.Debugf("generating manifest from %q", manifestTemplate)
	return generateManifest(manifestTemplate)
}

// WriteClusterConfigToManifest write to file
func WriteClusterConfigToManifest(oc *sdk.OpenShiftManagedCluster, manifestFile string) error {
	out, err := yaml.Marshal(oc)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(manifestFile, out, 0666)
}

func generateManifest(manifestFile string) (*sdk.OpenShiftManagedCluster, error) {
	t, err := template.ParseFiles(manifestFile)
	if err != nil {
		return nil, errors.Wrapf(err, "failed parsing the manifest %q", manifestFile)
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

func loadManifestFromFile(manifest string) (*sdk.OpenShiftManagedCluster, error) {
	in, err := ioutil.ReadFile(manifest)
	if err != nil {
		return nil, err
	}
	var oc sdk.OpenShiftManagedCluster
	if err := yaml.Unmarshal(in, &oc); err != nil {
		return nil, err
	}

	if oc.OpenShiftManagedClusterProperties != nil {
		oc.OpenShiftManagedClusterProperties.ProvisioningState = nil
	}
	return &oc, nil
}
