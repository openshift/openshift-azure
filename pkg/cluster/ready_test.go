package cluster

import (
	"context"
	"fmt"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/config"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_azureclient"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_kubeclient"
)

func TestUpgraderWaitForNodes(t *testing.T) {
	vmListErr := fmt.Errorf("vm list failed")
	nodeGetErr := fmt.Errorf("node get failed")
	testRg := "myrg"
	tests := []struct {
		name        string
		expect      map[string][]compute.VirtualMachineScaleSetVM
		wantErr     bool
		expectedErr error
	}{
		{
			name:    "nothing to wait for",
			wantErr: false,
		},
		{
			name:        "list vm error",
			wantErr:     true,
			expectedErr: vmListErr,
		},
		{
			name:        "node get error",
			wantErr:     true,
			expectedErr: nodeGetErr,
			expect: map[string][]compute.VirtualMachineScaleSetVM{
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
			name: "all ready",
			expect: map[string][]compute.VirtualMachineScaleSetVM{
				"master": {
					{
						Name: to.StringPtr("ss-master"),
						VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
							OsProfile: &compute.OSProfile{
								ComputerName: to.StringPtr("master-00000A"),
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
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &api.OpenShiftManagedCluster{
				Properties: api.Properties{
					AzProfile: api.AzProfile{ResourceGroup: testRg},
					AgentPoolProfiles: []api.AgentPoolProfile{
						{
							Name: "foo",
							Role: api.AgentPoolProfileRoleCompute,
						},
						{
							Name: "infra",
							Role: api.AgentPoolProfileRoleInfra,
						},
						{
							Name: "master",
							Role: api.AgentPoolProfileRoleMaster,
						},
					},
				},
			}

			ctx := context.Background()
			gmc := gomock.NewController(t)
			defer gmc.Finish()
			virtualMachineScaleSetsClient := mock_azureclient.NewMockVirtualMachineScaleSetsClient(gmc)
			virtualMachineScaleSetVMsClient := mock_azureclient.NewMockVirtualMachineScaleSetVMsClient(gmc)

			kubeclient := mock_kubeclient.NewMockKubeclient(gmc)
			if tt.wantErr {
				if tt.expectedErr == vmListErr {
					virtualMachineScaleSetVMsClient.EXPECT().List(ctx, testRg, config.GetScalesetName(&api.AgentPoolProfile{Name: "master", Role: api.AgentPoolProfileRoleMaster}, ""), "", "", "").Return(nil, tt.expectedErr)
				} else {
					virtualMachineScaleSetVMsClient.EXPECT().List(ctx, testRg, config.GetScalesetName(&api.AgentPoolProfile{Name: "master", Role: api.AgentPoolProfileRoleMaster}, ""), "", "", "").Return(tt.expect["master"], nil)
					kubeclient.EXPECT().WaitForReadyMaster(ctx, gomock.Any()).Return(nodeGetErr)
				}
			} else {
				virtualMachineScaleSetVMsClient.EXPECT().List(ctx, testRg, config.GetScalesetName(&api.AgentPoolProfile{Name: "master", Role: api.AgentPoolProfileRoleMaster}, ""), "", "", "").Return(tt.expect["master"], nil)
				virtualMachineScaleSetVMsClient.EXPECT().List(ctx, testRg, config.GetScalesetName(&api.AgentPoolProfile{Name: "infra", Role: api.AgentPoolProfileRoleInfra}, ""), "", "", "").Return(tt.expect["infra"], nil)
				virtualMachineScaleSetVMsClient.EXPECT().List(ctx, testRg, config.GetScalesetName(&api.AgentPoolProfile{Name: "foo", Role: api.AgentPoolProfileRoleCompute}, ""), "", "", "").Return(tt.expect["compute"], nil)
				kubeclient.EXPECT().WaitForReadyMaster(ctx, gomock.Any()).Times(len(tt.expect["master"])).Return(nil)
				kubeclient.EXPECT().WaitForReadyWorker(ctx, gomock.Any()).Times(len(tt.expect["infra"]) + len(tt.expect["compute"])).Return(nil)
			}

			u := &simpleUpgrader{
				vmc:        virtualMachineScaleSetVMsClient,
				ssc:        virtualMachineScaleSetsClient,
				kubeclient: kubeclient,
				log:        logrus.NewEntry(logrus.StandardLogger()),
			}
			err := u.waitForNodes(ctx, cs, "")
			if tt.wantErr && tt.expectedErr != err {
				t.Errorf("simpleUpgrader.waitForNodes() wrong error got = %v, expected %v", err, tt.expectedErr)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("simpleUpgrader.waitForNodes() unexpected error = %v", err)
			}
		})
	}
}
