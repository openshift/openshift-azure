package validate

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	api "github.com/Azure/acs-engine/pkg/api"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name           string
		cs             *api.ContainerService
		expectedResult error
	}{
		{
			name:           "empty",
			cs:             &api.ContainerService{},
			expectedResult: fmt.Errorf("malformed manifest"),
		},
		{
			name: "invalid openshift version",
			cs: &api.ContainerService{
				Properties: &api.Properties{
					OrchestratorProfile: &api.OrchestratorProfile{
						OpenShiftConfig: &api.OpenShiftConfig{},
					},
					ServicePrincipalProfile: &api.ServicePrincipalProfile{},
				},
			},
			expectedResult: fmt.Errorf("invalid openShiftVersion \"\""),
		},
		{
			name: "invalid number of agentPoolProfiles",
			cs: &api.ContainerService{
				Properties: &api.Properties{
					OrchestratorProfile: &api.OrchestratorProfile{
						OpenShiftConfig: &api.OpenShiftConfig{
							OpenShiftVersion: "v3.10",
						},
					},
					ServicePrincipalProfile: &api.ServicePrincipalProfile{},
					AgentPoolProfiles:       nil,
				},
			},
			expectedResult: fmt.Errorf("invalid number of agentPoolProfiles"),
		},
		{
			name: "invalid number of agentPoolProfiles",
			cs: &api.ContainerService{
				Properties: &api.Properties{
					OrchestratorProfile: &api.OrchestratorProfile{
						OpenShiftConfig: &api.OpenShiftConfig{
							OpenShiftVersion: "v3.10",
						},
					},
					ServicePrincipalProfile: &api.ServicePrincipalProfile{},
					AgentPoolProfiles: []*api.AgentPoolProfile{
						{
							VnetSubnetID: "vnet1",
						},
						{
							VnetSubnetID: "vnet2",
						},
						{
							VnetSubnetID: "vnet1",
						},
					},
				},
			},
			expectedResult: fmt.Errorf("non-identical vnetSubnetIDs"),
		},
	}
	for _, tt := range tests {
		err := Validate(tt.cs, tt.cs)
		if !reflect.DeepEqual(err, tt.expectedResult) {
			t.Errorf("fail: %#v \n%#v\n%#v", tt.name, err, tt.expectedResult)
		}
	}

}

func TestValidateDevelopmentSwitches(t *testing.T) {
	tests := []struct {
		name           string
		deployOS       string
		expectedResult error
	}{
		{
			name:           "rhel",
			deployOS:       "",
			expectedResult: nil,
		},
		{
			name:           "centos7",
			deployOS:       "centos7",
			expectedResult: nil,
		},
		{
			name:           "not supported",
			deployOS:       "ubuntu",
			expectedResult: fmt.Errorf("invalid DEPLOY_OS \"ubuntu\""),
		},
	}

	for _, tt := range tests {
		os.Setenv("DEPLOY_OS", tt.deployOS)
		err := validateDevelopmentSwitches()
		if !reflect.DeepEqual(err, tt.expectedResult) {
			t.Errorf("fail: %#v \n%#v\n%#v", tt.name, err, tt.expectedResult)
		}
	}

}
