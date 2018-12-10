package updatehash

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster/updateblob"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_azureclient/mock_storage"
)

func TestReload(t *testing.T) {
	tests := []struct {
		name    string
		blob    string
		want    updateblob.Updateblob
		wantErr error
	}{
		{
			name:    "empty",
			wantErr: io.EOF,
		},
		{
			name: "ok",
			blob: `[{"InstanceName":"ss-compute_0","scalesetHash":"7x99="},{"instanceName":"ss-infra_0","scalesetHash":"45"}]`,
			want: updateblob.Updateblob{
				"ss-infra_0":   "45",
				"ss-compute_0": "7x99=",
			},
		},
	}
	gmc := gomock.NewController(t)
	defer gmc.Finish()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateCr := mock_storage.NewMockContainer(gmc)
			updateBlob := mock_storage.NewMockBlob(gmc)
			updateCr.EXPECT().GetBlobReference(updateBlobName).Return(updateBlob)
			data := ioutil.NopCloser(strings.NewReader(tt.blob))
			updateBlob.EXPECT().Get(nil).Return(data, nil)
			u := &updateHash{
				updateContainer: updateCr,
			}

			err := u.Reload()
			if (err != nil) != (tt.wantErr != nil) {
				t.Errorf("UpdateInfo.Reload() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr != nil && err != tt.wantErr {
				t.Errorf("UpdateInfo.Reload() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && !reflect.DeepEqual(u.blob, tt.want) {
				t.Errorf("UpdateInfo.Reload() = %v, want %v", u.blob, tt.want)
			}
		})
	}
}

func TestSave(t *testing.T) {
	tests := []struct {
		name    string
		blob    updateblob.Updateblob
		want    string
		wantErr string
	}{
		{
			name: "empty",
			want: "[]",
		},
		{
			name: "uptodate",
			want: "[]",
		},
		{
			name: "valid",
			blob: updateblob.Updateblob{
				"ss-infra_0":   "45",
				"ss-compute_0": "7x99=",
			},
			want: `[{"instanceName":"ss-compute_0","scalesetHash":"7x99="},{"instanceName":"ss-infra_0","scalesetHash":"45"}]`,
		},
	}
	gmc := gomock.NewController(t)
	defer gmc.Finish()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &updateHash{
				blob: tt.blob,
			}
			updateBlob := mock_storage.NewMockBlob(gmc)
			updateBlob.EXPECT().CreateBlockBlobFromReader(bytes.NewReader([]byte(tt.want)), nil)
			updateCr := mock_storage.NewMockContainer(gmc)
			updateCr.EXPECT().GetBlobReference("update").Return(updateBlob)
			u.updateContainer = updateCr

			if err := u.Save(); (err != nil) != (tt.wantErr != "") {
				t.Errorf("updateHash.Save() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

const masterHash = "RmID82LhPjuQbCEdiVa5cGCVEkdLGD6iU6ozX3vxkD0="

func TestHashScaleSets(t *testing.T) {
	tests := []struct {
		name string
		vmss *compute.VirtualMachineScaleSet
		exp  updateblob.Hash
	}{
		{
			name: "expect a scale set",
			vmss: &compute.VirtualMachineScaleSet{
				Sku:  &compute.Sku{},
				Name: to.StringPtr("ss-master"),
				Type: to.StringPtr("Microsoft.Compute/virtualMachineScaleSets"),
			},
			exp: masterHash,
		},
		{
			name: "hash is invariant with capacity",
			vmss: &compute.VirtualMachineScaleSet{
				Sku: &compute.Sku{
					Capacity: to.Int64Ptr(3),
				},
				Name: to.StringPtr("ss-master"),
				Type: to.StringPtr("Microsoft.Compute/virtualMachineScaleSets"),
			},
			exp: masterHash,
		},
	}

	for _, test := range tests {
		uh := &updateHash{
			log:      logrus.NewEntry(logrus.StandardLogger()),
			ssHashes: map[ScalesetName]updateblob.Hash{},
		}
		err := uh.hashVMSS(test.vmss)
		if err != nil {
			t.Errorf("%s: unexpected error: %v", test.name, err)
		}

		if !reflect.DeepEqual(uh.ssHashes[ScalesetName(*test.vmss.Name)], test.exp) {
			t.Errorf("%s: expected:\n%#v\ngot:\n%#v", test.name, test.exp, uh.ssHashes[ScalesetName(*test.vmss.Name)])
		}
	}
}

func TestFilterOldVMs(t *testing.T) {
	tests := []struct {
		name     string
		vms      []compute.VirtualMachineScaleSetVM
		blob     updateblob.Updateblob
		ssHashes map[ScalesetName]updateblob.Hash
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
			blob: updateblob.Updateblob{
				"ss-master_0": "newhash",
				"ss-master_1": "oldhash",
				"ss-master_2": "oldhash",
			},
			ssHashes: map[ScalesetName]updateblob.Hash{
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
			blob: updateblob.Updateblob{
				"ss-master_0": "newhash",
				"ss-master_1": "newhash",
				"ss-master_2": "newhash",
			},
			ssHashes: map[ScalesetName]updateblob.Hash{
				"ss-master": "newhash",
			},
			exp: nil,
		},
	}

	gmc := gomock.NewController(t)
	defer gmc.Finish()
	for _, test := range tests {
		updateCr := mock_storage.NewMockContainer(gmc)
		u := &updateHash{
			log:             logrus.NewEntry(logrus.StandardLogger()),
			ssHashes:        test.ssHashes,
			updateContainer: updateCr,
		}
		updateBlob := mock_storage.NewMockBlob(gmc)
		updateCr.EXPECT().GetBlobReference(updateBlobName).Return(updateBlob)

		data, err := json.Marshal(test.blob)
		if err != nil {
			t.Error(err)
		}
		updateBlob.EXPECT().Get(nil).Return(ioutil.NopCloser(bytes.NewReader(data)), nil)

		t.Logf("running scenario %q", test.name)
		got, err := u.FilterOldVMs(test.vms)
		if !reflect.DeepEqual(got, test.exp) {
			t.Errorf("expected vms:\n%#v\ngot:\n%#v", test.exp, got)
		}
		if err != nil {
			t.Errorf("expected error %v", err)
		}
	}
}

func TestInitialize(t *testing.T) {
	tests := []struct {
		name     string
		wantBlob updateblob.Updateblob
		cs       *api.OpenShiftManagedCluster
		wantErr  bool
	}{
		{
			name: "coverage",
			wantBlob: updateblob.Updateblob{
				"ss-master_0":  "",
				"ss-master_1":  "",
				"ss-master_2":  "",
				"ss-infra_0":   "",
				"ss-infra_1":   "",
				"ss-compute_0": "",
			},
			cs: &api.OpenShiftManagedCluster{
				Properties: api.Properties{
					AgentPoolProfiles: []api.AgentPoolProfile{
						{
							Name:  "master",
							Role:  api.AgentPoolProfileRoleMaster,
							Count: 3,
						},
						{
							Name:  "infra",
							Role:  api.AgentPoolProfileRoleInfra,
							Count: 2,
						},
						{
							Name:  "compute",
							Role:  api.AgentPoolProfileRoleCompute,
							Count: 1,
						},
					},
				},
			},
		},
	}
	gmc := gomock.NewController(t)
	defer gmc.Finish()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateCr := mock_storage.NewMockContainer(gmc)
			updateCr.EXPECT().CreateIfNotExists(nil).Return(true, nil)

			updateBlob := mock_storage.NewMockBlob(gmc)
			updateBlob.EXPECT().CreateBlockBlobFromReader(gomock.Any(), nil)
			updateCr.EXPECT().GetBlobReference("update").Return(updateBlob)

			i := &updateHash{
				updateContainer: updateCr,
				blob:            updateblob.Updateblob{},
				log:             logrus.NewEntry(logrus.StandardLogger()),
			}
			if err := i.Initialize(tt.cs); (err != nil) != tt.wantErr {
				t.Errorf("updateHash.Initialize() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(i.blob, tt.wantBlob) {
				t.Errorf("expected blob:\n%#v\ngot:\n%#v", tt.wantBlob, i.blob)
			}
		})
	}
}

func TestDeleteAllBut(t *testing.T) {
	tests := []struct {
		name        string
		existingVMs map[updateblob.InstanceName]struct{}
		beforeBlob  updateblob.Updateblob
		afterBlob   updateblob.Updateblob
	}{
		{
			name: "delete none",
			beforeBlob: updateblob.Updateblob{
				"ss-compute_0": "",
				"ss-compute_1": "",
				"ss-compute_2": "",
			},
			afterBlob: updateblob.Updateblob{
				"ss-compute_0": "",
				"ss-compute_1": "",
				"ss-compute_2": "",
			},
			existingVMs: map[updateblob.InstanceName]struct{}{
				"ss-compute_0": {},
				"ss-compute_1": {},
				"ss-compute_2": {},
			},
		},
		{
			name: "delete one",
			beforeBlob: updateblob.Updateblob{
				"ss-compute_0": "",
				"ss-compute_1": "",
				"ss-compute_2": "",
			},
			afterBlob: updateblob.Updateblob{
				"ss-compute_0": "",
				"ss-compute_2": "",
			},
			existingVMs: map[updateblob.InstanceName]struct{}{
				"ss-compute_0": {},
				"ss-compute_2": {},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &updateHash{
				blob: tt.beforeBlob,
			}
			gmc := gomock.NewController(t)
			defer gmc.Finish()
			updateCr := mock_storage.NewMockContainer(gmc)
			if len(tt.beforeBlob) > len(tt.existingVMs) {
				updateBlob := mock_storage.NewMockBlob(gmc)
				updateCr.EXPECT().GetBlobReference("update").Return(updateBlob)
				updateBlob.EXPECT().CreateBlockBlobFromReader(gomock.Any(), nil)
			}
			i.updateContainer = updateCr
			i.DeleteAllBut(tt.existingVMs)
			if !reflect.DeepEqual(i.blob, tt.afterBlob) {
				t.Errorf("expected blob:\n%#v\ngot:\n%#v", tt.afterBlob, i.blob)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	gmc := gomock.NewController(t)
	defer gmc.Finish()
	updateBlob := mock_storage.NewMockBlob(gmc)
	updateCr := mock_storage.NewMockContainer(gmc)
	updateCr.EXPECT().GetBlobReference("update").Return(updateBlob)
	updateBlob.EXPECT().Delete(nil).Return(nil)

	i := &updateHash{
		updateContainer: updateCr,
	}
	if err := i.Delete(); err != nil {
		t.Errorf("updateHash.Delete() error = %v", err)
	}
}
