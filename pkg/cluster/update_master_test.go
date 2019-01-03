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

func TestFilterOldVMs(t *testing.T) {
	tests := []struct {
		name   string
		vms    []compute.VirtualMachineScaleSetVM
		blob   *updateblob
		ssHash []byte
		exp    []compute.VirtualMachineScaleSetVM
	}{
		{
			name: "one updated, two old vms",
			vms: []compute.VirtualMachineScaleSetVM{
				{
					Name: to.StringPtr("ss-master_0"),
				},
				{
					Name: to.StringPtr("ss-master_1"),
				},
				{
					Name: to.StringPtr("ss-master_2"),
				},
			},
			blob: &updateblob{
				InstanceHashes: instanceHashMap{
					"ss-master_0": []byte("newhash"),
					"ss-master_1": []byte("oldhash"),
					"ss-master_2": []byte("oldhash"),
				},
			},
			ssHash: []byte("newhash"),
			exp: []compute.VirtualMachineScaleSetVM{
				{
					Name: to.StringPtr("ss-master_1"),
				},
				{
					Name: to.StringPtr("ss-master_2"),
				},
			},
		},
		{
			name: "all updated",
			vms: []compute.VirtualMachineScaleSetVM{
				{
					Name: to.StringPtr("ss-master_0"),
				},
				{
					Name: to.StringPtr("ss-master_1"),
				},
				{
					Name: to.StringPtr("ss-master_2"),
				},
			},
			blob: &updateblob{
				InstanceHashes: instanceHashMap{
					"ss-master_0": []byte("newhash"),
					"ss-master_1": []byte("newhash"),
					"ss-master_2": []byte("newhash"),
				},
			},
			ssHash: []byte("newhash"),
			exp:    nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			u := &simpleUpgrader{
				log: logrus.NewEntry(logrus.StandardLogger()).WithField("test", test.name),
			}
			got := u.filterOldVMs(test.vms, test.blob, test.ssHash)
			if !reflect.DeepEqual(got, test.exp) {
				t.Errorf("expected vms:\n%#v\ngot:\n%#v", test.exp, got)
			}
		})
	}
}

func TestUpdateMasterAgentPool(t *testing.T) {
	testRg := "testrg"
	tests := []struct {
		name     string
		app      *api.AgentPoolProfile
		cs       *api.OpenShiftManagedCluster
		ssHashes map[string][]byte
		vmsList  []compute.VirtualMachineScaleSetVM
		want     *api.PluginError
	}{
		{
			name:     "basic coverage",
			app:      &api.AgentPoolProfile{Role: api.AgentPoolProfileRoleMaster, Name: "master", Count: 1},
			ssHashes: map[string][]byte{"ss-master": []byte("hashish")},
			vmsList: []compute.VirtualMachineScaleSetVM{
				{
					Name:       to.StringPtr("ss-master_0"),
					InstanceID: to.StringPtr("0123456"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						OsProfile: &compute.OSProfile{
							ComputerName: to.StringPtr("master-000000"),
						},
					},
				},
			},
			cs: &api.OpenShiftManagedCluster{
				Properties: api.Properties{
					AzProfile: api.AzProfile{ResourceGroup: testRg},
					AgentPoolProfiles: []api.AgentPoolProfile{
						{Role: api.AgentPoolProfileRoleMaster, Name: "master", Count: 1},
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
			client := mock_kubeclient.NewMockKubeclient(gmc)
			virtualMachineScaleSetsClient := mock_azureclient.NewMockVirtualMachineScaleSetsClient(gmc)
			virtualMachineScaleSetVMsClient := mock_azureclient.NewMockVirtualMachineScaleSetVMsClient(gmc)
			storageClient := mock_storage.NewMockClient(gmc)
			mockUpdateBlob := mock_storage.NewMockBlob(gmc)
			updateContainer := mock_storage.NewMockContainer(gmc)
			updateContainer.EXPECT().GetBlobReference("update").Return(mockUpdateBlob)
			data := ioutil.NopCloser(strings.NewReader(`{}`))
			mockUpdateBlob.EXPECT().Get(nil).Return(data, nil)
			virtualMachineScaleSetVMsClient.EXPECT().List(ctx, testRg, config.GetScalesetName(tt.app, ""), "", "", "").Return(tt.vmsList, nil)
			uBlob := newUpdateBlob()
			hasher := mock_cluster.NewMockHasher(gmc)
			hasher.EXPECT().HashScaleSet(gomock.Any(), gomock.Any()).Return(tt.ssHashes["ss-master"], nil)
			for _, vm := range tt.vmsList {
				compName := kubeclient.ComputerName(*vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)

				// 1 drain
				client.EXPECT().DeleteMaster(compName).Return(nil)

				// 2 deallocate
				arc := autorest.NewClientWithUserAgent("unittest")
				req, _ := http.NewRequest("delete", "http://example.com", nil)
				fakeResp := http.Response{Request: req, StatusCode: 200}
				ft, _ := azure.NewFutureFromResponse(&fakeResp)
				{
					virtualMachineScaleSetVMsClient.EXPECT().Client().Return(arc)
					vFt := compute.VirtualMachineScaleSetVMsDeallocateFuture{Future: ft}
					virtualMachineScaleSetVMsClient.EXPECT().Deallocate(ctx, testRg, "ss-master", *vm.InstanceID).Return(vFt, nil)
				}
				// 3  updateinstances
				{
					virtualMachineScaleSetsClient.EXPECT().Client().Return(arc)
					vFt := compute.VirtualMachineScaleSetsUpdateInstancesFuture{Future: ft}
					virtualMachineScaleSetsClient.EXPECT().UpdateInstances(ctx, testRg, "ss-master", compute.VirtualMachineScaleSetVMInstanceRequiredIDs{
						InstanceIds: &[]string{*vm.InstanceID},
					}).Return(vFt, nil)
				}
				// 4. reimage
				{
					virtualMachineScaleSetVMsClient.EXPECT().Client().Return(arc)
					vFt := compute.VirtualMachineScaleSetVMsReimageFuture{Future: ft}
					virtualMachineScaleSetVMsClient.EXPECT().Reimage(ctx, testRg, "ss-master", *vm.InstanceID).Return(vFt, nil)
				}
				// 5. start
				{
					virtualMachineScaleSetVMsClient.EXPECT().Client().Return(arc)
					vFt := compute.VirtualMachineScaleSetVMsStartFuture{Future: ft}
					virtualMachineScaleSetVMsClient.EXPECT().Start(ctx, testRg, "ss-master", *vm.InstanceID).Return(vFt, nil)
				}
				// 6. waitforready
				client.EXPECT().WaitForReadyMaster(ctx, compName).Return(nil)
				uBlob.InstanceHashes[*vm.Name] = tt.ssHashes["ss-master"]

				// write the updatehash
				hashData, _ := json.Marshal(uBlob)
				mockUpdateBlob.EXPECT().CreateBlockBlobFromReader(bytes.NewReader([]byte(hashData)), nil)
				updateContainer.EXPECT().GetBlobReference("update").Return(mockUpdateBlob)
			}
			u := &simpleUpgrader{
				updateContainer: updateContainer,
				vmc:             virtualMachineScaleSetVMsClient,
				ssc:             virtualMachineScaleSetsClient,
				storageClient:   storageClient,
				kubeclient:      client,
				log:             logrus.NewEntry(logrus.StandardLogger()).WithField("test", tt.name),
				hasher:          hasher,
			}
			if got := u.updateMasterAgentPool(ctx, tt.cs, tt.app); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("simpleUpgrader.updateInPlace() = %v, want %v", got, tt.want)
			}
		})
	}
}
