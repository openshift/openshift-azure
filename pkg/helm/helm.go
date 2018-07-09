package helm

//go:generate go get github.com/go-bindata/go-bindata/go-bindata
//go:generate go-bindata -nometadata -pkg $GOPACKAGE -prefix data data/...
//go:generate gofmt -s -l -w bindata.go

import (
	acsapi "github.com/Azure/acs-engine/pkg/api"

	"github.com/jim-minter/azure-helm/pkg/config"
	"github.com/jim-minter/azure-helm/pkg/util"
)

func Generate(cs *acsapi.ContainerService, c *config.Config) ([]byte, error) {
	tmpl, err := Asset("values.yaml")
	if err != nil {
		return nil, err
	}

	return util.Template(string(tmpl), nil, cs, c, nil)
}
