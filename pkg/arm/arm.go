package arm

//go:generate go get github.com/go-bindata/go-bindata/go-bindata
//go:generate go-bindata -nometadata -pkg $GOPACKAGE -prefix data data/...
//go:generate gofmt -s -l -w bindata.go

import (
	"text/template"

	acsapi "github.com/Azure/acs-engine/pkg/api"

	"github.com/jim-minter/azure-helm/pkg/config"
	"github.com/jim-minter/azure-helm/pkg/util"
)

func Generate(m *acsapi.ContainerService, c *config.Config) ([]byte, error) {
	startup, err := Asset("startup.sh")
	if err != nil {
		return nil, err
	}

	tmpl, err := Asset("azuredeploy.json")
	if err != nil {
		return nil, err
	}
	return util.Template(string(tmpl), template.FuncMap{
		"Startup": func(role string) ([]byte, error) {
			return util.Template(string(startup), nil, m, c, map[string]interface{}{"Role": role})
		},
	}, m, c, nil)
}
