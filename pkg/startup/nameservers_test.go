package startup

import (
	"reflect"
	"testing"
)

func TestGetServerFromDNSConf(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
		wantErr bool
	}{
		{
			name:    "expected",
			content: "server=168.63.129.16",
			want:    []string{"168.63.129.16"},
		},
		{
			name:    "bad format",
			content: "168.63.129.16",
			wantErr: true,
		},
		{
			name: "complex",
			want: []string{"168.63.129.16", "1.2.3.4"},
			content: `dns-forward-max=5000
server=168.63.129.16
server=1.2.3.4`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getServerFromDNSConf(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("getServerFromDNSConf() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getServerFromDNSConf() = %v, want %v", got, tt.want)
			}
		})
	}
}
