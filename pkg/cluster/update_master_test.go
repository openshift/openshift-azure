package cluster

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster/updateblob"
	"github.com/openshift/openshift-azure/pkg/config"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_azureclient"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_cluster"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_kubeclient"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_updateblob"
)

func TestUpdateMasterAgentPool(t *testing.T) {
	tests := []struct {
		name string
		cs   *api.OpenShiftManagedCluster
		want *api.PluginError
	}{
		{
			name: "basic coverage",
			cs: &api.OpenShiftManagedCluster{
				Properties: api.Properties{
					AgentPoolProfiles: []api.AgentPoolProfile{
						{
							Name:  "master",
							Count: 2,
							Role:  api.AgentPoolProfileRoleMaster,
						},
					},
					AzProfile: api.AzProfile{
						ResourceGroup: "resourcegroup",
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

			c := ubs.EXPECT().Read().Return(updateblob.NewUpdateBlob(), nil)

			for i := int64(0); i < tt.cs.Properties.AgentPoolProfiles[0].Count; i++ {
				c = hasher.EXPECT().HashMasterScaleSet(tt.cs, &tt.cs.Properties.AgentPoolProfiles[0], i).Return([]byte("updated"), nil).After(c)

				hostname := config.GetHostname(&tt.cs.Properties.AgentPoolProfiles[0], "", i)
				instanceID := fmt.Sprintf("%d", i)

				// 1. drain
				c = kclient.EXPECT().DeleteMaster(hostname).Return(nil).After(c)

				// 2. deallocate
				c = vmc.EXPECT().Deallocate(ctx, tt.cs.Properties.AzProfile.ResourceGroup, "ss-master", instanceID).Return(nil).After(c)

				// 3. updateinstances
				c = ssc.EXPECT().UpdateInstances(ctx, tt.cs.Properties.AzProfile.ResourceGroup, "ss-master", compute.VirtualMachineScaleSetVMInstanceRequiredIDs{
					InstanceIds: &[]string{instanceID},
				}).Return(nil).After(c)

				// 4. reimage
				c = vmc.EXPECT().Reimage(ctx, tt.cs.Properties.AzProfile.ResourceGroup, "ss-master", instanceID, nil).Return(nil).After(c)

				// 5. start
				c = vmc.EXPECT().Start(ctx, tt.cs.Properties.AzProfile.ResourceGroup, "ss-master", instanceID).Return(nil).After(c)

				// 6. waitforready
				c = kclient.EXPECT().WaitForReadyMaster(ctx, hostname).Return(nil).After(c)

				// 7. write the updatehash
				hostnameHashes[hostname] = []byte("updated")

				uBlob := updateblob.NewUpdateBlob()
				for k, v := range hostnameHashes {
					uBlob.HostnameHashes[k] = v
				}
				if i == 0 {
					c = hasher.EXPECT().HashSyncPod(tt.cs).Return([]byte("updated"), nil).After(c)
				}
				uBlob.SyncPodHash = []byte("updated")

				c = ubs.EXPECT().Write(uBlob).Return(nil).After(c)
			}
			if got := u.UpdateMasterAgentPool(ctx, tt.cs, &tt.cs.Properties.AgentPoolProfiles[0]); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("simpleUpgrader.updateInPlace() = %v, want %v", got, tt.want)
			}
		})
	}
}
