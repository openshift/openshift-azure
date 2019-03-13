package scaler

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_azureclient"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_kubeclient"
)

func TestScaleUp(t *testing.T) {
	testRg := "testRg"
	testSS := "ss-compute"
	tests := []struct {
		name      string
		vmsBefore []compute.VirtualMachineScaleSetVM
		vmsAfter  []compute.VirtualMachineScaleSetVM
		count     int64
		want      *api.PluginError
	}{
		{
			name:  "no change",
			count: 1,
			vmsBefore: []compute.VirtualMachineScaleSetVM{
				{
					Name:       to.StringPtr("ss-compute_0"),
					InstanceID: to.StringPtr("0"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						OsProfile: &compute.OSProfile{
							ComputerName: to.StringPtr("compute-000000"),
						},
					},
				},
			},
			want: nil,
		},
		{
			name:  "up by one",
			count: 2,
			want:  nil,
			vmsBefore: []compute.VirtualMachineScaleSetVM{
				{
					Name:       to.StringPtr("ss-compute_0"),
					InstanceID: to.StringPtr("0"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						OsProfile: &compute.OSProfile{
							ComputerName: to.StringPtr("compute-000000"),
						},
					},
				},
			},
			vmsAfter: []compute.VirtualMachineScaleSetVM{
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
			ctx := context.Background()
			gmc := gomock.NewController(t)
			defer gmc.Finish()
			kc := mock_kubeclient.NewMockKubeclient(gmc)
			ssc := mock_azureclient.NewMockVirtualMachineScaleSetsClient(gmc)
			vmc := mock_azureclient.NewMockVirtualMachineScaleSetVMsClient(gmc)

			if len(tt.vmsBefore) != int(tt.count) {
				// initial listing
				c := vmc.EXPECT().List(ctx, testRg, testSS, "", "", "").Return(tt.vmsBefore, nil)
				// update
				c = ssc.EXPECT().Update(ctx, testRg, testSS, compute.VirtualMachineScaleSetUpdate{
					Sku: &compute.Sku{
						Capacity: to.Int64Ptr(tt.count),
					},
				}).After(c)
				// list with new vms
				c = vmc.EXPECT().List(ctx, testRg, testSS, "", "", "").Return(tt.vmsAfter, nil).After(c)

				for i, vm := range tt.vmsAfter {
					if i >= len(tt.vmsBefore) {
						hostname := strings.ToLower(*vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
						c = kc.EXPECT().WaitForReadyWorker(ctx, hostname).Return(nil).After(c)
					}
				}
			}
			ws := &workerScaler{
				vmc: vmc,
				ssc: ssc,
				ss: &compute.VirtualMachineScaleSet{
					Name: &testSS,
					Sku:  &compute.Sku{Capacity: to.Int64Ptr(int64(len(tt.vmsBefore)))}},
				kubeclient:    kc,
				log:           logrus.NewEntry(logrus.StandardLogger()).WithField("test", t.Name),
				resourceGroup: testRg,
			}
			if got := ws.Scale(ctx, tt.count); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("workerScaler.Scale() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScaleDown(t *testing.T) {
	testRg := "testRg"
	testSS := "ss-compute"
	tests := []struct {
		name      string
		vmsBefore []compute.VirtualMachineScaleSetVM
		vmsAfter  []compute.VirtualMachineScaleSetVM
		count     int64
		want      *api.PluginError
	}{
		{
			name:  "no change",
			count: 1,
			vmsBefore: []compute.VirtualMachineScaleSetVM{
				{
					Name:       to.StringPtr("ss-compute_0"),
					InstanceID: to.StringPtr("0"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						OsProfile: &compute.OSProfile{
							ComputerName: to.StringPtr("compute-000000"),
						},
					},
				},
			},
			vmsAfter: []compute.VirtualMachineScaleSetVM{
				{
					Name:       to.StringPtr("ss-compute_0"),
					InstanceID: to.StringPtr("0"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						OsProfile: &compute.OSProfile{
							ComputerName: to.StringPtr("compute-000000"),
						},
					},
				},
			},
			want: nil,
		},
		{
			name:  "down by one",
			count: 1,
			want:  nil,
			vmsAfter: []compute.VirtualMachineScaleSetVM{
				{
					Name:       to.StringPtr("ss-compute_0"),
					InstanceID: to.StringPtr("0"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						OsProfile: &compute.OSProfile{
							ComputerName: to.StringPtr("compute-000000"),
						},
					},
				},
			},
			vmsBefore: []compute.VirtualMachineScaleSetVM{
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
			ctx := context.Background()
			gmc := gomock.NewController(t)
			defer gmc.Finish()
			vmc := mock_azureclient.NewMockVirtualMachineScaleSetVMsClient(gmc)
			ssc := mock_azureclient.NewMockVirtualMachineScaleSetsClient(gmc)
			kc := mock_kubeclient.NewMockKubeclient(gmc)

			if len(tt.vmsBefore) != int(tt.count) {
				// initial listing
				c := vmc.EXPECT().List(ctx, testRg, testSS, "", "", "").Return(tt.vmsBefore, nil)

				for i, vm := range tt.vmsBefore {
					if i >= len(tt.vmsAfter) {
						hostname := strings.ToLower(*vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
						c = kc.EXPECT().DrainAndDeleteWorker(ctx, hostname).Return(nil).After(c)
						c = vmc.EXPECT().Delete(ctx, testRg, testSS, *vm.InstanceID).Return(nil).After(c)
					}
				}
			}
			ws := &workerScaler{
				vmc: vmc,
				ssc: ssc,
				ss: &compute.VirtualMachineScaleSet{
					Name: &testSS,
					Sku:  &compute.Sku{Capacity: to.Int64Ptr(int64(len(tt.vmsBefore)))}},
				kubeclient:    kc,
				log:           logrus.NewEntry(logrus.StandardLogger()),
				resourceGroup: testRg,
			}
			if got := ws.Scale(ctx, tt.count); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("workerScaler.ScaleDown() = %v, want %v", got, tt.want)
			}
		})
	}
}
