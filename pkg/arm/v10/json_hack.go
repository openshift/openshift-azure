package arm

import (
	"encoding/json"

	"github.com/openshift/openshift-azure/pkg/util/arm"
)

// HACK: Don't spread into the new versions and don't try to make it reusable.
// It's here only to fix hashes and prevent node rotations
// after upgrading to the latest version of Azure SDK where JSON marshaling changed.
// Just leave it here and let it die with the old versions of the ARO plugins.

// fixupArmResource sorts top-level fields, but keeps underlying marshaling logic intact
type fixupArmResource struct {
	*arm.Resource
}

func (r *fixupArmResource) MarshalJSON() ([]byte, error) {
	b, err := json.Marshal(r.Resource)
	if err != nil {
		return nil, err
	}

	var fixupMap map[string]*json.RawMessage
	err = json.Unmarshal(b, &fixupMap)
	if err != nil {
		return nil, err
	}

	return json.Marshal(fixupMap)
}
