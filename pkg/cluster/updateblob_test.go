package cluster

import (
	"bytes"
	"io"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"

	gomock "github.com/golang/mock/gomock"

	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_azureclient/mock_storage"
)

func TestReadUpdateBlob(t *testing.T) {
	tests := []struct {
		name    string
		want    updateblob
		wantErr error
		blob    string
	}{
		{
			name:    "empty",
			wantErr: io.EOF,
		},
		{
			name: "ok",
			want: updateblob{
				"ss-infra_0":   "45",
				"ss-compute_0": "7x99=",
			},
			blob: `[{"instanceName": "ss-infra_0", "scalesetHash": "45"},{"instanceName":"ss-compute_0","scalesetHash":"7x99="}]`,
		},
	}
	gmc := gomock.NewController(t)
	for _, tt := range tests {
		updateBlob := mock_storage.NewMockBlob(gmc)
		data := ioutil.NopCloser(strings.NewReader(tt.blob))
		updateBlob.EXPECT().Get(nil).Return(data, nil)
		u := &simpleUpgrader{
			updateBlob: updateBlob,
		}

		got, err := u.readUpdateBlob()
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
		b       updateblob
		wantErr string
		blob    string
	}{
		{
			name: "empty",
			blob: "[]",
		},
		{
			name: "valid",
			b: updateblob{
				"ss-infra_0":   "45",
				"ss-compute_0": "7x99=",
			},
			blob: `[{"instanceName":"ss-infra_0","scalesetHash":"45"},{"instanceName":"ss-compute_0","scalesetHash":"7x99="}]`,
		},
	}
	gmc := gomock.NewController(t)
	for _, tt := range tests {
		updateBlob := mock_storage.NewMockBlob(gmc)
		updateBlob.EXPECT().CreateBlockBlobFromReader(bytes.NewReader([]byte(tt.blob)), nil)
		u := &simpleUpgrader{
			updateBlob: updateBlob,
		}

		if err := u.writeUpdateBlob(tt.b); (err != nil) != (tt.wantErr != "") {
			t.Errorf("simpleUpgrader.writeUpdateBlob() error = %v, wantErr %v", err, tt.wantErr)
		}
	}
}
