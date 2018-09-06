package api

import (
	"reflect"
	"testing"
)

// testContainerService and testOpenShiftCluster are defined in
// converterfromv20180930preview_test.go.

func TestConvertToV20180930preview(t *testing.T) {
	oc := ConvertToV20180930preview(internalManagedCluster)
	if !reflect.DeepEqual(oc, v20180930previewManagedCluster) {
		t.Errorf("ConvertToV20180930preview returned unexpected result\n%#v\n", oc)
	}
}
