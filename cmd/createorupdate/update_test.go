package main

import (
	"reflect"
	"testing"

	"github.com/openshift/openshift-azure/pkg/api"
)

func TestIsScaleUpdate(t *testing.T) {
	tests := []struct {
		name      string
		cs, oldCs *api.OpenShiftManagedCluster
		exp       bool
	}{
		{
			name: "no change",
			cs: &api.OpenShiftManagedCluster{
				Properties: &api.Properties{
					OpenShiftVersion: "v3.10",
					AgentPoolProfiles: []api.AgentPoolProfile{
						{
							Name:  "compute",
							Count: 3,
							Role:  "compute",
						},
					},
				},
			},
			oldCs: &api.OpenShiftManagedCluster{
				Properties: &api.Properties{
					OpenShiftVersion: "v3.10",
					AgentPoolProfiles: []api.AgentPoolProfile{
						{
							Name:  "compute",
							Count: 3,
							Role:  "compute",
						},
					},
				},
			},
			exp: false,
		},
		{
			name: "image change",
			cs: &api.OpenShiftManagedCluster{
				Properties: &api.Properties{
					OpenShiftVersion: "v3.10",
					AgentPoolProfiles: []api.AgentPoolProfile{
						{
							Name:  "compute",
							Count: 3,
							Role:  "compute",
						},
					},
				},
			},
			oldCs: &api.OpenShiftManagedCluster{
				Properties: &api.Properties{
					OpenShiftVersion: "v3.11",
					AgentPoolProfiles: []api.AgentPoolProfile{
						{
							Name:  "compute",
							Count: 3,
							Role:  "compute",
						},
					},
				},
			},
			exp: false,
		},
		{
			name: "scale up",
			cs: &api.OpenShiftManagedCluster{
				Properties: &api.Properties{
					OpenShiftVersion: "v3.10",
					AgentPoolProfiles: []api.AgentPoolProfile{
						{
							Name:  "infra",
							Count: 2,
							Role:  "infra",
						},
						{
							Name:  "compute",
							Count: 3,
							Role:  "compute",
						},
					},
				},
			},
			oldCs: &api.OpenShiftManagedCluster{
				Properties: &api.Properties{
					OpenShiftVersion: "v3.10",
					AgentPoolProfiles: []api.AgentPoolProfile{
						{
							Name:  "infra",
							Count: 1,
							Role:  "infra",
						},
						{
							Name:  "compute",
							Count: 3,
							Role:  "compute",
						},
					},
				},
			},
			exp: true,
		},
		{
			name: "scale down",
			cs: &api.OpenShiftManagedCluster{
				Properties: &api.Properties{
					OpenShiftVersion: "v3.10",
					AgentPoolProfiles: []api.AgentPoolProfile{
						{
							Name:  "infra",
							Count: 1,
							Role:  "infra",
						},
						{
							Name:  "compute",
							Count: 3,
							Role:  "compute",
						},
					},
				},
			},
			oldCs: &api.OpenShiftManagedCluster{
				Properties: &api.Properties{
					OpenShiftVersion: "v3.10",
					AgentPoolProfiles: []api.AgentPoolProfile{
						{
							Name:  "infra",
							Count: 1,
							Role:  "infra",
						},
						{
							Name:  "compute",
							Count: 5,
							Role:  "compute",
						},
					},
				},
			},
			exp: true,
		},
		{
			name: "scale with other changes is not scale",
			cs: &api.OpenShiftManagedCluster{
				Properties: &api.Properties{
					OpenShiftVersion: "v3.10",
					AgentPoolProfiles: []api.AgentPoolProfile{
						{
							Name:  "infra",
							Count: 1,
							Role:  "infra",
						},
						{
							Name:  "compute",
							Count: 3,
							Role:  "compute",
						},
					},
				},
			},
			oldCs: &api.OpenShiftManagedCluster{
				Properties: &api.Properties{
					OpenShiftVersion: "v3.11",
					AgentPoolProfiles: []api.AgentPoolProfile{
						{
							Name:  "infra",
							Count: 1,
							Role:  "infra",
						},
						{
							Name:  "compute",
							Count: 4,
							Role:  "compute",
						},
					},
				},
			},
			exp: false,
		},
	}

	for _, test := range tests {
		got := isScaleUpdate(test.cs, test.oldCs)
		if !reflect.DeepEqual(got, test.exp) {
			t.Errorf("%s: expected scale ops: %#v, got: %#v", test.name, test.exp, got)
		}
	}
}
