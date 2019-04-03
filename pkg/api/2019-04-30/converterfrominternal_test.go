package v20190430

import (
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"github.com/openshift/openshift-azure/pkg/api"
)

func TestFromInternal(t *testing.T) {
	tests := []struct {
		cs *api.OpenShiftManagedCluster
		oc *OpenShiftManagedCluster
	}{
		{
			cs: api.GetInternalMockCluster(true),
			oc: managedCluster(),
		},
	}

	for _, test := range tests {
		oc := FromInternal(test.cs)
		if !reflect.DeepEqual(oc, test.oc) {
			t.Errorf("unexpected result:\n%#v\nexpected:\n%#v", spew.Sprint(oc), spew.Sprint(test.oc))
		}
	}
}
