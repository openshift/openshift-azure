package main

import "testing"

func TestPluginconfigValidate(t *testing.T) {
	tests := []struct {
		name     string
		template *simpleConfig
		wantErr  bool
	}{
		{
			name: "pass",
			template: &simpleConfig{
				Versions: map[string]VersionConfig{
					"v3": VersionConfig{
						ImageVersion: "311.129.20190810",
						Images: map[string]string{
							"alertManager": "registry.access.redhat.com/openshift3/prometheus-alertmanager:v3.11.129",
						},
					},
				},
			},
		},
		{
			name:    "no tag",
			wantErr: true,
			template: &simpleConfig{
				Versions: map[string]VersionConfig{
					"v3": VersionConfig{
						ImageVersion: "311.129.20190810",
						Images: map[string]string{
							"alertManager": "registry.access.redhat.com/openshift3/prometheus-alertmanager",
						},
					},
				},
			},
		},
		{
			name:    "no minor version",
			wantErr: true,
			template: &simpleConfig{
				Versions: map[string]VersionConfig{
					"v3": VersionConfig{
						ImageVersion: "311.129.20190810",
						Images: map[string]string{
							"alertManager": "registry.access.redhat.com/openshift3/prometheus-alertmanager:v3.11",
						},
					},
				},
			},
		},
		{
			name:    "wrong tag version",
			wantErr: true,
			template: &simpleConfig{
				Versions: map[string]VersionConfig{
					"v3": VersionConfig{
						ImageVersion: "311.129.20190810",
						Images: map[string]string{
							"alertManager": "registry.access.redhat.com/openshift3/prometheus-alertmanager:v3.11.130",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validate(tt.template); (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
