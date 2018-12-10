package cluster

import (
	"context"
	"net/http"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	gomock "github.com/golang/mock/gomock"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_azureclient"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_azureclient/mock_storage"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_updatehash"
)

func TestEvacuate(t *testing.T) {
	gmc := gomock.NewController(t)
	ssc := mock_azureclient.NewMockVirtualMachineScaleSetsClient(gmc)
	uh := mock_updatehash.NewMockUpdateHash(gmc)
	storageClient := mock_storage.NewMockClient(gmc)
	u := &simpleUpgrader{
		pluginConfig:  api.PluginConfig{},
		storageClient: storageClient,
		updateHash:    uh,
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
	uh.EXPECT().SetContainer(updateCr)
	uh.EXPECT().Delete()

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
