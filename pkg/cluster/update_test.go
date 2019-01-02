package cluster

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
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
	"github.com/openshift/openshift-azure/pkg/config"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_azureclient"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_azureclient/mock_storage"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_cluster"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_kubeclient"
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
		name              string
		cs                *api.OpenShiftManagedCluster
		nodes             map[kubeclient.ComputerName]struct{}
		vmsList           map[string][]compute.VirtualMachineScaleSetVM
		wantHashes        *updateblob
		wantErr           error
		initialUpdateBlob updateblob
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
			name:       "wait for new nodes",
			wantHashes: &updateblob{InstanceHashes: instanceHashMap{"ss-master_0": []byte("hashish")}},
			nodes:      map[kubeclient.ComputerName]struct{}{},
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
			name:              "clear blob of stale instances",
			nodes:             map[kubeclient.ComputerName]struct{}{"master-000000": {}},
			initialUpdateBlob: updateblob{InstanceHashes: instanceHashMap{"ss-master_0": []byte("oldhash"), "ss-master_1": []byte("oldhash")}},
			wantHashes:        &updateblob{InstanceHashes: instanceHashMap{"ss-master_0": []byte("oldhash")}},
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
			updateContainer := mock_storage.NewMockContainer(gmc)
			updateBlob := mock_storage.NewMockBlob(gmc)
			blob, _ := json.Marshal(tt.initialUpdateBlob)
			updateContainer.EXPECT().GetBlobReference("update").Return(updateBlob)
			updateBlob.EXPECT().Get(nil).Return(ioutil.NopCloser(bytes.NewReader(blob)), nil)
			hasher := mock_cluster.NewMockHasher(gmc)
			hasher.EXPECT().HashScaleSet(gomock.Any(), gomock.Any()).Return([]byte("hashish"), nil)

			if len(tt.nodes) < len(tt.vmsList["master"]) {
				client.EXPECT().WaitForReadyWorker(ctx, kubeclient.ComputerName("master-000000")).Return(tt.wantErr)
			}
			if tt.wantErr == nil && (len(tt.nodes) < len(tt.vmsList["master"]) || len(tt.initialUpdateBlob.InstanceHashes) > len(tt.vmsList["master"])) {
				updateContainer.EXPECT().GetBlobReference("update").Return(updateBlob)
				hashData, _ := json.Marshal(tt.wantHashes)
				updateBlob.EXPECT().CreateBlockBlobFromReader(bytes.NewReader([]byte(hashData)), nil)
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
			mockListVMs(ctx, gmc, virtualMachineScaleSetVMsClient, config.GetScalesetName("master"), testRg, tt.vmsList["master"], nil)

			err := u.waitForNewNodes(ctx, tt.cs, tt.nodes)
			if !reflect.DeepEqual(err, tt.wantErr) {
				t.Errorf("simpleUpgrader.waitForNewNodes() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
