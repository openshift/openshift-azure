package api

import (
	"reflect"
	"testing"
)

// testContainerService and testOpenShiftCluster are defined in
// converterfromosaapi_test.go.

func TestConvertOpenShiftManagedClusterToV1OpenShiftManagedCluster(t *testing.T) {
	oc := ConvertOpenShiftManagedClusterToV1OpenShiftManagedCluster(testContainerService)
	if !reflect.DeepEqual(oc, testOpenShiftCluster) {
		t.Errorf("ConvertOpenShiftManagedClusterToV1OpenShiftManagedCluster returned unexpected result\n%#v\n", oc)
	}
}
