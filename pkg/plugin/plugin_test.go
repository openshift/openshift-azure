package plugin

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	pluginapi "github.com/openshift/openshift-azure/pkg/api/plugin/api"
	"github.com/openshift/openshift-azure/pkg/cluster"
	"github.com/openshift/openshift-azure/pkg/config"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_arm"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_cluster"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_config"
)

func TestCreateOrUpdate(t *testing.T) {
	gmc := gomock.NewController(t)
	defer gmc.Finish()

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
		Config: api.Config{ConfigStorageAccount: "config"},
		Properties: api.Properties{
			AgentPoolProfiles: []api.AgentPoolProfile{
				{Role: api.AgentPoolProfileRoleMaster, Name: "master"},
				{Role: api.AgentPoolProfileRoleCompute, Name: "compute"},
				{Role: api.AgentPoolProfileRoleInfra, Name: "infra"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			armGenerator := mock_arm.NewMockGenerator(gmc)
			clusterUpgrader := mock_cluster.NewMockUpgrader(gmc)
			c := clusterUpgrader.EXPECT().CreateOrUpdateConfigStorageAccount(nil, cs).Return(nil)
			c = armGenerator.EXPECT().Generate(nil, cs, "", tt.isUpdate, gomock.Any()).Return(nil, nil).After(c)
			c = clusterUpgrader.EXPECT().WriteStartupBlobs(cs).Return(nil).After(c)
			if !tt.isUpdate {
				c = clusterUpgrader.EXPECT().WriteSyncBlob(cs).Return(nil).After(c)
			}
			c = clusterUpgrader.EXPECT().EnrichCSFromVault(nil, cs).Return(nil)
			c = clusterUpgrader.EXPECT().EnrichCSStorageAccountKeys(nil, cs).Return(nil)
			if tt.isUpdate {
				c = clusterUpgrader.EXPECT().SortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleMaster).Return([]api.AgentPoolProfile{cs.Properties.AgentPoolProfiles[0]}).After(c)
				c = clusterUpgrader.EXPECT().UpdateMasterAgentPool(nil, cs, &cs.Properties.AgentPoolProfiles[0]).Return(nil).After(c)
				c = clusterUpgrader.EXPECT().SortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleInfra).Return([]api.AgentPoolProfile{cs.Properties.AgentPoolProfiles[2]}).After(c)
				c = clusterUpgrader.EXPECT().UpdateWorkerAgentPool(nil, cs, &cs.Properties.AgentPoolProfiles[2], gomock.Any()).Return(nil).After(c)
				c = clusterUpgrader.EXPECT().SortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleCompute).Return([]api.AgentPoolProfile{cs.Properties.AgentPoolProfiles[1]}).After(c)
				c = clusterUpgrader.EXPECT().UpdateWorkerAgentPool(nil, cs, &cs.Properties.AgentPoolProfiles[1], gomock.Any()).Return(nil).After(c)
				c = clusterUpgrader.EXPECT().WriteSyncBlob(cs).Return(nil).After(c)
				c = clusterUpgrader.EXPECT().UpdateSyncPod(nil, cs).Return(nil).After(c)
			} else {
				c = clusterUpgrader.EXPECT().InitializeUpdateBlob(cs, gomock.Any()).Return(nil).After(c)
				c = clusterUpgrader.EXPECT().WaitForHealthzStatusOk(nil, cs).Return(nil).After(c)
				c = clusterUpgrader.EXPECT().SortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleMaster).Return([]api.AgentPoolProfile{cs.Properties.AgentPoolProfiles[0]}).After(c)
				c = clusterUpgrader.EXPECT().WaitForNodesInAgentPoolProfile(nil, cs, &cs.Properties.AgentPoolProfiles[0], gomock.Any()).Return(nil).After(c)
				c = clusterUpgrader.EXPECT().SortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleInfra).Return([]api.AgentPoolProfile{cs.Properties.AgentPoolProfiles[2]}).After(c)
				c = clusterUpgrader.EXPECT().WaitForNodesInAgentPoolProfile(nil, cs, &cs.Properties.AgentPoolProfiles[2], gomock.Any()).Return(nil).After(c)
				c = clusterUpgrader.EXPECT().SortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleCompute).Return([]api.AgentPoolProfile{cs.Properties.AgentPoolProfiles[1]}).After(c)
				c = clusterUpgrader.EXPECT().WaitForNodesInAgentPoolProfile(nil, cs, &cs.Properties.AgentPoolProfiles[1], gomock.Any()).Return(nil).After(c)
				c = clusterUpgrader.EXPECT().WaitForReadySyncPod(nil).Return(nil).After(c)
			}
			c = clusterUpgrader.EXPECT().HealthCheck(nil, cs).Return(nil).After(c)
			p := &plugin{
				upgraderFactory: func(ctx context.Context, log *logrus.Entry, cs *api.OpenShiftManagedCluster, initializeStorageClients, disableKeepAlives bool, testConfig api.TestConfig) (cluster.Upgrader, error) {
					return clusterUpgrader, nil
				},
				armGenerator: armGenerator,
				log:          logrus.NewEntry(logrus.StandardLogger()),
			}
			if err := p.CreateOrUpdate(nil, cs, tt.isUpdate, deployer); err != nil {
				t.Errorf("plugin.CreateOrUpdate(%s) error = %v", tt.name, err)
			}
		})
	}
}

func TestRecoverEtcdCluster(t *testing.T) {
	gmc := gomock.NewController(t)
	defer gmc.Finish()

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
	armGenerator := mock_arm.NewMockGenerator(gmc)
	clusterUpgrader := mock_cluster.NewMockUpgrader(gmc)

	c := armGenerator.EXPECT().Generate(nil, cs, gomock.Any(), true, gomock.Any()).Return(testDataWithBackup, nil)
	c = clusterUpgrader.EXPECT().EtcdBlobExists(nil, "test-backup").Return(nil).After(c)
	c = clusterUpgrader.EXPECT().EtcdRestoreDeleteMasterScaleSet(nil, cs).Return(nil).After(c)

	// deploy masters
	c = clusterUpgrader.EXPECT().EtcdRestoreDeleteMasterScaleSetHashes(nil, cs).Return(nil).After(c)
	c = clusterUpgrader.EXPECT().WaitForHealthzStatusOk(nil, cs).Return(nil).After(c)
	c = clusterUpgrader.EXPECT().SortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleMaster).Return([]api.AgentPoolProfile{cs.Properties.AgentPoolProfiles[0]}).After(c)
	c = clusterUpgrader.EXPECT().WaitForNodesInAgentPoolProfile(nil, cs, &cs.Properties.AgentPoolProfiles[0], "").Return(nil).After(c)
	// update
	c = clusterUpgrader.EXPECT().CreateOrUpdateConfigStorageAccount(nil, cs).Return(nil).After(c)
	c = armGenerator.EXPECT().Generate(nil, cs, gomock.Any(), true, gomock.Any()).Return(testData, nil).After(c)
	c = clusterUpgrader.EXPECT().WriteStartupBlobs(cs).Return(nil).After(c)
	c = clusterUpgrader.EXPECT().EnrichCSFromVault(nil, cs).Return(nil)
	c = clusterUpgrader.EXPECT().EnrichCSStorageAccountKeys(nil, cs).Return(nil)
	c = clusterUpgrader.EXPECT().SortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleMaster).Return([]api.AgentPoolProfile{cs.Properties.AgentPoolProfiles[0]}).After(c)
	c = clusterUpgrader.EXPECT().UpdateMasterAgentPool(nil, cs, &cs.Properties.AgentPoolProfiles[0]).Return(nil).After(c)
	c = clusterUpgrader.EXPECT().SortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleInfra).Return([]api.AgentPoolProfile{cs.Properties.AgentPoolProfiles[2]}).After(c)
	c = clusterUpgrader.EXPECT().UpdateWorkerAgentPool(nil, cs, &cs.Properties.AgentPoolProfiles[2], gomock.Any()).Return(nil).After(c)
	c = clusterUpgrader.EXPECT().SortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleCompute).Return([]api.AgentPoolProfile{cs.Properties.AgentPoolProfiles[1]}).After(c)
	c = clusterUpgrader.EXPECT().UpdateWorkerAgentPool(nil, cs, &cs.Properties.AgentPoolProfiles[1], gomock.Any()).Return(nil).After(c)
	c = clusterUpgrader.EXPECT().WriteSyncBlob(cs).Return(nil).After(c)
	c = clusterUpgrader.EXPECT().UpdateSyncPod(nil, cs).Return(nil).After(c)
	c = clusterUpgrader.EXPECT().HealthCheck(nil, cs).Return(nil).After(c)

	p := &plugin{
		upgraderFactory: func(ctx context.Context, log *logrus.Entry, cs *api.OpenShiftManagedCluster, initializeStorageClients, disableKeepAlives bool, testConfig api.TestConfig) (cluster.Upgrader, error) {
			return clusterUpgrader, nil
		},
		armGenerator: armGenerator,
		log:          logrus.NewEntry(logrus.StandardLogger()),
	}

	if err := p.RecoverEtcdCluster(nil, cs, deployer, "test-backup"); err != nil {
		t.Errorf("plugin.RecoverEtcdCluster error = %v", err)
	}
}

func TestRotateClusterSecrets(t *testing.T) {
	gmc := gomock.NewController(t)
	defer gmc.Finish()

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

	configGenerator := mock_config.NewMockGenerator(gmc)
	clusterUpgrader := mock_cluster.NewMockUpgrader(gmc)
	armGenerator := mock_arm.NewMockGenerator(gmc)

	c := configGenerator.EXPECT().InvalidateSecrets(cs).Return(nil)
	c = configGenerator.EXPECT().Generate(cs, nil).Return(nil).After(c)
	c = clusterUpgrader.EXPECT().CreateOrUpdateConfigStorageAccount(nil, cs).Return(nil).After(c)
	c = armGenerator.EXPECT().Generate(nil, cs, "", true, gomock.Any()).Return(nil, nil).After(c)
	c = clusterUpgrader.EXPECT().WriteStartupBlobs(cs).Return(nil).After(c)
	c = clusterUpgrader.EXPECT().EnrichCSFromVault(nil, cs).Return(nil)
	c = clusterUpgrader.EXPECT().EnrichCSStorageAccountKeys(nil, cs).Return(nil)
	c = clusterUpgrader.EXPECT().SortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleMaster).Return([]api.AgentPoolProfile{cs.Properties.AgentPoolProfiles[0]}).After(c)
	c = clusterUpgrader.EXPECT().UpdateMasterAgentPool(nil, cs, &cs.Properties.AgentPoolProfiles[0]).Return(nil).After(c)
	c = clusterUpgrader.EXPECT().SortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleInfra).Return([]api.AgentPoolProfile{cs.Properties.AgentPoolProfiles[2]}).After(c)
	c = clusterUpgrader.EXPECT().UpdateWorkerAgentPool(nil, cs, &cs.Properties.AgentPoolProfiles[2], gomock.Any()).Return(nil).After(c)
	c = clusterUpgrader.EXPECT().SortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleCompute).Return([]api.AgentPoolProfile{cs.Properties.AgentPoolProfiles[1]}).After(c)
	c = clusterUpgrader.EXPECT().UpdateWorkerAgentPool(nil, cs, &cs.Properties.AgentPoolProfiles[1], gomock.Any()).Return(nil).After(c)
	c = clusterUpgrader.EXPECT().WriteSyncBlob(cs).Return(nil).After(c)
	c = clusterUpgrader.EXPECT().UpdateSyncPod(nil, cs).Return(nil).After(c)
	c = clusterUpgrader.EXPECT().HealthCheck(nil, cs).Return(nil).After(c)

	p := &plugin{
		armGenerator: armGenerator,
		upgraderFactory: func(ctx context.Context, log *logrus.Entry, cs *api.OpenShiftManagedCluster, initializeStorageClients, disableKeepAlives bool, testConfig api.TestConfig) (cluster.Upgrader, error) {
			return clusterUpgrader, nil
		},
		configGenerator: configGenerator,
		log:             logrus.NewEntry(logrus.StandardLogger()),
	}

	if err := p.RotateClusterSecrets(nil, cs, deployer); err != nil {
		t.Errorf("plugin.RotateClusterSecrets error = %v", err)
	}
}

func TestForceUpdate(t *testing.T) {
	gmc := gomock.NewController(t)
	defer gmc.Finish()

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

	clusterUpgrader := mock_cluster.NewMockUpgrader(gmc)
	armGenerator := mock_arm.NewMockGenerator(gmc)

	c := clusterUpgrader.EXPECT().ResetUpdateBlob(cs).Return(nil)
	c = clusterUpgrader.EXPECT().CreateOrUpdateConfigStorageAccount(nil, cs).Return(nil).After(c)
	c = armGenerator.EXPECT().Generate(nil, cs, "", true, gomock.Any()).Return(nil, nil).After(c)
	c = clusterUpgrader.EXPECT().WriteStartupBlobs(cs).Return(nil).After(c)
	c = clusterUpgrader.EXPECT().EnrichCSFromVault(nil, cs).Return(nil)
	c = clusterUpgrader.EXPECT().EnrichCSStorageAccountKeys(nil, cs).Return(nil)
	c = clusterUpgrader.EXPECT().SortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleMaster).Return([]api.AgentPoolProfile{cs.Properties.AgentPoolProfiles[0]}).After(c)
	c = clusterUpgrader.EXPECT().UpdateMasterAgentPool(nil, cs, &cs.Properties.AgentPoolProfiles[0]).Return(nil).After(c)
	c = clusterUpgrader.EXPECT().SortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleInfra).Return([]api.AgentPoolProfile{cs.Properties.AgentPoolProfiles[2]}).After(c)
	c = clusterUpgrader.EXPECT().UpdateWorkerAgentPool(nil, cs, &cs.Properties.AgentPoolProfiles[2], gomock.Any()).Return(nil).After(c)
	c = clusterUpgrader.EXPECT().SortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleCompute).Return([]api.AgentPoolProfile{cs.Properties.AgentPoolProfiles[1]}).After(c)
	c = clusterUpgrader.EXPECT().UpdateWorkerAgentPool(nil, cs, &cs.Properties.AgentPoolProfiles[1], gomock.Any()).Return(nil).After(c)
	c = clusterUpgrader.EXPECT().WriteSyncBlob(cs).Return(nil).After(c)
	c = clusterUpgrader.EXPECT().UpdateSyncPod(nil, cs).Return(nil).After(c)
	c = clusterUpgrader.EXPECT().HealthCheck(nil, cs).Return(nil).After(c)

	p := &plugin{
		armGenerator: armGenerator,
		upgraderFactory: func(ctx context.Context, log *logrus.Entry, cs *api.OpenShiftManagedCluster, initializeStorageClients, disableKeepAlives bool, testConfig api.TestConfig) (cluster.Upgrader, error) {
			return clusterUpgrader, nil
		},
		log: logrus.NewEntry(logrus.StandardLogger()),
	}

	if err := p.ForceUpdate(nil, cs, deployer); err != nil {
		t.Errorf("plugin.ForceUpdate error = %v", err)
	}
}

func TestListClusterVMs(t *testing.T) {
	gmc := gomock.NewController(t)
	defer gmc.Finish()

	clusterUpgrader := mock_cluster.NewMockUpgrader(gmc)

	clusterUpgrader.EXPECT().ListVMHostnames(nil, nil).Return(nil, nil)

	p := &plugin{
		upgraderFactory: func(ctx context.Context, log *logrus.Entry, cs *api.OpenShiftManagedCluster, initializeStorageClients, disableKeepAlives bool, testConfig api.TestConfig) (cluster.Upgrader, error) {
			return clusterUpgrader, nil
		},
		log: logrus.NewEntry(logrus.StandardLogger()),
	}

	if _, err := p.ListClusterVMs(nil, nil); err != nil {
		t.Errorf("plugin.ListClusterVMs() error = %v", err)
	}
}

func TestReimage(t *testing.T) {
	gmc := gomock.NewController(t)
	defer gmc.Finish()

	tests := []struct {
		name     string
		hostname string
		isMaster bool
	}{
		{
			name:     "reimage master vm",
			hostname: "master-000A00",
			isMaster: true,
		},
		{
			name:     "reimage compute vm",
			hostname: "compute-1550971226-000000",
			isMaster: false,
		},
		{
			name:     "reimage infra vm",
			hostname: "infra-1550971226-000000",
			isMaster: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clusterUpgrader := mock_cluster.NewMockUpgrader(gmc)

			scaleset, instanceID, err := config.GetScaleSetNameAndInstanceID(tt.hostname)
			if err != nil {
				t.Fatal(err)
			}

			c := clusterUpgrader.EXPECT().Reimage(nil, nil, scaleset, instanceID).Return(nil)

			if tt.isMaster {
				c = clusterUpgrader.EXPECT().WaitForReadyMaster(nil, tt.hostname).Return(nil).After(c)
			} else {
				c = clusterUpgrader.EXPECT().WaitForReadyWorker(nil, tt.hostname).Return(nil).After(c)
			}

			p := &plugin{
				upgraderFactory: func(ctx context.Context, log *logrus.Entry, cs *api.OpenShiftManagedCluster, initializeStorageClients, disableKeepAlives bool, testConfig api.TestConfig) (cluster.Upgrader, error) {
					return clusterUpgrader, nil
				},
				log: logrus.NewEntry(logrus.StandardLogger()),
			}

			if err := p.Reimage(nil, nil, tt.hostname); err != nil {
				t.Errorf("plugin.Reimage(%s) error = %v", tt.hostname, err)
			}
		})
	}
}
func TestBackupEtcdCluster(t *testing.T) {
	gmc := gomock.NewController(t)
	defer gmc.Finish()

	backupName := "etcd-backup"
	cs := &api.OpenShiftManagedCluster{
		Properties: api.Properties{
			AgentPoolProfiles: []api.AgentPoolProfile{
				{Role: api.AgentPoolProfileRoleMaster, Name: "master"},
				{Role: api.AgentPoolProfileRoleCompute, Name: "compute"},
				{Role: api.AgentPoolProfileRoleInfra, Name: "infra"},
			},
		},
	}

	clusterUpgrader := mock_cluster.NewMockUpgrader(gmc)
	clusterUpgrader.EXPECT().BackupCluster(nil, backupName).Return(nil)

	p := &plugin{
		upgraderFactory: func(ctx context.Context, log *logrus.Entry, cs *api.OpenShiftManagedCluster, initializeStorageClients, disableKeepAlives bool, testConfig api.TestConfig) (cluster.Upgrader, error) {
			return clusterUpgrader, nil
		},
		log: logrus.NewEntry(logrus.StandardLogger()),
	}

	err := p.BackupEtcdCluster(nil, cs, backupName)
	if err != nil {
		t.Errorf("plugin.BackupEtcdCluster error = %v", err)
	}
}

func TestGetPluginVersion(t *testing.T) {
	p := &plugin{
		pluginConfig: &pluginapi.Config{
			PluginVersion: "v0.0",
		},
	}
	result := p.GetPluginVersion(nil)
	if *result.PluginVersion != p.pluginConfig.PluginVersion {
		t.Errorf("expected plugin version %s, got %s", p.pluginConfig.PluginVersion, *result.PluginVersion)
	}
}
