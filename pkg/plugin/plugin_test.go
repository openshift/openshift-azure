package plugin

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster/updateblob"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_arm"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_cluster"
)

func TestCreateOrUpdate(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockUp := mock_cluster.NewMockUpgrader(mockCtrl)
	tests := []struct {
		name     string
		isUpdate bool
	}{
		{
			name:     "deploy",
			isUpdate: false,
		},
		{
			name:     "update",
			isUpdate: true,
		},
	}
	deployer := func(ctx context.Context, azuretemplate map[string]interface{}) error {
		return nil
	}
	cs := &api.OpenShiftManagedCluster{
		Properties: api.Properties{
			AgentPoolProfiles: []api.AgentPoolProfile{
				{Role: api.AgentPoolProfileRoleMaster, Name: "master"},
				{Role: api.AgentPoolProfileRoleCompute, Name: "compute"},
				{Role: api.AgentPoolProfileRoleInfra, Name: "infra"},
			},
		},
	}

	for _, tt := range tests {
		mockGen := mock_arm.NewMockGenerator(mockCtrl)
		mockGen.EXPECT().Generate(nil, cs, "", tt.isUpdate, gomock.Any()).Return(nil, nil)
		mockUp.EXPECT().CreateClients(nil, cs).Return(nil)
		mockUp.EXPECT().Initialize(nil, cs).Return(nil)
		if tt.isUpdate {
			mockUp.EXPECT().UpdateMasterAgentPool(nil, cs, &cs.Properties.AgentPoolProfiles[0]).Return(nil)
			mockUp.EXPECT().UpdateWorkerAgentPool(nil, cs, &cs.Properties.AgentPoolProfiles[1], gomock.Any()).Return(nil)
			mockUp.EXPECT().UpdateWorkerAgentPool(nil, cs, &cs.Properties.AgentPoolProfiles[2], gomock.Any()).Return(nil)
		} else {
			mockUp.EXPECT().InitializeUpdateBlob(cs, gomock.Any()).Return(nil)
			mockUp.EXPECT().WaitForHealthzStatusOk(nil, cs).Return(nil)
			mockUp.EXPECT().WaitForNodes(nil, cs, gomock.Any()).Return(nil)
		}
		mockUp.EXPECT().WaitForInfraServices(nil, cs).Return(nil)
		mockUp.EXPECT().HealthCheck(nil, cs).Return(nil)
		p := &plugin{
			clusterUpgrader: mockUp,
			armGenerator:    mockGen,
			log:             logrus.NewEntry(logrus.StandardLogger()),
		}
		if err := p.CreateOrUpdate(nil, cs, tt.isUpdate, deployer); err != nil {
			t.Errorf("plugin.CreateOrUpdate(%s) error = %v", tt.name, err)
		}
	}
}

func TestRecoverEtcdCluster(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	deployer := func(ctx context.Context, azuretemplate map[string]interface{}) error {
		return nil
	}
	cs := &api.OpenShiftManagedCluster{
		Properties: api.Properties{
			AgentPoolProfiles: []api.AgentPoolProfile{
				{Role: api.AgentPoolProfileRoleMaster, Name: "master"},
				{Role: api.AgentPoolProfileRoleCompute, Name: "compute"},
				{Role: api.AgentPoolProfileRoleInfra, Name: "infra"},
			},
		},
	}

	testData := map[string]interface{}{"test": "data"}
	testDataWithBackup := map[string]interface{}{"test": "backup"}
	mockGen := mock_arm.NewMockGenerator(mockCtrl)
	mockUp := mock_cluster.NewMockUpgrader(mockCtrl)
	gomock.InOrder(
		mockGen.EXPECT().Generate(nil, cs, gomock.Any(), true, gomock.Any()).Return(testDataWithBackup, nil),
		mockGen.EXPECT().Generate(nil, cs, gomock.Any(), true, gomock.Any()).Return(testData, nil),
	)
	mockUp.EXPECT().CreateClients(nil, cs).Return(nil)
	mockUp.EXPECT().Evacuate(nil, cs).Return(nil)

	// deploy masters
	mockUp.EXPECT().Initialize(nil, cs).Return(nil)
	ub := updateblob.NewUpdateBlob()
	mockUp.EXPECT().ReadUpdateBlob().Return(ub, nil)
	mockUp.EXPECT().WriteUpdateBlob(gomock.Any()).Return(nil)
	mockUp.EXPECT().WaitForHealthzStatusOk(nil, cs).Return(nil)
	mockUp.EXPECT().WaitForMasters(nil, cs).Return(nil)
	// update
	mockUp.EXPECT().CreateClients(nil, cs).Return(nil)
	mockUp.EXPECT().Initialize(nil, cs).Return(nil)
	mockUp.EXPECT().UpdateMasterAgentPool(nil, cs, &cs.Properties.AgentPoolProfiles[0]).Return(nil)
	mockUp.EXPECT().UpdateWorkerAgentPool(nil, cs, &cs.Properties.AgentPoolProfiles[1], gomock.Any()).Return(nil)
	mockUp.EXPECT().UpdateWorkerAgentPool(nil, cs, &cs.Properties.AgentPoolProfiles[2], gomock.Any()).Return(nil)
	mockUp.EXPECT().WaitForInfraServices(nil, cs).Return(nil)
	mockUp.EXPECT().HealthCheck(nil, cs).Return(nil)

	p := &plugin{
		clusterUpgrader: mockUp,
		armGenerator:    mockGen,
		log:             logrus.NewEntry(logrus.StandardLogger()),
	}

	if err := p.RecoverEtcdCluster(nil, cs, deployer, "test-backup"); err != nil {
		t.Errorf("plugin.RecoverEtcdCluster error = %v", err)
	}
}
