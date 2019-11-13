package version

import "testing"

func TestPluginToDevVersion(t *testing.T) {
	tests := []struct {
		pluginVersion string
		want          string
		wantErr       bool
		previous      int
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
		{
			pluginVersion: "v12.4",
			previous:      1,
			want:          "v123",
		},
		{
			pluginVersion: "v12.4",
			previous:      2,
			want:          "v122",
		},
		{
			pluginVersion: "v12.4",
			previous:      4,
			want:          "v12",
		},
		{
			pluginVersion: "v12.4",
			previous:      5,
			wantErr:       true,
			want:          "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.pluginVersion, func(t *testing.T) {
			got, err := nextDevVersion(tt.pluginVersion, tt.previous)
			if tt.wantErr != (err != nil) {
				t.Errorf("nextDevVersion(%s:%s) = %v, want %v", tt.pluginVersion, tt.want, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("nextDevVersion(%s:%s) = %v, want %v", tt.pluginVersion, tt.want, got, tt.want)
			}
		})
	}
}
