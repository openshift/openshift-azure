package cluster

import (
	"context"
	"fmt"
	"testing"

	compute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/config"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_azureclient"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_kubeclient"
)

func mockListVMs(ctx context.Context, gmc *gomock.Controller, virtualMachineScaleSetVMsClient *mock_azureclient.MockVirtualMachineScaleSetVMsClient, cs *api.OpenShiftManagedCluster, role api.AgentPoolProfileRole, rg string, outVMS []compute.VirtualMachineScaleSetVM, outErr error) {
	mPage := mock_azureclient.NewMockVirtualMachineScaleSetVMListResultPage(gmc)
	if len(outVMS) > 0 {
		mPage.EXPECT().Values().Return(outVMS)
		mPage.EXPECT().Next()
	}
	callTimes := func(vms []compute.VirtualMachineScaleSetVM) int {
		if len(vms) > 0 {
			// NotDone gets called twice once for yes, there is data, and once more for no data
			return 2
		}
		// NotDone gets called once for there is no data
		return 1
	}
	if outErr == nil {
		mNotDone := len(outVMS) > 0
		mPage.EXPECT().NotDone().Times(callTimes(outVMS)).DoAndReturn(func() bool {
			ret := mNotDone
			mNotDone = false
			return ret
		})
	}
	scalesetName := config.GetScalesetName(cs, role)
	if outErr != nil {
		virtualMachineScaleSetVMsClient.EXPECT().List(ctx, rg, scalesetName, "", "", "").Return(nil, outErr)
	} else {
		virtualMachineScaleSetVMsClient.EXPECT().List(ctx, rg, scalesetName, "", "", "").Return(mPage, nil)
	}
}

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
					mockListVMs(ctx, gmc, virtualMachineScaleSetVMsClient, cs, "master", testRg, nil, tt.expectedErr)
				} else {
					mockListVMs(ctx, gmc, virtualMachineScaleSetVMsClient, cs, "master", testRg, tt.expect["master"], nil)
					kubeclient.EXPECT().WaitForReady(ctx, api.AgentPoolProfileRoleMaster, gomock.Any()).Return(nodeGetErr)
				}
			} else {
				mockListVMs(ctx, gmc, virtualMachineScaleSetVMsClient, cs, "master", testRg, tt.expect["master"], nil)
				mockListVMs(ctx, gmc, virtualMachineScaleSetVMsClient, cs, "infra", testRg, tt.expect["infra"], nil)
				mockListVMs(ctx, gmc, virtualMachineScaleSetVMsClient, cs, "compute", testRg, tt.expect["compute"], nil)
				kubeclient.EXPECT().WaitForReady(ctx, gomock.Any(), gomock.Any()).Times(len(tt.expect)).Return(nil)
			}

			u := &simpleUpgrader{
				vmc:        virtualMachineScaleSetVMsClient,
				ssc:        virtualMachineScaleSetsClient,
				kubeclient: kubeclient,
				log:        logrus.NewEntry(logrus.StandardLogger()),
			}
			err := u.waitForNodes(ctx, cs)
			if tt.wantErr && tt.expectedErr != err {
				t.Errorf("simpleUpgrader.waitForNodes() wrong error got = %v, expected %v", err, tt.expectedErr)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("simpleUpgrader.waitForNodes() unexpected error = %v", err)
			}
		})
	}
}
