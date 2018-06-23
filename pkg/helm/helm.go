package helm

//go get github.com/go-bindata/go-bindata/go-bindata
//go:generate go-bindata -nometadata -pkg $GOPACKAGE -prefix data data/...
//go:generate gofmt -s -l -w bindata.go

import (
	"github.com/jim-minter/azure-helm/pkg/api"
	"github.com/jim-minter/azure-helm/pkg/config"
	"github.com/jim-minter/azure-helm/pkg/util"
)

func Generate(m *api.Manifest, c *config.Config) ([]byte, error) {
	tmpl, err := Asset("values.yaml")
	if err != nil {
		return nil, err
	}

	return util.Template(string(tmpl), nil, m, c, nil)
}
