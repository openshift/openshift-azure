package main

import "testing"

func TestPluginToDevVersion(t *testing.T) {
	tests := []struct {
		pluginVersion string
		want          string
	}{
		{
			pluginVersion: "v9.0",
			want:          "v9",
		},
		{
			pluginVersion: "v9.2",
			want:          "v92",
		},
		{
			pluginVersion: "v11.0",
			want:          "v11",
		},
		{
			pluginVersion: "v12.4",
			want:          "v124",
		},
	}
	for _, tt := range tests {
		t.Run(tt.pluginVersion, func(t *testing.T) {
			if got := pluginToDevVersion(tt.pluginVersion); got != tt.want {
				t.Errorf("pluginToDevVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}
