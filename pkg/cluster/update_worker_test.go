package cluster

import (
	"context"
	"crypto/rsa"
	"reflect"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster/updateblob"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_azureclient"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_cluster"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_kubeclient"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_scaler"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_updateblob"
)

func TestUpdateWorkerAgentPool(t *testing.T) {
	tests := []struct {
		name   string
		cs     *api.OpenShiftManagedCluster
		vms    []compute.VirtualMachineScaleSetVM
		suffix string
		want   *api.PluginError
	}{
		{
			name:   "basic coverage",
			suffix: "foo",
			cs: &api.OpenShiftManagedCluster{
				Config: api.Config{
					SSHKey: &rsa.PrivateKey{},
				},
				Properties: api.Properties{
					AgentPoolProfiles: []api.AgentPoolProfile{
						{
							Name:  "compute",
							Role:  api.AgentPoolProfileRoleCompute,
							Count: 2,
						},
					},
					AzProfile: api.AzProfile{
						ResourceGroup: "resourcegroup",
					},
				},
			},
			vms: []compute.VirtualMachineScaleSetVM{
				{
					Name:       to.StringPtr("ss-compute_0"),
					InstanceID: to.StringPtr("0"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						OsProfile: &compute.OSProfile{
							ComputerName: to.StringPtr("compute-000000"),
						},
					},
				},
				{
					Name:       to.StringPtr("ss-compute_1"),
					InstanceID: to.StringPtr("1"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						OsProfile: &compute.OSProfile{
							ComputerName: to.StringPtr("compute-000001"),
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
			log := logrus.NewEntry(logrus.StandardLogger())
			ubs := mock_updateblob.NewMockBlobService(gmc)
			vmc := mock_azureclient.NewMockVirtualMachineScaleSetVMsClient(gmc)
			ssc := mock_azureclient.NewMockVirtualMachineScaleSetsClient(gmc)
			kc := mock_kubeclient.NewMockInterface(gmc)
			hasher := mock_cluster.NewMockHasher(gmc)
			sss := []compute.VirtualMachineScaleSet{
				{
					Name: to.StringPtr("ss-compute-1"),
					Sku:  &compute.Sku{Capacity: to.Int64Ptr(2)},
				},
				{
					Name: to.StringPtr("ss-compute-2"),
					Sku:  &compute.Sku{Capacity: to.Int64Ptr(2)},
				},
			}

			targetScaler := mock_scaler.NewMockScaler(gmc)
			sourceScaler := mock_scaler.NewMockScaler(gmc)
			scalerFactory := mock_scaler.NewMockFactory(gmc)

			c := hasher.EXPECT().HashScaleSet(tt.cs, &tt.cs.Properties.AgentPoolProfiles[0]).Return([]byte("updated"), nil)
			ublob := updateblob.NewUpdateBlob()
			ublob.ScalesetHashes[*sss[0].Name] = []byte("updated")
			c = ubs.EXPECT().Read().Return(ublob, nil).After(c)
			c = ssc.EXPECT().List(ctx, tt.cs.Properties.AzProfile.ResourceGroup).Return(sss, nil).After(c)
			c = scalerFactory.EXPECT().New(log, ssc, vmc, kc, tt.cs.Properties.AzProfile.ResourceGroup, &sss[0]).Return(targetScaler).After(c)
			c = scalerFactory.EXPECT().New(log, ssc, vmc, kc, tt.cs.Properties.AzProfile.ResourceGroup, &sss[1]).Return(sourceScaler).After(c)
			// scaling
			c = sourceScaler.EXPECT().Scale(ctx, int64(1)).After(c).DoAndReturn(
				func(ctx context.Context, count int64) *api.PluginError {
					sss[1].Sku.Capacity = to.Int64Ptr(1) // because, it's mocked this does not get changed.
					return nil
				})
			c = sourceScaler.EXPECT().Scale(ctx, int64(0)).After(c).DoAndReturn(
				func(ctx context.Context, count int64) *api.PluginError {
					sss[1].Sku.Capacity = to.Int64Ptr(0) // because, it's mocked this does not get changed.
					return nil
				})
			// delete the old scaleset
			c = ubs.EXPECT().Write(ublob).Return(nil)
			c = ssc.EXPECT().Delete(ctx, tt.cs.Properties.AzProfile.ResourceGroup, *sss[1].Name)
			// last scale
			c = targetScaler.EXPECT().Scale(ctx, int64(2)).After(c)

			u := &Upgrade{
				UpdateBlobService: ubs,
				ScalerFactory:     scalerFactory,
				Vmc:               vmc,
				Ssc:               ssc,
				Interface:         kc,
				Log:               log,
				Hasher:            hasher,
				Cs:                tt.cs,
			}
			if got := u.UpdateWorkerAgentPool(ctx, &tt.cs.Properties.AgentPoolProfiles[0], tt.suffix); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Upgrade.UpdateWorkerAgentPool() = %v, want %v", got, tt.want)
			}
		})
	}
}
