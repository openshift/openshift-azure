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
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/log"
	"github.com/openshift/openshift-azure/pkg/util"
)

// TODO: Move in pkg/api to share between pkg/arm and pkg/upgrade
const HashKey = "scaleset-checksum"

type Generator interface {
	Generate(ctx context.Context, cs *api.OpenShiftManagedCluster, isUpdate bool) (map[string]interface{}, error)
}

type simpleGenerator struct {
	pluginConfig api.PluginConfig
}

var _ Generator = &simpleGenerator{}

// NewSimpleGenerator create a new SimpleGenerator
func NewSimpleGenerator(entry *logrus.Entry, pluginConfig *api.PluginConfig) Generator {
	log.New(entry)
	return &simpleGenerator{
		pluginConfig: *pluginConfig,
	}
}

func (g *simpleGenerator) Generate(ctx context.Context, cs *api.OpenShiftManagedCluster, isUpdate bool) (map[string]interface{}, error) {
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
				return util.Template(string(masterStartup), nil, cs, map[string]interface{}{
					"Role":       role,
					"TestConfig": g.pluginConfig.TestConfig,
				})
			}
			return util.Template(string(nodeStartup), nil, cs, map[string]interface{}{
				"Role":       role,
				"TestConfig": g.pluginConfig.TestConfig,
			})
		},
		"IsUpgrade": func() bool {
			return isUpdate
		},
	}, cs, map[string]interface{}{
		"TestConfig": g.pluginConfig.TestConfig,
	})
	if err != nil {
		return nil, err
	}

	var original, copied map[string]interface{}
	if err := json.Unmarshal(azuredeploy, &original); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(azuredeploy, &copied); err != nil {
		return nil, err
	}
	if err := hashScaleSets(original, copied); err != nil {
		return nil, err
	}
	return original, nil
}

func hashScaleSets(original, copied map[string]interface{}) error {
	for key, value := range copied {
		if key != "resources" {
			continue
		}

		for _, r := range value.([]interface{}) {
			resource, ok := r.(map[string]interface{})
			if !ok {
				continue
			}

			if !isScaleSet(resource) {
				continue
			}

			// cleanup capacity so that no unnecessary VM rotations are going
			// to occur because of a scale up/down.
			deleteField(resource, "sku", "capacity")

			// cleanup previous hash
			deleteField(resource, "tags", HashKey)

			// hash scale set
			data, err := json.Marshal(resource)
			if err != nil {
				return err
			}
			hf := sha256.New()
			fmt.Fprintf(hf, string(data))
			h := base64.StdEncoding.EncodeToString(hf.Sum(nil))

			// update tags in the original template
			role := getRole(resource)
			if added := addTag(role, HashKey, h, original); !added {
				return fmt.Errorf("could not tag ARM template with new hash for role %q", role)
			}
		}
	}
	return nil
}

func isScaleSet(resource map[string]interface{}) bool {
	for k, v := range resource {
		if k == "type" && v.(string) == "Microsoft.Compute/virtualMachineScaleSets" {
			return true
		}
	}
	return false
}

func deleteField(resource map[string]interface{}, parent, field string) {
	for key, value := range resource {
		if key != parent {
			continue
		}
		if p, ok := value.(map[string]interface{}); ok {
			delete(p, field)
			resource[key] = p
			return
		}
	}
}

func getRole(resource map[string]interface{}) string {
	for k, v := range resource {
		if k == "name" && strings.HasPrefix(v.(string), "ss-") {
			return v.(string)[3:]
		}
	}
	return ""
}

// addTag adds the provided key and value as a tag in the scaleset
// with the given role inside azuretemplate. It returns true when
// the tag has been added in the template, false otherwise (ie. when
// the provided role is missing from the ARM template or there is no
// scale set).
func addTag(role, key, value string, azuretemplate map[string]interface{}) bool {
	for k, resources := range azuretemplate {
		if k != "resources" {
			continue
		}

		for _, r := range resources.([]interface{}) {
			resource, ok := r.(map[string]interface{})
			if !ok {
				continue
			}

			if !isScaleSet(resource) {
				continue
			}
			if !hasRole(role, resource) {
				continue
			}
			newTags := make(map[string]interface{})
			if tags, ok := resource["tags"].(map[string]interface{}); ok {
				for k, v := range tags {
					newTags[k] = v
				}
			}
			newTags[key] = value
			resource["tags"] = newTags
			return true
		}
	}
	return false
}

func GetTag(role, key string, azuretemplate map[string]interface{}) string {
	for k, resources := range azuretemplate {
		if k != "resources" {
			continue
		}
		for _, r := range resources.([]interface{}) {
			resource, ok := r.(map[string]interface{})
			if !ok {
				continue
			}
			if !isScaleSet(resource) {
				continue
			}
			if !hasRole(role, resource) {
				continue
			}
			if tags, ok := resource["tags"].(map[string]interface{}); ok {
				if tag, ok := tags[key].(string); ok {
					return tag
				}
			}
		}
	}
	return ""
}

func hasRole(role string, resource map[string]interface{}) bool {
	for k, v := range resource {
		if k == "name" && v.(string) == "ss-"+role {
			return true
		}
	}
	return false
}
