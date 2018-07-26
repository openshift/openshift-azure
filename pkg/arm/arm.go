package arm

//go:generate go get github.com/go-bindata/go-bindata/go-bindata
//go:generate go-bindata -nometadata -pkg $GOPACKAGE -prefix data data/...
//go:generate gofmt -s -l -w bindata.go

import (
	"text/template"

	acsapi "github.com/Azure/acs-engine/pkg/api"

	"github.com/openshift/openshift-azure/pkg/config"
	"github.com/openshift/openshift-azure/pkg/util"
)

func Generate(m *acsapi.ContainerService, c *config.Config) ([]byte, error) {
	masterStartup, err := Asset("master-startup.sh")
	if err != nil {
		return nil, err
	}

	nodeStartup, err := Asset("node-startup.sh")
	if err != nil {
		return nil, err
	}

	tmpl, err := Asset("azuredeploy.json")
	if err != nil {
		return nil, err
	}
	return util.Template(string(tmpl), template.FuncMap{
		"Startup": func(role acsapi.AgentPoolProfileRole) ([]byte, error) {
			if role == acsapi.AgentPoolProfileRoleMaster {
				return util.Template(string(masterStartup), nil, m, c, map[string]interface{}{"Role": role})
			} else {
				return util.Template(string(nodeStartup), nil, m, c, map[string]interface{}{"Role": role})
			}
		},
	}, m, c, nil)
}
