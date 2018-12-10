package updateblob

import (
	"reflect"
	"testing"
)

func TestMarshalling(t *testing.T) {
	tests := []struct {
		name string
		blob string
		want Updateblob
	}{
		{
			name: "empty",
			blob: `[]`,
			want: Updateblob{},
		},
		{
			name: "one",
			blob: `[{"instanceName":"ss-compute_0","scalesetHash":"7x99="}]`,
			want: Updateblob{
				"ss-compute_0": "7x99=",
			},
		},
		{
			name: "two",
			blob: `[{"instanceName":"ss-compute_0","scalesetHash":"7x99="},{"instanceName":"ss-infra_0","scalesetHash":"45"}]`,
			want: Updateblob{
				"ss-infra_0":   "45",
				"ss-compute_0": "7x99=",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blob := Updateblob{}
			blobBytes := []byte(tt.blob)
			err := blob.UnmarshalJSON(blobBytes)
			if err != nil {
				t.Fatalf("Updateblob.UnmarshalJSON() error = %v", err)
			}
			if !reflect.DeepEqual(blob, tt.want) {
				t.Fatalf("Updateblob.UnmarshalJSON() = %v", blob)
			}
			got, err := blob.MarshalJSON()
			if err != nil {
				t.Fatalf("Updateblob.MarshalJSON() error = %v", err)
			}
			if !reflect.DeepEqual(got, blobBytes) {
				t.Fatalf("Updateblob.MarshalJSON() = got %v, want %v", string(got), string(tt.blob))
			}
		})
	}
}
