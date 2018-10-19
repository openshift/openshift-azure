package azureclient

import (
	"reflect"
	"testing"

	v20180930preview "github.com/openshift/openshift-azure/pkg/api/2018-09-30-preview/api"
	sdk "github.com/openshift/openshift-azure/pkg/util/azureclient/osa-go-sdk/services/containerservice/mgmt/2018-09-30-preview/containerservice"
)

func TestFieldParity(t *testing.T) {
	sdkType := reflect.TypeOf(sdk.OpenShiftManagedCluster{})
	extType := reflect.TypeOf(v20180930preview.OpenShiftManagedCluster{})

	numOfFields := extType.NumField()
	if sdkNum := sdkType.NumField(); sdkNum != numOfFields {
		t.Fatalf("number of fields mismatch: sdk type has %d fields, openshift-azure type has %d fields", sdkNum, numOfFields)
	}

	for i := 0; i < numOfFields; i++ {
		sdkField := sdkType.Field(i)
		extField := extType.Field(i)

		if sdkField.Tag != extField.Tag {
			t.Fatalf("struct field mismatch: sdk field tag is %q, openshift-azure field tag is %q", sdkField.Tag, extField.Tag)
		}
	}
}
