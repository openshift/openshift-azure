package fakerp

import (
	"testing"
)

func TestGetLastUsableIP(t *testing.T) {
	tests := []struct {
		subnet  string
		want    string
		wantErr bool
	}{
		{
			subnet: "10.0.2.0/24",
			want:   "10.0.2.254",
		},
		{
			subnet: "172.0.16.0/28",
			want:   "172.0.16.14",
		},
		{
			subnet: "8.0.2.0/16",
			want:   "8.0.255.254",
		},
	}
	for _, tt := range tests {
		t.Run(tt.subnet, func(t *testing.T) {
			got, err := getLastUsableIP(tt.subnet)
			if (err != nil) != tt.wantErr {
				t.Errorf("getLastUsableIP() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getLastUsableIP() = %v, want %v", got, tt.want)
			}
		})
	}
}
