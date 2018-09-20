package arm

import (
	"context"
	"strings"
	"testing"

	acsapi "github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/config"
	"github.com/openshift/openshift-azure/pkg/util/fixtures"
	"github.com/sirupsen/logrus"
)

func Test_simpleGenerator_Generate(t *testing.T) {
	tests := []struct {
		name              string
		cs                *acsapi.OpenShiftManagedCluster
		isUpdate          bool
		want              []byte
		wantErr           bool
		wantResourceTypes []string
	}{
		{
			name:     "good-create",
			cs:       fixtures.NewTestOpenShiftCluster(),
			isUpdate: false,
			wantErr:  false,
			wantResourceTypes: []string{
				"\"type\": \"Microsoft.OperationalInsights/workspaces\"",
				"\"type\": \"Microsoft.Network/virtualNetworks\"",
				"\"type\": \"Microsoft.Network/publicIPAddresses\"",
				"\"type\": \"Microsoft.Network/loadBalancers\"",
				"\"type\": \"Microsoft.Storage/storageAccounts\"",
				"\"type\": \"Microsoft.Compute/virtualMachineScaleSets\"",
				"\"type\": \"Microsoft.Network/networkSecurityGroups\"",
			},
		},
		{
			name:     "good-update",
			cs:       fixtures.NewTestOpenShiftCluster(),
			isUpdate: true,
			wantErr:  false,
			wantResourceTypes: []string{
				"\"type\": \"Microsoft.OperationalInsights/workspaces\"",
				"\"type\": \"Microsoft.Network/virtualNetworks\"",
				"\"type\": \"Microsoft.Network/publicIPAddresses\"",
				"\"type\": \"Microsoft.Network/loadBalancers\"",
				"\"type\": \"Microsoft.Storage/storageAccounts\"",
				"\"type\": \"Microsoft.Compute/virtualMachineScaleSets\"",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := config.Generate(tt.cs)
			if err != nil {
				t.Errorf("config.Generate():%s %v", tt.name, err)
				return
			}

			s := NewSimpleGenerator(logrus.NewEntry(logrus.New()))
			got, err := s.Generate(context.Background(), tt.cs, tt.isUpdate)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSimpleGenerator.Generate():%s error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			for _, resType := range tt.wantResourceTypes {
				if !strings.Contains(string(got), resType) {
					t.Fatalf("NewSimpleGenerator.Generate():%s missing resource type: %s", tt.name, resType)
				}
			}
		})
	}
}
