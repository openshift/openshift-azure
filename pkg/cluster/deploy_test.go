package cluster

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	gomock "github.com/golang/mock/gomock"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_azureclient"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_azureclient/mock_storage"
)

const masterHash = "RmID82LhPjuQbCEdiVa5cGCVEkdLGD6iU6ozX3vxkD0="

func TestHashScaleSets(t *testing.T) {
	tests := []struct {
		name string
		t    map[string]interface{}
		exp  map[scalesetName]hash
	}{
		{
			name: "expect a scale set",
			t: map[string]interface{}{
				"schema": "schemaversion",
				"resources": []interface{}{
					map[string]interface{}{
						"type": "Microsoft.Compute/virtualMachineScaleSets",
						"dependsOn": []interface{}{
							"[resourceId('Microsoft.Network/virtualNetworks', 'vnet')]",
							"[resourceId('Microsoft.Network/networkSecurityGroups', 'nsg-master')]",
						},
						"sku": map[string]interface{}{
							"capacity": "3",
						},
						"name": "ss-master",
					},
					map[string]interface{}{
						"type": "Microsoft.Storage/storageAccounts",
						"name": "dsdgskjgjner",
					},
				},
			},
			exp: map[scalesetName]hash{
				"ss-master": masterHash,
			},
		},
		{
			name: "expect three scale sets",
			t: map[string]interface{}{
				"schema": "schemaversion",
				"resources": []interface{}{
					map[string]interface{}{

						"type": "Microsoft.Compute/virtualMachineScaleSets",
						"dependsOn": []interface{}{
							"[resourceId('Microsoft.Network/virtualNetworks', 'vnet')]",
						},
						"sku": map[string]interface{}{
							"capacity": "2",
						},
						"name": "ss-master",
					},
					map[string]interface{}{
						"type": "Microsoft.Compute/virtualMachineScaleSets",
						"sku": map[string]interface{}{
							"capacity": "2",
						},
						"name": "ss-infra",
					},
					map[string]interface{}{
						"type": "Microsoft.Compute/virtualMachineScaleSets",
						"sku": map[string]interface{}{
							"capacity": "1",
						},
						"name": "ss-compute",
					},
					map[string]interface{}{
						"type": "Microsoft.Storage/storageAccounts",
						"name": "dsdgskjgjner",
					},
				},
			},
			exp: map[scalesetName]hash{
				"ss-master":  masterHash,
				"ss-infra":   "aqOO0n4n/nx5onYVUEwoW3s/GCnFoEZIZBowvhaHD6c=",
				"ss-compute": "iWDo277FXQHmvzHj5z1l4o+L/hoRvVSzTGroojwA2ZU=",
			},
		},
	}

	for _, test := range tests {
		got, err := hashScaleSets(test.t)
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
