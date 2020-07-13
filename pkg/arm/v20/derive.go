package arm

import (
	"encoding/json"
	"strings"

	"github.com/openshift/openshift-azure/pkg/api"
)

type derivedType struct{}

var derived = &derivedType{}

func (derivedType) MasterLBCNamePrefix(cs *api.OpenShiftManagedCluster) string {
	return strings.Split(cs.Properties.FQDN, ".")[0]
}

type DockerConfigJson struct {
	Auths DockerConfig `json:"auths"`
}

type DockerConfig map[string]DockerConfigEntry

type DockerConfigEntry struct {
	Auth string `json:"auth"`
}

func (derivedType) CombinedImagePullSecret(cfg *api.Config) ([]byte, error) {
	var config DockerConfigJson
	err := json.Unmarshal(cfg.Images.ImagePullSecret, &config)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(cfg.Images.GenevaImagePullSecret, &config)
	if err != nil {
		return nil, err
	}

	return json.Marshal(config)
}
