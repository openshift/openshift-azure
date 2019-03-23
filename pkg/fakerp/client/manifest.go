package client

import (
	"io/ioutil"
	"os"
	"text/template"

	"github.com/ghodss/yaml"

	v20180930preview "github.com/openshift/openshift-azure/pkg/api/2018-09-30-preview/api"
	admin "github.com/openshift/openshift-azure/pkg/api/admin/api"
	utiltemplate "github.com/openshift/openshift-azure/pkg/util/template"
)

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
	b, err := ioutil.ReadFile(manifestFile)
	if err != nil {
		return nil, err
	}

	b, err = utiltemplate.Template(manifestFile, string(b), template.FuncMap{
		"Getenv": os.Getenv,
	}, nil)
	if err != nil {
		return nil, err
	}

	oc := &v20180930preview.OpenShiftManagedCluster{}
	if err = yaml.Unmarshal(b, oc); err != nil {
		return nil, err
	}

	if oc.Properties != nil {
		oc.Properties.ProvisioningState = nil // TODO: should not need to do this
	}
	return oc, nil
}

// GenerateManifestAdmin accepts a manifest file and returns a typed
// OSA admin struct that can be used to request OSA admin updates.
// If the provided manifest is templatized, it will be parsed
// appropriately.
func GenerateManifestAdmin(manifestFile string) (*admin.OpenShiftManagedCluster, error) {
	b, err := ioutil.ReadFile(manifestFile)
	if err != nil {
		return nil, err
	}

	b, err = utiltemplate.Template(manifestFile, string(b), template.FuncMap{
		"Getenv": os.Getenv,
	}, nil)
	if err != nil {
		return nil, err
	}

	oc := &admin.OpenShiftManagedCluster{}
	if err = yaml.Unmarshal(b, oc); err != nil {
		return nil, err
	}

	if oc.Properties != nil {
		oc.Properties.ProvisioningState = nil // TODO: should not need to do this
	}
	return oc, nil
}
