package cluster

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	gomock "github.com/golang/mock/gomock"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster/updateblob"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_azureclient"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_azureclient/mock_storage"
	"github.com/openshift/openshift-azure/test/util/tls"
)

type dummyRT struct {
	req  *http.Request
	resp *http.Response
	err  error
}

var _ http.RoundTripper = &dummyRT{}

func (rt *dummyRT) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.req = req
	return rt.resp, rt.err
}

func TestEtcdRestore(t *testing.T) {
	gmc := gomock.NewController(t)
	defer gmc.Finish()

	ssc := mock_azureclient.NewMockVirtualMachineScaleSetsClient(gmc)
	vmc := mock_azureclient.NewMockVirtualMachineScaleSetVMsClient(gmc)
	storageClient := mock_storage.NewMockClient(gmc)
	u := &simpleUpgrader{
		storageClient: storageClient,
		ssc:           ssc,
		vmc:           vmc,
		rt: &dummyRT{
			resp: &http.Response{StatusCode: http.StatusOK},
		},
	}
	cs := &api.OpenShiftManagedCluster{
		Properties: api.Properties{
			FQDN:      "fqdn",
			AzProfile: api.AzProfile{ResourceGroup: "test-rg"},
			AgentPoolProfiles: []api.AgentPoolProfile{
				{Role: api.AgentPoolProfileRoleMaster, Name: "master"},
			},
		},
		Config: api.Config{
			Certificates: api.CertificateConfig{
				Ca: api.CertKeyPair{
					Cert: tls.GetDummyCertificate(),
				},
			},
		},
	}
	ctx := context.Background()

	bsa := mock_storage.NewMockBlobStorageClient(gmc)
	storageClient.EXPECT().GetBlobService().Return(bsa)

	etcdCr := mock_storage.NewMockContainer(gmc)
	bsa.EXPECT().GetContainerReference(EtcdBackupContainerName).Return(etcdCr)
	etcdCr.EXPECT().CreateIfNotExists(nil).Return(true, nil)

	updateCr := mock_storage.NewMockContainer(gmc)
	bsa.EXPECT().GetContainerReference(updateblob.UpdateContainerName).Return(updateCr)
	updateCr.EXPECT().CreateIfNotExists(nil).Return(true, nil)

	updateBlob := mock_storage.NewMockBlob(gmc)
	updateCr.EXPECT().GetBlobReference(updateblob.UpdateContainerName).Return(updateBlob)

	updateBlob.EXPECT().CreateBlockBlobFromReader(bytes.NewReader([]byte("{}")), nil).Return(nil)

	configCr := mock_storage.NewMockContainer(gmc)
	bsa.EXPECT().GetContainerReference(ConfigContainerName).Return(configCr)
	configCr.EXPECT().CreateIfNotExists(nil).Return(true, nil)

	configBlob := mock_storage.NewMockBlob(gmc)
	configCr.EXPECT().GetBlobReference(ConfigBlobName).Return(configBlob)

	csj, err := json.Marshal(cs)
	if err != nil {
		t.Fatal(err)
	}
	configBlob.EXPECT().CreateBlockBlobFromReader(bytes.NewReader(csj), nil).Return(nil)

	vmc.EXPECT().List(ctx, "test-rg", "ss-master", "", "", "").Return(nil, nil)

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
	if got := u.EtcdRestore(ctx, cs, nil, func(context.Context, map[string]interface{}) error { return nil }); got != nil {
		t.Errorf("simpleUpgrader.EtcdRestore() = %v", got)
	}
}
