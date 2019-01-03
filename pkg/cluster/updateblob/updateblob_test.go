package updateblob

import (
	"bytes"
	"io"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_azureclient/mock_storage"
)

func TestReadUpdateBlob(t *testing.T) {
	tests := []struct {
		name    string
		blob    string
		want    *UpdateBlob
		wantErr error
	}{
		{
			name:    "empty",
			wantErr: io.EOF,
		},
		{
			name: "ok",
			blob: `{"instanceHashes":[{"instanceName":"ss-compute_0","hash":"N3g5OT0="},{"instanceName":"ss-infra_0","hash":"NDU="}]}`,
			want: &UpdateBlob{
				InstanceHashes: InstanceHashes{
					"ss-infra_0":   []byte("45"),
					"ss-compute_0": []byte("7x99="),
				},
				ScalesetHashes: ScalesetHashes{},
			},
		},
	}
	gmc := gomock.NewController(t)
	defer gmc.Finish()
	for _, tt := range tests {
		updateCr := mock_storage.NewMockContainer(gmc)
		updateBlob := mock_storage.NewMockBlob(gmc)
		updateCr.EXPECT().GetBlobReference(UpdateBlobName).Return(updateBlob)
		data := ioutil.NopCloser(strings.NewReader(tt.blob))
		updateBlob.EXPECT().Get(nil).Return(data, nil)
		u := &blobService{
			updateContainer: updateCr,
		}

		got, err := u.Read()
		if (err != nil) != (tt.wantErr != nil) {
			t.Errorf("simpleUpgrader.readUpdateBlob() error = %v, wantErr %v", err, tt.wantErr)
			return
		}
		if tt.wantErr != nil && err != tt.wantErr {
			t.Errorf("simpleUpgrader.readUpdateBlob() error = %v, wantErr %v", err, tt.wantErr)
		}
		if tt.wantErr == nil && !reflect.DeepEqual(got, tt.want) {
			t.Errorf("simpleUpgrader.readUpdateBlob() = %v, want %v", got, tt.want)
		}
	}
}

func TestWriteUpdateBlob(t *testing.T) {
	tests := []struct {
		name    string
		blob    *UpdateBlob
		want    string
		wantErr string
	}{
		{
			name: "empty",
			blob: NewUpdateBlob(),
			want: "{}",
		},
		{
			name: "valid",
			blob: &UpdateBlob{
				InstanceHashes: InstanceHashes{
					"ss-infra_0":   []byte("45"),
					"ss-compute_0": []byte("7x99="),
				},
			},
			want: `{"instanceHashes":[{"instanceName":"ss-compute_0","hash":"N3g5OT0="},{"instanceName":"ss-infra_0","hash":"NDU="}]}`,
		},
	}
	gmc := gomock.NewController(t)
	defer gmc.Finish()
	for _, tt := range tests {
		updateCr := mock_storage.NewMockContainer(gmc)
		updateBlob := mock_storage.NewMockBlob(gmc)
		updateCr.EXPECT().GetBlobReference("update").Return(updateBlob)
		updateBlob.EXPECT().CreateBlockBlobFromReader(bytes.NewReader([]byte(tt.want)), nil)
		u := &blobService{
			updateContainer: updateCr,
		}

		if err := u.Write(tt.blob); (err != nil) != (tt.wantErr != "") {
			t.Errorf("simpleUpgrader.writeUpdateBlob() error = %v, wantErr %v", err, tt.wantErr)
		}
	}
}
