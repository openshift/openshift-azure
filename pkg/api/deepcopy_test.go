package api

import (
	"reflect"
	"testing"
)

func TestDeepCopy(t *testing.T) {
	copy := internalManagedCluster.DeepCopy()
	if !reflect.DeepEqual(internalManagedCluster, copy) {
		t.Error("OpenShiftManagedCluster differed after DeepCopy")
	}
	copy.Tags["test"] = "update"
	if reflect.DeepEqual(internalManagedCluster, copy) {
		t.Error("copy should differ from testContainerService after mutation")
	}
}
