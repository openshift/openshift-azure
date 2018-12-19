package cluster

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	gomock "github.com/golang/mock/gomock"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_azureclient"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_azureclient/mock_storage"
)

const masterHash = "RmID82LhPjuQbCEdiVa5cGCVEkdLGD6iU6ozX3vxkD0="

func TestHashScaleSets(t *testing.T) {
	tests := []struct {
		name string
		vmss *compute.VirtualMachineScaleSet
		exp  hash
	}{
		{
			name: "expect a scale set",
			vmss: &compute.VirtualMachineScaleSet{
				Sku:  &compute.Sku{},
				Name: to.StringPtr("ss-master"),
				Type: to.StringPtr("Microsoft.Compute/virtualMachineScaleSets"),
			},
			exp: masterHash,
		},
		{
			name: "hash is invariant with capacity",
			vmss: &compute.VirtualMachineScaleSet{
				Sku: &compute.Sku{
					Capacity: to.Int64Ptr(3),
				},
				Name: to.StringPtr("ss-master"),
				Type: to.StringPtr("Microsoft.Compute/virtualMachineScaleSets"),
			},
			exp: masterHash,
		},
	}

	for _, test := range tests {
		got, err := hashVMSS(test.vmss)
		if err != nil {
			t.Errorf("%s: unexpected error: %v", test.name, err)
		}
		if !reflect.DeepEqual(got, test.exp) {
			t.Errorf("%s: expected:\n%#v\ngot:\n%#v", test.name, test.exp, got)
		}
	}
}

func TestEvacuate(t *testing.T) {
	gmc := gomock.NewController(t)
	ssc := mock_azureclient.NewMockVirtualMachineScaleSetsClient(gmc)
	storageClient := mock_storage.NewMockClient(gmc)
	u := &simpleUpgrader{
		pluginConfig:  api.PluginConfig{},
		storageClient: storageClient,
		ssc:           ssc,
	}
	cs := &api.OpenShiftManagedCluster{
		Properties: api.Properties{
			AzProfile: api.AzProfile{ResourceGroup: "test-rg"},
			AgentPoolProfiles: []api.AgentPoolProfile{
				{Role: api.AgentPoolProfileRoleMaster, Name: "master"},
			},
		},
	}
	ctx := context.Background()

	bsa := mock_storage.NewMockBlobStorageClient(gmc)
	storageClient.EXPECT().GetBlobService().Return(bsa)
	updateCr := mock_storage.NewMockContainer(gmc)
	bsa.EXPECT().GetContainerReference("update").Return(updateCr)
	updateBlob := mock_storage.NewMockBlob(gmc)
	updateCr.EXPECT().GetBlobReference("update").Return(updateBlob)
	updateBlob.EXPECT().Delete(nil).Return(nil)

	arc := autorest.NewClientWithUserAgent("unittest")
	ssc.EXPECT().Client().Return(arc)

	req, _ := http.NewRequest("delete", "http://example.com", nil)
	fakeResp := http.Response{
		Request:    req,
		StatusCode: 200,
	}
	ft, _ := azure.NewFutureFromResponse(&fakeResp)
	sscFt := compute.VirtualMachineScaleSetsDeleteFuture{Future: ft}

	ssc.EXPECT().Delete(ctx, "test-rg", "ss-master").Return(sscFt, nil)
	if got := u.Evacuate(ctx, cs); got != nil {
		t.Errorf("simpleUpgrader.Evacuate() = %v", got)
	}
}
