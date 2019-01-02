package cluster

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster/kubeclient"
	"github.com/openshift/openshift-azure/pkg/config"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_azureclient"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_azureclient/mock_storage"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_cluster"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_kubeclient"
)

func TestUpdateWorkerAgentPool(t *testing.T) {
	testRg := "testrg"
	tests := []struct {
		name     string
		cs       *api.OpenShiftManagedCluster
		app      *api.AgentPoolProfile
		ssHashes map[scalesetName][]byte
		want     *api.PluginError
		vmsList1 []compute.VirtualMachineScaleSetVM
		vmsList2 []compute.VirtualMachineScaleSetVM
	}{
		{
			name: "basic coverage",
			app: &api.AgentPoolProfile{
				Role:  api.AgentPoolProfileRoleCompute,
				Name:  "compute",
				Count: 1,
			},
			ssHashes: map[scalesetName][]byte{"ss-compute": []byte("hashish")},
			vmsList1: []compute.VirtualMachineScaleSetVM{
				{
					Name:       to.StringPtr("ss-compute_0"),
					InstanceID: to.StringPtr("0123456"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						OsProfile: &compute.OSProfile{
							ComputerName: to.StringPtr("compute-000000"),
						},
					},
				},
			},
			vmsList2: []compute.VirtualMachineScaleSetVM{
				{
					Name:       to.StringPtr("ss-compute_0"),
					InstanceID: to.StringPtr("0123456-0"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						OsProfile: &compute.OSProfile{
							ComputerName: to.StringPtr("compute-000000"),
						},
					},
				},
				{
					Name:       to.StringPtr("ss-compute_1"),
					InstanceID: to.StringPtr("0123456-1"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						OsProfile: &compute.OSProfile{
							ComputerName: to.StringPtr("compute-000001"),
						},
					},
				},
			},
			cs: &api.OpenShiftManagedCluster{
				Properties: api.Properties{
					AzProfile: api.AzProfile{ResourceGroup: testRg},
					AgentPoolProfiles: []api.AgentPoolProfile{
						{Role: api.AgentPoolProfileRoleCompute, Name: "compute", Count: 1},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			gmc := gomock.NewController(t)
			defer gmc.Finish()

			// get updateBlob
			updateContainer := mock_storage.NewMockContainer(gmc)
			mockUpdateBlob := mock_storage.NewMockBlob(gmc)
			updateContainer.EXPECT().GetBlobReference("update").Return(mockUpdateBlob)
			data := ioutil.NopCloser(strings.NewReader(`{}`))
			mockUpdateBlob.EXPECT().Get(nil).Return(data, nil)

			vmc := mock_azureclient.NewMockVirtualMachineScaleSetVMsClient(gmc)
			ssc := mock_azureclient.NewMockVirtualMachineScaleSetsClient(gmc)

			// update scale set
			req, _ := http.NewRequest("put", "http://example.com", nil)
			fakeResp := http.Response{Request: req, StatusCode: 200}
			ft, _ := azure.NewFutureFromResponse(&fakeResp)
			vFt := compute.VirtualMachineScaleSetsUpdateFuture{Future: ft}
			ssc.EXPECT().Update(ctx, testRg, "ss-compute", compute.VirtualMachineScaleSetUpdate{
				Sku: &compute.Sku{
					Capacity: to.Int64Ptr(int64(2)),
				},
			}).Return(vFt, nil)
			arc := autorest.NewClientWithUserAgent("unittest")
			ssc.EXPECT().Client().Return(arc)
			// initial listing
			mockListVMs(ctx, gmc, vmc, config.GetScalesetName(tt.app.Name), testRg, tt.vmsList1, nil)
			// once updated to count+1
			mockListVMs(ctx, gmc, vmc, config.GetScalesetName(tt.app.Name), testRg, tt.vmsList2, nil)
			// waitforready
			client := mock_kubeclient.NewMockKubeclient(gmc)
			uBlob := newUpdateBlob()
			for _, vm := range tt.vmsList2 {
				compName := kubeclient.ComputerName(*vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
				client.EXPECT().WaitForReadyWorker(ctx, compName).Return(nil)

				// write the updatehash
				uBlob.InstanceHashes[instanceName(*vm.Name)] = tt.ssHashes[scalesetName("ss-"+string(tt.app.Role))]
				updateContainer.EXPECT().GetBlobReference("update").Return(mockUpdateBlob)
				hashData, _ := json.Marshal(uBlob)
				mockUpdateBlob.EXPECT().CreateBlockBlobFromReader(bytes.NewReader([]byte(hashData)), nil)
			}
			// delete the old node
			victim := kubeclient.ComputerName(*tt.vmsList2[0].VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
			client.EXPECT().DrainAndDeleteWorker(ctx, victim)
			vdFt := compute.VirtualMachineScaleSetVMsDeleteFuture{Future: ft}
			vmc.EXPECT().Delete(ctx, testRg, "ss-compute", gomock.Any()).Return(vdFt, nil)
			vmc.EXPECT().Client().Return(arc)

			// final updateBlob write
			delete(uBlob.InstanceHashes, instanceName(*tt.vmsList2[0].Name))
			updateContainer.EXPECT().GetBlobReference("update").Return(mockUpdateBlob)
			hashData, _ := json.Marshal(uBlob)
			mockUpdateBlob.EXPECT().CreateBlockBlobFromReader(bytes.NewReader([]byte(hashData)), nil)

			hasher := mock_cluster.NewMockHasher(gmc)
			hasher.EXPECT().HashScaleSet(gomock.Any(), gomock.Any()).Return(tt.ssHashes[scalesetName("ss-compute")], nil)

			u := &simpleUpgrader{
				updateContainer: updateContainer,
				vmc:             vmc,
				ssc:             ssc,
				storageClient:   mock_storage.NewMockClient(gmc),
				kubeclient:      client,
				log:             logrus.NewEntry(logrus.StandardLogger()).WithField("test", tt.name),
				hasher:          hasher,
			}
			if got := u.updateWorkerAgentPool(ctx, tt.cs, tt.app); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("simpleUpgrader.updateWorkerAgentPool() = %v, want %v", got, tt.want)
			}
		})
	}
}
