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

	"github.com/openshift/openshift-azure/pkg/api"
)

type Generator interface {
	Generate(ctx context.Context, cs *api.OpenShiftManagedCluster, backupBlob string, isUpdate bool, suffix string) (map[string]interface{}, error)
}

type simpleGenerator struct {
	pluginConfig api.PluginConfig
}

var _ Generator = &simpleGenerator{}

type armTemplate struct {
	Schema         string        `json:"$schema,omitempty"`
	ContentVersion string        `json:"contentVersion,omitempty"`
	Parameters     struct{}      `json:"parameters,omitempty"`
	Variables      struct{}      `json:"variables,omitempty"`
	Resources      []interface{} `json:"resources,omitempty"`
	Outputs        struct{}      `json:"outputs,omitempty"`
}

// NewSimpleGenerator create a new SimpleGenerator
func NewSimpleGenerator(pluginConfig *api.PluginConfig) Generator {
	return &simpleGenerator{
		pluginConfig: *pluginConfig,
	}
}

func (g *simpleGenerator) Generate(ctx context.Context, cs *api.OpenShiftManagedCluster, backupBlob string, isUpdate bool, suffix string) (map[string]interface{}, error) {
	t := armTemplate{
		Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
		ContentVersion: "1.0.0.0",
		Resources: []interface{}{
			vnet(cs),
			eipAPIServer(cs),
			elbAPIServer(&g.pluginConfig, cs),
			ilbAPIServer(&g.pluginConfig, cs),
			storageRegistry(cs),
			nsgMaster(cs),
		},
	}
	if !isUpdate {
		t.Resources = append(t.Resources, ipOutbound(cs), lbKubernetes(&g.pluginConfig, cs), nsgWorker(cs))
	}
	for _, app := range cs.Properties.AgentPoolProfiles {
		if app.Role == api.AgentPoolProfileRoleMaster || !isUpdate {
			vmss, err := Vmss(&g.pluginConfig, cs, &app, backupBlob, suffix)
			if err != nil {
				return nil, err
			}
			t.Resources = append(t.Resources, vmss)
		}
	}

	b, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}

	var azuretemplate map[string]interface{}
	err = json.Unmarshal(b, &azuretemplate)
	if err != nil {
		return nil, err
	}

	fixupAPIVersions(azuretemplate)
	fixupDepends(&cs.Properties.AzProfile, azuretemplate)

	return azuretemplate, nil
}
