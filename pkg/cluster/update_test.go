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
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/kubeclient"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_azureclient"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_kubeclient"
)

func TestFilterOldVMs(t *testing.T) {
	tests := []struct {
		name     string
		vms      []compute.VirtualMachineScaleSetVM
		blob     updateblob
		ssHashes map[scalesetName]hash
		exp      []compute.VirtualMachineScaleSetVM
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
			blob: updateblob{
				"ss-master_0": "newhash",
				"ss-master_1": "oldhash",
				"ss-master_2": "oldhash",
			},
			ssHashes: map[scalesetName]hash{
				"ss-master": "newhash",
			},
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
			blob: updateblob{
				"ss-master_0": "newhash",
				"ss-master_1": "newhash",
				"ss-master_2": "newhash",
			},
			ssHashes: map[scalesetName]hash{
				"ss-master": "newhash",
			},
			exp: nil,
		},
	}

	u := &simpleUpgrader{
		log: logrus.NewEntry(logrus.StandardLogger()),
	}
	for _, test := range tests {
		t.Logf("running scenario %q", test.name)
		got := u.filterOldVMs(test.vms, test.blob, test.ssHashes)
		if !reflect.DeepEqual(got, test.exp) {
			t.Errorf("expected vms:\n%#v\ngot:\n%#v", test.exp, got)
		}
	}
}

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
						{Role: api.AgentPoolProfileRoleMaster, Count: 1},
						{Role: api.AgentPoolProfileRoleInfra, Count: 1},
						{Role: api.AgentPoolProfileRoleCompute, Count: 1},
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
						{Role: api.AgentPoolProfileRoleMaster, Count: 1},
						{Role: api.AgentPoolProfileRoleInfra, Count: 1},
						{Role: api.AgentPoolProfileRoleCompute, Count: 1},
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
			mockListVMs(ctx, gmc, virtualMachineScaleSetVMsClient, tt.cs, "master", testRg, tt.vmsBefore["master"], nil)
			mockListVMs(ctx, gmc, virtualMachineScaleSetVMsClient, tt.cs, "infra", testRg, tt.vmsBefore["infra"], nil)
			mockListVMs(ctx, gmc, virtualMachineScaleSetVMsClient, tt.cs, "compute", testRg, tt.vmsBefore["compute"], nil)

			for comp, scalesetName := range tt.expectDrain {
				kubeclient.EXPECT().Drain(ctx, gomock.Any(), comp)
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
