package configblob

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestStorageKeyMarshal(t *testing.T) {
	tests := []struct {
		Name    string
		Key     string
		want    string
		wantErr bool
	}{
		{
			Name: "foo",
			Key:  "thisisit",
			want: "{\"name\":\"foo\",\"key\":\"thisisit\"}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			s := &StorageKey{
				Name: tt.Name,
				Key:  tt.Key,
			}
			got, err := json.Marshal(s)
			if (err != nil) != tt.wantErr {
				t.Errorf("StorageKey.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(string(got), tt.want) {
				t.Errorf("StorageKey.MarshalJSON() = %v, want %v", string(got), tt.want)
			}
		})
	}
}
