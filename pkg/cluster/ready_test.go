package cluster

import (
	"context"
	"fmt"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster/names"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_azureclient"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_kubeclient"
)

func TestWaitForNodesInAgentPoolProfile(t *testing.T) {
	vmListErr := fmt.Errorf("vm list failed")
	nodeGetErr := fmt.Errorf("node get failed")
	testRg := "myrg"
	tests := []struct {
		name        string
		expect      []compute.VirtualMachineScaleSetVM
		appIndex    int
		wantErr     bool
		expectedErr error
	}{
		{
			name:     "nothing to wait for",
			appIndex: 1,
			wantErr:  false,
		},
		{
			name:        "list vm error",
			appIndex:    1,
			wantErr:     true,
			expectedErr: vmListErr,
		},
		{
			name:        "node get error",
			wantErr:     true,
			appIndex:    0,
			expectedErr: nodeGetErr,
			expect: []compute.VirtualMachineScaleSetVM{
				{
					Name: to.StringPtr("ss-master"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						OsProfile: &compute.OSProfile{
							ComputerName: to.StringPtr("master-000000"),
						},
					},
				},
			},
		},
		{
			name:     "masters ready",
			appIndex: 0,
			expect: []compute.VirtualMachineScaleSetVM{
				{
					Name: to.StringPtr("ss-master"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						OsProfile: &compute.OSProfile{
							ComputerName: to.StringPtr("master-00000A"),
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:     "infra ready",
			appIndex: 1,
			expect: []compute.VirtualMachineScaleSetVM{
				{
					Name: to.StringPtr("ss-infra"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						OsProfile: &compute.OSProfile{
							ComputerName: to.StringPtr("infra-00000A"),
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:     "computes ready",
			appIndex: 2,
			expect: []compute.VirtualMachineScaleSetVM{
				{
					Name: to.StringPtr("ss-compute"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						OsProfile: &compute.OSProfile{
							ComputerName: to.StringPtr("compute-00000A"),
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
							Name: "master",
							Role: api.AgentPoolProfileRoleMaster,
						},
						{
							Name: "infra",
							Role: api.AgentPoolProfileRoleInfra,
						},
						{
							Name: "foo",
							Role: api.AgentPoolProfileRoleCompute,
						},
					},
				},
			}

			ctx := context.Background()
			gmc := gomock.NewController(t)
			defer gmc.Finish()
			ssc := mock_azureclient.NewMockVirtualMachineScaleSetsClient(gmc)
			vmc := mock_azureclient.NewMockVirtualMachineScaleSetVMsClient(gmc)

			kc := mock_kubeclient.NewMockInterface(gmc)
			if tt.wantErr {
				if tt.expectedErr == vmListErr {
					vmc.EXPECT().List(ctx, testRg, names.GetScalesetName(&cs.Properties.AgentPoolProfiles[tt.appIndex], ""), "", "", "").Return(nil, tt.expectedErr)
				} else {
					vmc.EXPECT().List(ctx, testRg, names.GetScalesetName(&cs.Properties.AgentPoolProfiles[tt.appIndex], ""), "", "", "").Return(tt.expect, nil)
					if cs.Properties.AgentPoolProfiles[tt.appIndex].Role == api.AgentPoolProfileRoleMaster {
						kc.EXPECT().WaitForReadyMaster(ctx, gomock.Any()).Return(nodeGetErr)
					} else {
						kc.EXPECT().WaitForReadyWorker(ctx, gomock.Any()).Return(nodeGetErr)
					}
				}
			} else {
				vmc.EXPECT().List(ctx, testRg, names.GetScalesetName(&cs.Properties.AgentPoolProfiles[tt.appIndex], ""), "", "", "").Return(tt.expect, nil)
				if cs.Properties.AgentPoolProfiles[tt.appIndex].Role == api.AgentPoolProfileRoleMaster {
					kc.EXPECT().WaitForReadyMaster(ctx, gomock.Any()).Times(len(tt.expect)).Return(nil)
				} else {
					kc.EXPECT().WaitForReadyWorker(ctx, gomock.Any()).Times(len(tt.expect)).Return(nil)
				}
			}

			u := &SimpleUpgrader{
				Vmc:       vmc,
				Ssc:       ssc,
				Interface: kc,
				Log:       logrus.NewEntry(logrus.StandardLogger()),
				Cs:        cs,
			}
			err := u.WaitForNodesInAgentPoolProfile(ctx, &cs.Properties.AgentPoolProfiles[tt.appIndex], "")
			if tt.wantErr && tt.expectedErr != err {
				t.Errorf("SimpleUpgrader.waitForNodes() wrong error got = %v, expected %v", err, tt.expectedErr)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("SimpleUpgrader.waitForNodes() unexpected error = %v", err)
			}
		})
	}
}
