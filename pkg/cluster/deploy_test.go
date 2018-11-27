package cluster

import (
	"bytes"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"

	gomock "github.com/golang/mock/gomock"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_azureclient/mock_storage"
)

const masterHash = "RmID82LhPjuQbCEdiVa5cGCVEkdLGD6iU6ozX3vxkD0="

func TestHashScaleSets(t *testing.T) {
	tests := []struct {
		name string
		t    map[string]interface{}
		exp  map[scalesetName]hash
	}{
		{
			name: "expect a scale set",
			t: map[string]interface{}{
				"schema": "schemaversion",
				"resources": []interface{}{
					map[string]interface{}{
						"type": "Microsoft.Compute/virtualMachineScaleSets",
						"dependsOn": []interface{}{
							"[resourceId('Microsoft.Network/virtualNetworks', 'vnet')]",
							"[resourceId('Microsoft.Network/networkSecurityGroups', 'nsg-master')]",
						},
						"sku": map[string]interface{}{
							"capacity": "3",
						},
						"name": "ss-master",
					},
					map[string]interface{}{
						"type": "Microsoft.Storage/storageAccounts",
						"name": "dsdgskjgjner",
					},
				},
			},
			exp: map[scalesetName]hash{
				"ss-master": masterHash,
			},
		},
		{
			name: "expect three scale sets",
			t: map[string]interface{}{
				"schema": "schemaversion",
				"resources": []interface{}{
					map[string]interface{}{

						"type": "Microsoft.Compute/virtualMachineScaleSets",
						"dependsOn": []interface{}{
							"[resourceId('Microsoft.Network/virtualNetworks', 'vnet')]",
						},
						"sku": map[string]interface{}{
							"capacity": "2",
						},
						"name": "ss-master",
					},
					map[string]interface{}{
						"type": "Microsoft.Compute/virtualMachineScaleSets",
						"sku": map[string]interface{}{
							"capacity": "2",
						},
						"name": "ss-infra",
					},
					map[string]interface{}{
						"type": "Microsoft.Compute/virtualMachineScaleSets",
						"sku": map[string]interface{}{
							"capacity": "1",
						},
						"name": "ss-compute",
					},
					map[string]interface{}{
						"type": "Microsoft.Storage/storageAccounts",
						"name": "dsdgskjgjner",
					},
				},
			},
			exp: map[scalesetName]hash{
				"ss-master":  masterHash,
				"ss-infra":   "aqOO0n4n/nx5onYVUEwoW3s/GCnFoEZIZBowvhaHD6c=",
				"ss-compute": "iWDo277FXQHmvzHj5z1l4o+L/hoRvVSzTGroojwA2ZU=",
			},
		},
	}

	for _, test := range tests {
		got, err := hashScaleSets(test.t)
		if err != nil {
			t.Errorf("%s: unexpected error: %v", test.name, err)
		}
		if !reflect.DeepEqual(got, test.exp) {
			t.Errorf("%s: expected:\n%#v\ngot:\n%#v", test.name, test.exp, got)
		}
	}
}

func TestReadUpdateBlob(t *testing.T) {
	tests := []struct {
		name    string
		want    map[instanceName]hash
		wantErr string
		blob    string
	}{
		{
			name:    "empty",
			want:    map[instanceName]hash{},
			blob:    "",
			wantErr: "unexpected end of JSON input",
		},
		{
			name: "ok",
			want: map[instanceName]hash{
				"ss-infra_0":   "45",
				"ss-compute_0": "7x99=",
			},
			blob: "[{\"instanceName\": \"ss-infra_0\", \"scalesetHash\": \"45\"},{\"instanceName\":\"ss-compute_0\",\"scalesetHash\":\"7x99=\"}]",
		},
	}
	gmc := gomock.NewController(t)
	storageClient := mock_storage.NewMockClient(gmc)
	u := &simpleUpgrader{
		pluginConfig:  api.PluginConfig{},
		storageClient: storageClient,
	}

	for _, tt := range tests {
		bsa := mock_storage.NewMockBlobStorageClient(gmc)
		storageClient.EXPECT().GetBlobService().Return(bsa)
		updateCr := mock_storage.NewMockContainer(gmc)
		bsa.EXPECT().GetContainerReference(updateContainerName).Return(updateCr)
		updateBlob := mock_storage.NewMockBlob(gmc)
		updateCr.EXPECT().GetBlobReference(updateBlobName).Return(updateBlob)
		data := ioutil.NopCloser(strings.NewReader(tt.blob))
		updateBlob.EXPECT().Get(nil).Return(data, nil)

		got, err := u.readUpdateBlob()
		if (err != nil) != (len(tt.wantErr) > 0) {
			t.Errorf("simpleUpgrader.readUpdateBlob() error = %v, wantErr %v", err, tt.wantErr)
			return
		}
		if len(tt.wantErr) > 0 && !strings.Contains(err.Error(), tt.wantErr) {
			t.Errorf("simpleUpgrader.readUpdateBlob() error = %v, wantErr %v", err, tt.wantErr)
		}
		if !reflect.DeepEqual(got, tt.want) && len(tt.wantErr) == 0 {
			t.Errorf("simpleUpgrader.readUpdateBlob() = %v, want %v", got, tt.want)
		}
	}
}

func TestWriteUpdateBlob(t *testing.T) {
	tests := []struct {
		name    string
		b       map[instanceName]hash
		wantErr string
		blob    string
	}{
		{
			name: "empty",
			b:    map[instanceName]hash{},
			blob: "[]",
		},
		{
			name: "valid",
			b: map[instanceName]hash{
				"ss-infra_0":   "45",
				"ss-compute_0": "7x99=",
			},
			blob: "[{\"instanceName\":\"ss-infra_0\",\"scalesetHash\":\"45\"},{\"instanceName\":\"ss-compute_0\",\"scalesetHash\":\"7x99=\"}]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gmc := gomock.NewController(t)
			storageClient := mock_storage.NewMockClient(gmc)

			bsa := mock_storage.NewMockBlobStorageClient(gmc)
			storageClient.EXPECT().GetBlobService().Return(bsa)
			updateCr := mock_storage.NewMockContainer(gmc)
			bsa.EXPECT().GetContainerReference(updateContainerName).Return(updateCr)
			updateBlob := mock_storage.NewMockBlob(gmc)
			updateCr.EXPECT().GetBlobReference(updateBlobName).Return(updateBlob)
			updateBlob.EXPECT().CreateBlockBlobFromReader(bytes.NewReader([]byte(tt.blob)), nil)
			u := &simpleUpgrader{
				pluginConfig:  api.PluginConfig{},
				storageClient: storageClient,
			}
			if err := u.writeUpdateBlob(tt.b); (err != nil) != (tt.wantErr != "") {
				t.Errorf("simpleUpgrader.writeUpdateBlob() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
