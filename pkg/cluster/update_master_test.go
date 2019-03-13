package cluster

import (
	"context"
	"reflect"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster/kubeclient"
	"github.com/openshift/openshift-azure/pkg/cluster/updateblob"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_azureclient"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_cluster"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_kubeclient"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_updateblob"
)

func TestFilterOldVMs(t *testing.T) {
	tests := []struct {
		name   string
		vms    []compute.VirtualMachineScaleSetVM
		blob   *updateblob.UpdateBlob
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
			blob: &updateblob.UpdateBlob{
				HostnameHashes: updateblob.HostnameHashes{
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
			blob: &updateblob.UpdateBlob{
				HostnameHashes: updateblob.HostnameHashes{
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
	tests := []struct {
		name string
		cs   *api.OpenShiftManagedCluster
		vms  []compute.VirtualMachineScaleSetVM
		want *api.PluginError
	}{
		{
			name: "basic coverage",
			cs: &api.OpenShiftManagedCluster{
				Properties: api.Properties{
					AgentPoolProfiles: []api.AgentPoolProfile{
						{
							Role: api.AgentPoolProfileRoleMaster,
						},
					},
					AzProfile: api.AzProfile{
						ResourceGroup: "resourcegroup",
					},
				},
			},
			vms: []compute.VirtualMachineScaleSetVM{
				{
					Name:       to.StringPtr("ss-master_0"),
					InstanceID: to.StringPtr("0"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						OsProfile: &compute.OSProfile{
							ComputerName: to.StringPtr("master-000000"),
						},
					},
				},
				{
					Name:       to.StringPtr("ss-master_1"),
					InstanceID: to.StringPtr("1"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						OsProfile: &compute.OSProfile{
							ComputerName: to.StringPtr("master-000001"),
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gmc := gomock.NewController(t)
			defer gmc.Finish()

			ctx := context.Background()

			ubs := mock_updateblob.NewMockBlobService(gmc)
			vmc := mock_azureclient.NewMockVirtualMachineScaleSetVMsClient(gmc)
			ssc := mock_azureclient.NewMockVirtualMachineScaleSetsClient(gmc)
			kclient := mock_kubeclient.NewMockKubeclient(gmc)
			hasher := mock_cluster.NewMockHasher(gmc)

			u := &simpleUpgrader{
				updateBlobService: ubs,
				vmc:               vmc,
				ssc:               ssc,
				kubeclient:        kclient,
				log:               logrus.NewEntry(logrus.StandardLogger()),
				hasher:            hasher,
			}

			hostnameHashes := map[string][]byte{}

			c := hasher.EXPECT().HashScaleSet(tt.cs, &tt.cs.Properties.AgentPoolProfiles[0]).Return([]byte("updated"), nil)
			c = ubs.EXPECT().Read().Return(updateblob.NewUpdateBlob(), nil).After(c)
			c = vmc.EXPECT().List(ctx, tt.cs.Properties.AzProfile.ResourceGroup, "ss-master", "", "", "").Return(tt.vms, nil).After(c)

			for _, vm := range tt.vms {
				computerName := kubeclient.ComputerName(*vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)

				// 1. drain
				c = kclient.EXPECT().DeleteMaster(computerName).Return(nil).After(c)

				// 2. deallocate
				c = vmc.EXPECT().Deallocate(ctx, tt.cs.Properties.AzProfile.ResourceGroup, "ss-master", *vm.InstanceID).Return(nil).After(c)

				// 3. updateinstances
				c = ssc.EXPECT().UpdateInstances(ctx, tt.cs.Properties.AzProfile.ResourceGroup, "ss-master", compute.VirtualMachineScaleSetVMInstanceRequiredIDs{
					InstanceIds: &[]string{*vm.InstanceID},
				}).Return(nil).After(c)

				// 4. reimage
				c = vmc.EXPECT().Reimage(ctx, tt.cs.Properties.AzProfile.ResourceGroup, "ss-master", *vm.InstanceID, nil).Return(nil).After(c)

				// 5. start
				c = vmc.EXPECT().Start(ctx, tt.cs.Properties.AzProfile.ResourceGroup, "ss-master", *vm.InstanceID).Return(nil).After(c)

				// 6. waitforready
				c = kclient.EXPECT().WaitForReadyMaster(ctx, computerName).Return(nil).After(c)

				// 7. write the updatehash
				hostnameHashes[*vm.Name] = []byte("updated")

				uBlob := updateblob.NewUpdateBlob()
				for k, v := range hostnameHashes {
					uBlob.HostnameHashes[k] = v
				}

				c = ubs.EXPECT().Write(uBlob).Return(nil).After(c)
			}
			if got := u.UpdateMasterAgentPool(ctx, tt.cs, &tt.cs.Properties.AgentPoolProfiles[0]); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("simpleUpgrader.updateInPlace() = %v, want %v", got, tt.want)
			}
		})
	}
}
