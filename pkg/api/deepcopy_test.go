package api

import (
	"reflect"
	"testing"
)

func TestDeepCopy(t *testing.T) {
	copy := testContainerService.DeepCopy()
	if !reflect.DeepEqual(testContainerService, copy) {
		t.Error("OpenShiftManagedCluster differed after DeepCopy")
	}
}
