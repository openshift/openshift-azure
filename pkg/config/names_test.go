package config

import (
	"testing"
)

func TestGetNames(t *testing.T) {
	tests := []struct {
		instance     int
		wantScaleset string
		wantInstance string
		name         string
	}{
		{
			name:         "master",
			wantScaleset: "ss-master",
			wantInstance: "ss-master_2",
			instance:     2,
		},
		{
			name:         "thingy",
			wantScaleset: "ss-thingy",
			wantInstance: "ss-thingy_3",
			instance:     3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetScalesetName(tt.name); got != tt.wantScaleset {
				t.Errorf("GetScalesetName() = %v, want %v", got, tt.wantScaleset)
			}
			if got := GetInstanceName(tt.name, tt.instance); got != tt.wantInstance {
				t.Errorf("GetInstanceName() = %v, want %v", got, tt.wantInstance)
			}
		})
	}
}
