package v20180930preview

import (
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"github.com/openshift/openshift-azure/pkg/api"
)

// api.GetInternalMockCluster and v20180930previewManagedCluster are defined in
// converterfromv20180930preview_test.go.

func TestConvertToV20180930preview(t *testing.T) {
	tests := []struct {
		cs *api.OpenShiftManagedCluster
		oc *OpenShiftManagedCluster
	}{
		{
			cs: api.GetInternalMockCluster(),
			oc: v20180930previewManagedCluster(),
		},
	}

	for _, test := range tests {
		oc := ConvertToV20180930preview(test.cs)
		if !reflect.DeepEqual(oc, test.oc) {
			t.Errorf("unexpected result:\n%#v\nexpected:\n%#v", spew.Sprint(oc), spew.Sprint(test.oc))
		}
	}
}
