package arm

//go:generate go get github.com/go-bindata/go-bindata/go-bindata
//go:generate go-bindata -nometadata -pkg $GOPACKAGE -prefix data data/...
//go:generate gofmt -s -l -w bindata.go

import (
	"text/template"

	"github.com/sirupsen/logrus"

	acsapi "github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util"
)

type Generator interface {
	Generate(m *acsapi.ContainerService) ([]byte, error)
}

type simpleGenerator struct {
	log *logrus.Entry
}

var _ Generator = &simpleGenerator{}

func NewSimpleGenerator(log *logrus.Entry) Generator {
	return &simpleGenerator{
		log: log,
	}
}

func (*simpleGenerator) Generate(m *acsapi.ContainerService) ([]byte, error) {
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
				return util.Template(string(masterStartup), nil, m, map[string]interface{}{"Role": role})
			} else {
				return util.Template(string(nodeStartup), nil, m, map[string]interface{}{"Role": role})
			}
		},
	}, m, nil)
}
