package validate

import (
	"fmt"
	"reflect"

	"github.com/go-test/deep"

	"github.com/openshift/openshift-azure/pkg/api"
)

func validateUpdateContainerService(cs, oldCs *api.OpenShiftManagedCluster) (errs []error) {
	if cs == nil || oldCs == nil {
		errs = append(errs, fmt.Errorf("cs and oldCs cannot be nil"))
		return
	}

	newAgents := make(map[string]*api.AgentPoolProfile)
	for i := range cs.Properties.AgentPoolProfiles {
		newAgent := cs.Properties.AgentPoolProfiles[i]
		newAgents[newAgent.Name] = &newAgent
	}

	old := oldCs.DeepCopy()

	for i, o := range old.Properties.AgentPoolProfiles {
		new, ok := newAgents[o.Name]
		if !ok {
			continue // we know we are going to fail the DeepEqual test below.
		}
		old.Properties.AgentPoolProfiles[i].Count = new.Count
	}

	if !reflect.DeepEqual(cs, old) {
		errs = append(errs, fmt.Errorf("invalid change %s", deep.Equal(cs, old)))
	}

	return
}

func validateUpdateConfig(config, oldConfig *api.Config) (errs []error) {
	if config == nil || oldConfig == nil {
		errs = append(errs, fmt.Errorf("internalConfig and oldConfig cannot be nil"))
		return
	}

	if !reflect.DeepEqual(config, oldConfig) {
		errs = append(errs, fmt.Errorf("invalid change %s", deep.Equal(config, oldConfig)))
	}

	return
}
