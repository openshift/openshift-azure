package cluster

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster/kubeclient"
	"github.com/openshift/openshift-azure/pkg/cluster/updateblob"
	"github.com/openshift/openshift-azure/pkg/config"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_azureclient"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_azureclient/mock_storage"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_kubeclient"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_updatehash"
)

func TestGetNodesAndDrain(t *testing.T) {
	testRg := "testRg"
	tests := []struct {
		name        string
		cs          *api.OpenShiftManagedCluster
		want        map[kubeclient.ComputerName]struct{}
		expectDrain map[kubeclient.ComputerName]string
		wantErr     error
		vmsBefore   map[string][]compute.VirtualMachineScaleSetVM
	}{
		{
			name: "all there",
			want: map[kubeclient.ComputerName]struct{}{"compute-000000": {}, "infra-000000": {}, "master-000000": {}},
			cs: &api.OpenShiftManagedCluster{
				Properties: api.Properties{
					AzProfile: api.AzProfile{ResourceGroup: testRg},
					AgentPoolProfiles: []api.AgentPoolProfile{
						{Role: api.AgentPoolProfileRoleMaster, Name: "master", Count: 1},
						{Role: api.AgentPoolProfileRoleInfra, Name: "infra", Count: 1},
						{Role: api.AgentPoolProfileRoleCompute, Name: "compute", Count: 1},
					},
				},
			},
			vmsBefore: map[string][]compute.VirtualMachineScaleSetVM{
				"master": {
					{
						Name: to.StringPtr("ss-master"),
						VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
							OsProfile: &compute.OSProfile{
								ComputerName: to.StringPtr("master-000000"),
							},
						},
					},
				},
				"infra": {
					{
						Name: to.StringPtr("ss-infra"),
						VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
							OsProfile: &compute.OSProfile{
								ComputerName: to.StringPtr("infra-000000"),
							},
						},
					},
				},
				"compute": {
					{
						Name: to.StringPtr("ss-compute"),
						VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
							OsProfile: &compute.OSProfile{
								ComputerName: to.StringPtr("compute-000000"),
							},
						},
					},
				},
			},
		},
		{
			name:        "too many, need draining",
			want:        map[kubeclient.ComputerName]struct{}{"compute-000000": {}, "infra-000000": {}, "master-000000": {}},
			expectDrain: map[kubeclient.ComputerName]string{"compute-000001": "ss-compute"},
			cs: &api.OpenShiftManagedCluster{
				Properties: api.Properties{
					AzProfile: api.AzProfile{ResourceGroup: testRg},
					AgentPoolProfiles: []api.AgentPoolProfile{
						{Role: api.AgentPoolProfileRoleMaster, Name: "master", Count: 1},
						{Role: api.AgentPoolProfileRoleInfra, Name: "infra", Count: 1},
						{Role: api.AgentPoolProfileRoleCompute, Name: "compute", Count: 1},
					},
				},
			},
			vmsBefore: map[string][]compute.VirtualMachineScaleSetVM{
				"master": {
					{
						Name: to.StringPtr("ss-master_0"),
						VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
							OsProfile: &compute.OSProfile{
								ComputerName: to.StringPtr("master-000000"),
							},
						},
					},
				},
				"infra": {
					{
						Name: to.StringPtr("ss-infra_0"),
						VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
							OsProfile: &compute.OSProfile{
								ComputerName: to.StringPtr("infra-000000"),
							},
						},
					},
				},
				"compute": {
					{
						Name: to.StringPtr("ss-compute_0"),
						VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
							OsProfile: &compute.OSProfile{
								ComputerName: to.StringPtr("compute-000000"),
							},
						},
					},
					{
						Name: to.StringPtr("ss-compute_1"),
						VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
							OsProfile: &compute.OSProfile{
								ComputerName: to.StringPtr("compute-000001"),
							},
						},
						InstanceID: to.StringPtr("0123456"),
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
			kubeclient := mock_kubeclient.NewMockKubeclient(gmc)
			virtualMachineScaleSetsClient := mock_azureclient.NewMockVirtualMachineScaleSetsClient(gmc)
			virtualMachineScaleSetVMsClient := mock_azureclient.NewMockVirtualMachineScaleSetVMsClient(gmc)
			mockListVMs(ctx, gmc, virtualMachineScaleSetVMsClient, config.GetScalesetName("master"), testRg, tt.vmsBefore["master"], nil)
			mockListVMs(ctx, gmc, virtualMachineScaleSetVMsClient, config.GetScalesetName("infra"), testRg, tt.vmsBefore["infra"], nil)
			mockListVMs(ctx, gmc, virtualMachineScaleSetVMsClient, config.GetScalesetName("compute"), testRg, tt.vmsBefore["compute"], nil)

			for comp, scalesetName := range tt.expectDrain {
				kubeclient.EXPECT().DrainAndDeleteWorker(ctx, comp)
				arc := autorest.NewClientWithUserAgent("unittest")
				virtualMachineScaleSetVMsClient.EXPECT().Client().Return(arc)
				req, _ := http.NewRequest("delete", "http://example.com", nil)
				fakeResp := http.Response{Request: req, StatusCode: 200}
				ft, _ := azure.NewFutureFromResponse(&fakeResp)
				vFt := compute.VirtualMachineScaleSetVMsDeleteFuture{Future: ft}
				virtualMachineScaleSetVMsClient.EXPECT().Delete(ctx, testRg, scalesetName, gomock.Any()).Return(vFt, nil)
			}
			u := &simpleUpgrader{
				vmc:        virtualMachineScaleSetVMsClient,
				ssc:        virtualMachineScaleSetsClient,
				kubeclient: kubeclient,
				log:        logrus.NewEntry(logrus.StandardLogger()).WithField("test", tt.name),
			}
			got, err := u.getNodesAndDrain(ctx, tt.cs)
			if !reflect.DeepEqual(err, tt.wantErr) {
				t.Errorf("simpleUpgrader.getNodesAndDrain() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("simpleUpgrader.getNodesAndDrain() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWaitForNewNodes(t *testing.T) {
	testRg := "testResourceG"
	tests := []struct {
		name    string
		cs      *api.OpenShiftManagedCluster
		nodes   map[kubeclient.ComputerName]struct{}
		vmsList map[string][]compute.VirtualMachineScaleSetVM
		wantErr error
	}{
		{
			name:  "no new nodes",
			nodes: map[kubeclient.ComputerName]struct{}{"master-000000": {}},
			vmsList: map[string][]compute.VirtualMachineScaleSetVM{
				"master": {
					{
						Name: to.StringPtr("ss-master_0"),
						VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
							OsProfile: &compute.OSProfile{
								ComputerName: to.StringPtr("master-000000"),
							},
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
		{
			name:  "wait for new nodes",
			nodes: map[kubeclient.ComputerName]struct{}{},
			vmsList: map[string][]compute.VirtualMachineScaleSetVM{
				"master": {
					{
						Name: to.StringPtr("ss-master_0"),
						VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
							OsProfile: &compute.OSProfile{
								ComputerName: to.StringPtr("master-000000"),
							},
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
		{
			name:  "clear blob of stale instances",
			nodes: map[kubeclient.ComputerName]struct{}{"master-000000": {}},
			vmsList: map[string][]compute.VirtualMachineScaleSetVM{
				"master": {
					{
						Name: to.StringPtr("ss-master_0"),
						VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
							OsProfile: &compute.OSProfile{
								ComputerName: to.StringPtr("master-000000"),
							},
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
		{
			name:    "new node not ready",
			wantErr: fmt.Errorf("node not ready test"),
			nodes:   map[kubeclient.ComputerName]struct{}{},
			vmsList: map[string][]compute.VirtualMachineScaleSetVM{
				"master": {
					{
						Name: to.StringPtr("ss-master_0"),
						VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
							OsProfile: &compute.OSProfile{
								ComputerName: to.StringPtr("master-000000"),
							},
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
			uh := mock_updatehash.NewMockUpdateHash(gmc)
			uh.EXPECT().Reload().Return(nil)

			existingVMs := make(map[updateblob.InstanceName]struct{})
			for _, vm := range tt.vmsList["master"] {
				computerName := kubeclient.ComputerName(*vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
				existingVMs[updateblob.InstanceName(*vm.Name)] = struct{}{}
				if _, found := tt.nodes[computerName]; !found {
					client.EXPECT().WaitForReadyWorker(ctx, kubeclient.ComputerName("master-000000")).Return(tt.wantErr)
					if tt.wantErr == nil {
						uh.EXPECT().UpdateInstanceHash(&vm).Return(nil)
					}
				}
			}
			if tt.wantErr == nil {
				uh.EXPECT().DeleteAllBut(existingVMs).Return(nil)
			}
			u := &simpleUpgrader{
				updateHash:    uh,
				vmc:           virtualMachineScaleSetVMsClient,
				ssc:           virtualMachineScaleSetsClient,
				storageClient: storageClient,
				kubeclient:    client,
				log:           logrus.NewEntry(logrus.StandardLogger()).WithField("test", tt.name),
			}
			mockListVMs(ctx, gmc, virtualMachineScaleSetVMsClient, config.GetScalesetName("master"), testRg, tt.vmsList["master"], nil)

			err := u.waitForNewNodes(ctx, tt.cs, tt.nodes)
			if !reflect.DeepEqual(err, tt.wantErr) {
				t.Errorf("simpleUpgrader.waitForNewNodes() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpdateMasterAgentPool(t *testing.T) {
	testRg := "testrg"
	tests := []struct {
		name    string
		cs      *api.OpenShiftManagedCluster
		vmsList []compute.VirtualMachineScaleSetVM
		want    *api.PluginError
	}{
		{
			name: "basic coverage",
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
			uh := mock_updatehash.NewMockUpdateHash(gmc)
			storageClient := mock_storage.NewMockClient(gmc)
			mockListVMs(ctx, gmc, virtualMachineScaleSetVMsClient, config.GetScalesetName("master"), testRg, tt.vmsList, nil)
			uh.EXPECT().FilterOldVMs(tt.vmsList).Return(tt.vmsList, nil)
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

				// write the updatehash
				uh.EXPECT().UpdateInstanceHash(&vm).Return(nil)
			}
			u := &simpleUpgrader{
				updateHash:    uh,
				vmc:           virtualMachineScaleSetVMsClient,
				ssc:           virtualMachineScaleSetsClient,
				storageClient: storageClient,
				kubeclient:    client,
				log:           logrus.NewEntry(logrus.StandardLogger()).WithField("test", tt.name),
			}
			if got := u.updateMasterAgentPool(ctx, tt.cs); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("simpleUpgrader.updateInPlace() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUpdatePlusOne(t *testing.T) {
	testRg := "testrg"
	tests := []struct {
		name     string
		cs       *api.OpenShiftManagedCluster
		app      *api.AgentPoolProfile
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
			vmsList1: []compute.VirtualMachineScaleSetVM{
				{
					Name:       to.StringPtr("ss-compute_0"),
					InstanceID: to.StringPtr("0123456-0"),
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

			uh := mock_updatehash.NewMockUpdateHash(gmc)
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
			uh.EXPECT().FilterOldVMs(tt.vmsList1).Return(tt.vmsList1, nil)
			// once updated to count+1
			mockListVMs(ctx, gmc, vmc, config.GetScalesetName(tt.app.Name), testRg, tt.vmsList2, nil)
			// waitforready
			client := mock_kubeclient.NewMockKubeclient(gmc)
			vm := tt.vmsList2[1]
			compName := kubeclient.ComputerName(*vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
			client.EXPECT().WaitForReadyWorker(ctx, compName).Return(nil)

			// write the updatehash
			uh.EXPECT().UpdateInstanceHash(&vm).Return(nil)

			// delete the old node
			victim := kubeclient.ComputerName(*tt.vmsList2[0].VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
			client.EXPECT().DrainAndDeleteWorker(ctx, victim)
			vdFt := compute.VirtualMachineScaleSetVMsDeleteFuture{Future: ft}
			vmc.EXPECT().Delete(ctx, testRg, "ss-compute", gomock.Any()).Return(vdFt, nil)
			vmc.EXPECT().Client().Return(arc)

			// final updateBlob write
			uh.EXPECT().DeleteInstanceHash(updateblob.InstanceName(*tt.vmsList2[0].Name)).Return(nil)

			u := &simpleUpgrader{
				updateHash:    uh,
				vmc:           vmc,
				ssc:           ssc,
				storageClient: mock_storage.NewMockClient(gmc),
				kubeclient:    client,
				log:           logrus.NewEntry(logrus.StandardLogger()).WithField("test", tt.name),
			}
			if got := u.updatePlusOne(ctx, tt.cs, tt.app); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("simpleUpgrader.updatePlusOne() = %v, want %v", got, tt.want)
			}
		})
	}
}
