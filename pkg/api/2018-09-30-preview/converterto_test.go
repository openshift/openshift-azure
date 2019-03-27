package v20180930preview

import (
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"github.com/openshift/openshift-azure/pkg/api"
)

func TestConvertTo(t *testing.T) {
	tests := []struct {
		cs *api.OpenShiftManagedCluster
		oc *OpenShiftManagedCluster
	}{
		{
			cs: api.GetInternalMockCluster(),
			oc: managedCluster(),
		},
	}

	for _, test := range tests {
		oc := ConvertTo(test.cs)
		if !reflect.DeepEqual(oc, test.oc) {
			t.Errorf("unexpected result:\n%#v\nexpected:\n%#v", spew.Sprint(oc), spew.Sprint(test.oc))
		}
	}
}
