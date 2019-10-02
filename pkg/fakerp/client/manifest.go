package client

import (
	"io/ioutil"

	"github.com/ghodss/yaml"

	utiltemplate "github.com/openshift/openshift-azure/pkg/util/template"
)

// WriteClusterConfigToManifest write to file
func WriteClusterConfigToManifest(oc interface{}, manifestFile string) error {
	out, err := yaml.Marshal(oc)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(manifestFile, out, 0666)
}

// GenerateManifest accepts a manifest file and returns a OSA struct of the type
// requested by the caller.  If the provided manifest is templatized, it will be
// parsed appropriately.
func GenerateManifest(conf *Config, i interface{}) error {
	b, err := ioutil.ReadFile(conf.Manifest)
	if err != nil {
		return err
	}

	b, err = utiltemplate.Template(conf.Manifest, string(b), nil, conf)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(b, i)
}
