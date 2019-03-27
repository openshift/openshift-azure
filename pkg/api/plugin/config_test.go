package plugin

import (
	"testing"

	"github.com/openshift/openshift-azure/pkg/util/structs"
)

// TestJSONTags ensures that all the `json:"..."` struct field tags under
// Config correspond with their field names
func TestJSONTags(t *testing.T) {
	o := Config{}
	for _, err := range structs.CheckJsonTags(o) {
		t.Errorf("mismatch in struct tags for %T: %s", o, err.Error())
	}
}
