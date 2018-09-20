package config

import (
	"github.com/ghodss/yaml"
	acsapi "github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/jsonpath"
	"github.com/openshift/openshift-azure/pkg/log"
	"github.com/openshift/openshift-azure/pkg/translate"
)

type NestedFlags int

const (
	NestedFlagsBase64 NestedFlags = (1 << iota)
)

func generateControlPlaceConfig(cs *acsapi.OpenShiftManagedCluster) {
	//c := cs.Config

	masterConfig, err := Asset("master-config/master-config.yaml")
	if err != nil {
		panic(err)
	}
	log.Debug(string(masterConfig))

	var o map[string]interface{}
	err = yaml.Unmarshal(masterConfig, &o)
	if err != nil {
		panic(err)
	}

	authConfig := o["authConfig"].(map[string]interface{})
	log.Debug(authConfig)

	err = translateAsset(o, cs)
	if err != nil {
		panic(err)
	}

	log.Debug(o["apiVersion"])

}

func translateAsset(o map[string]interface{}, cs *acsapi.OpenShiftManagedCluster) error {
	for _, tr := range Translations["master-config/master-config.yaml"] {

		err := translate.Translate(o, tr.Path, tr.NestedPath, tr.NestedFlags, "test")
		if err != nil {
			return err
		}
	}
	return nil
}

var Translations = map[string][]translate.Translation{
	"master-config/master-config.yaml": {
		{
			Path:     jsonpath.MustCompile("$.apiVersion"),
			Template: "test",
		},
	},
	"master-config/etcd.conf": {
		{
			Path:     jsonpath.MustCompile("$.spec.caBundle"),
			Template: "{{ Base64Encode (CertAsBytes .Config.Certificates.ServiceSigningCa.Cert) }}",
		},
	},
}
