package plugin

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_arm"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_cluster"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_config"
)

func TestCreateOrUpdate(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

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
		armGenerator := mock_arm.NewMockGenerator(mockCtrl)
		armGenerator.EXPECT().Generate(nil, cs, "", tt.isUpdate, gomock.Any()).Return(nil, nil)
		clusterUpgrader := mock_cluster.NewMockUpgrader(mockCtrl)
		c := clusterUpgrader.EXPECT().CreateClients(nil, cs, tt.isUpdate).Return(nil)
		c = clusterUpgrader.EXPECT().Initialize(nil, cs).Return(nil).After(c)
		if tt.isUpdate {
			c = clusterUpgrader.EXPECT().SortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleMaster).Return([]api.AgentPoolProfile{cs.Properties.AgentPoolProfiles[0]}).After(c)
			c = clusterUpgrader.EXPECT().UpdateMasterAgentPool(nil, cs, &cs.Properties.AgentPoolProfiles[0]).Return(nil).After(c)
			c = clusterUpgrader.EXPECT().SortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleInfra).Return([]api.AgentPoolProfile{cs.Properties.AgentPoolProfiles[2]}).After(c)
			c = clusterUpgrader.EXPECT().UpdateWorkerAgentPool(nil, cs, &cs.Properties.AgentPoolProfiles[2], gomock.Any()).Return(nil).After(c)
			c = clusterUpgrader.EXPECT().SortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleCompute).Return([]api.AgentPoolProfile{cs.Properties.AgentPoolProfiles[1]}).After(c)
			c = clusterUpgrader.EXPECT().UpdateWorkerAgentPool(nil, cs, &cs.Properties.AgentPoolProfiles[1], gomock.Any()).Return(nil).After(c)
		} else {
			c = clusterUpgrader.EXPECT().InitializeUpdateBlob(cs, gomock.Any()).Return(nil).After(c)
			c = clusterUpgrader.EXPECT().WaitForHealthzStatusOk(nil, cs).Return(nil).After(c)
			c = clusterUpgrader.EXPECT().SortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleMaster).Return([]api.AgentPoolProfile{cs.Properties.AgentPoolProfiles[0]}).After(c)
			c = clusterUpgrader.EXPECT().WaitForNodesInAgentPoolProfile(nil, cs, &cs.Properties.AgentPoolProfiles[0], gomock.Any()).Return(nil).After(c)
			c = clusterUpgrader.EXPECT().SortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleInfra).Return([]api.AgentPoolProfile{cs.Properties.AgentPoolProfiles[2]}).After(c)
			c = clusterUpgrader.EXPECT().WaitForNodesInAgentPoolProfile(nil, cs, &cs.Properties.AgentPoolProfiles[2], gomock.Any()).Return(nil).After(c)
			c = clusterUpgrader.EXPECT().SortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleCompute).Return([]api.AgentPoolProfile{cs.Properties.AgentPoolProfiles[1]}).After(c)
			c = clusterUpgrader.EXPECT().WaitForNodesInAgentPoolProfile(nil, cs, &cs.Properties.AgentPoolProfiles[1], gomock.Any()).Return(nil).After(c)
		}
		c = clusterUpgrader.EXPECT().HealthCheck(nil, cs).Return(nil).After(c)
		p := &plugin{
			clusterUpgrader: clusterUpgrader,
			armGenerator:    armGenerator,
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
	armGenerator := mock_arm.NewMockGenerator(mockCtrl)
	clusterUpgrader := mock_cluster.NewMockUpgrader(mockCtrl)
	gomock.InOrder(
		armGenerator.EXPECT().Generate(nil, cs, gomock.Any(), true, gomock.Any()).Return(testDataWithBackup, nil),
		armGenerator.EXPECT().Generate(nil, cs, gomock.Any(), true, gomock.Any()).Return(testData, nil),
	)
	c := clusterUpgrader.EXPECT().CreateClients(nil, cs, true).Return(nil)
	c = clusterUpgrader.EXPECT().EtcdRestoreDeleteMasterScaleSet(nil, cs).Return(nil).After(c)

	// deploy masters
	c = clusterUpgrader.EXPECT().Initialize(nil, cs).Return(nil).After(c)
	c = clusterUpgrader.EXPECT().EtcdRestoreDeleteMasterScaleSetHashes(nil, cs).Return(nil).After(c)
	c = clusterUpgrader.EXPECT().WaitForHealthzStatusOk(nil, cs).Return(nil).After(c)
	c = clusterUpgrader.EXPECT().SortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleMaster).Return([]api.AgentPoolProfile{cs.Properties.AgentPoolProfiles[0]}).After(c)
	c = clusterUpgrader.EXPECT().WaitForNodesInAgentPoolProfile(nil, cs, &cs.Properties.AgentPoolProfiles[0], "").Return(nil).After(c)
	// update
	c = clusterUpgrader.EXPECT().CreateClients(nil, cs, true).Return(nil).After(c)
	c = clusterUpgrader.EXPECT().Initialize(nil, cs).Return(nil).After(c)
	c = clusterUpgrader.EXPECT().SortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleMaster).Return([]api.AgentPoolProfile{cs.Properties.AgentPoolProfiles[0]}).After(c)
	c = clusterUpgrader.EXPECT().UpdateMasterAgentPool(nil, cs, &cs.Properties.AgentPoolProfiles[0]).Return(nil).After(c)
	c = clusterUpgrader.EXPECT().SortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleInfra).Return([]api.AgentPoolProfile{cs.Properties.AgentPoolProfiles[2]}).After(c)
	c = clusterUpgrader.EXPECT().UpdateWorkerAgentPool(nil, cs, &cs.Properties.AgentPoolProfiles[2], gomock.Any()).Return(nil).After(c)
	c = clusterUpgrader.EXPECT().SortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleCompute).Return([]api.AgentPoolProfile{cs.Properties.AgentPoolProfiles[1]}).After(c)
	c = clusterUpgrader.EXPECT().UpdateWorkerAgentPool(nil, cs, &cs.Properties.AgentPoolProfiles[1], gomock.Any()).Return(nil).After(c)
	c = clusterUpgrader.EXPECT().HealthCheck(nil, cs).Return(nil).After(c)

	p := &plugin{
		clusterUpgrader: clusterUpgrader,
		armGenerator:    armGenerator,
		log:             logrus.NewEntry(logrus.StandardLogger()),
	}

	if err := p.RecoverEtcdCluster(nil, cs, deployer, "test-backup"); err != nil {
		t.Errorf("plugin.RecoverEtcdCluster error = %v", err)
	}
}

func TestRotateClusterSecrets(t *testing.T) {
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

	mockGen := mock_config.NewMockGenerator(mockCtrl)
	mockUp := mock_cluster.NewMockUpgrader(mockCtrl)
	mockArm := mock_arm.NewMockGenerator(mockCtrl)

	c := mockGen.EXPECT().InvalidateSecrets(cs).Return(nil)
	c = mockGen.EXPECT().Generate(cs, nil).Return(nil).After(c)
	c = mockArm.EXPECT().Generate(nil, cs, "", true, gomock.Any()).Return(nil, nil).After(c)
	c = mockUp.EXPECT().CreateClients(nil, cs, true).Return(nil).After(c)
	c = mockUp.EXPECT().Initialize(nil, cs).Return(nil).After(c)
	c = mockUp.EXPECT().SortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleMaster).Return([]api.AgentPoolProfile{cs.Properties.AgentPoolProfiles[0]}).After(c)
	c = mockUp.EXPECT().UpdateMasterAgentPool(nil, cs, &cs.Properties.AgentPoolProfiles[0]).Return(nil).After(c)
	c = mockUp.EXPECT().SortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleInfra).Return([]api.AgentPoolProfile{cs.Properties.AgentPoolProfiles[2]}).After(c)
	c = mockUp.EXPECT().UpdateWorkerAgentPool(nil, cs, &cs.Properties.AgentPoolProfiles[2], gomock.Any()).Return(nil).After(c)
	c = mockUp.EXPECT().SortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleCompute).Return([]api.AgentPoolProfile{cs.Properties.AgentPoolProfiles[1]}).After(c)
	c = mockUp.EXPECT().UpdateWorkerAgentPool(nil, cs, &cs.Properties.AgentPoolProfiles[1], gomock.Any()).Return(nil).After(c)
	c = mockUp.EXPECT().HealthCheck(nil, cs).Return(nil).After(c)

	p := &plugin{
		armGenerator:    mockArm,
		clusterUpgrader: mockUp,
		configGenerator: mockGen,
		log:             logrus.NewEntry(logrus.StandardLogger()),
	}

	if err := p.RotateClusterSecrets(nil, cs, deployer, nil); err != nil {
		t.Errorf("plugin.RotateClusterSecrets error = %v", err)
	}
}
