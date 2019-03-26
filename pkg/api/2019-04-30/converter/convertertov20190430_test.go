package converter

import (
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"github.com/openshift/openshift-azure/pkg/api"
	v20190430 "github.com/openshift/openshift-azure/pkg/api/2019-04-30/api"
)

// api.GetInternalMockCluster and v20190430ManagedCluster are defined in
// converterfromv20190430_test.go.

func TestConvertTov20190430(t *testing.T) {
	tests := []struct {
		cs *api.OpenShiftManagedCluster
		oc *v20190430.OpenShiftManagedCluster
	}{
		{
			cs: api.GetInternalMockCluster(),
			oc: v20190430ManagedCluster(),
		},
	}

	for _, test := range tests {
		oc := ConvertTov20190430(test.cs)
		if !reflect.DeepEqual(oc, test.oc) {
			t.Errorf("unexpected result:\n%#v\nexpected:\n%#v", spew.Sprint(oc), spew.Sprint(test.oc))
		}
	}
}
