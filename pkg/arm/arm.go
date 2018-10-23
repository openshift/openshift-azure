package arm

//go:generate go get github.com/go-bindata/go-bindata/go-bindata
//go:generate go-bindata -nometadata -pkg $GOPACKAGE -prefix data data/...
//go:generate gofmt -s -l -w bindata.go
//go:generate go get github.com/golang/mock/gomock
//go:generate go install github.com/golang/mock/mockgen
//go:generate mockgen -destination=../util/mocks/mock_$GOPACKAGE/arm.go -package=mock_$GOPACKAGE -source arm.go
//go:generate gofmt -s -l -w ../util/mocks/mock_$GOPACKAGE/arm.go
//go:generate goimports -local=github.com/openshift/openshift-azure -e -w ../util/mocks/mock_$GOPACKAGE/arm.go

import (
	"context"
	"encoding/json"
	"text/template"

	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/log"
	"github.com/openshift/openshift-azure/pkg/util"
)

type Generator interface {
	Generate(ctx context.Context, cs *api.OpenShiftManagedCluster, isUpdate bool) (map[string]interface{}, error)
}

type simpleGenerator struct{}

var _ Generator = &simpleGenerator{}

// NewSimpleGenerator create a new SimpleGenerator
func NewSimpleGenerator(entry *logrus.Entry) Generator {
	log.New(entry)
	return &simpleGenerator{}
}

func (*simpleGenerator) Generate(ctx context.Context, cs *api.OpenShiftManagedCluster, isUpdate bool) (map[string]interface{}, error) {
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
	azuredeploy, err := util.Template(string(tmpl), template.FuncMap{
		"Startup": func(role api.AgentPoolProfileRole) ([]byte, error) {
			if role == api.AgentPoolProfileRoleMaster {
				return util.Template(string(masterStartup), nil, cs, map[string]interface{}{"Role": role})
			}
			return util.Template(string(nodeStartup), nil, cs, map[string]interface{}{"Role": role})
		},
		"IsUpgrade": func() bool {
			return isUpdate
		},
	}, cs, nil)
	if err != nil {
		return nil, err
	}

	var azuretemplate map[string]interface{}
	err = json.Unmarshal(azuredeploy, &azuretemplate)
	if err != nil {
		return nil, err
	}
	return azuretemplate, nil
}
