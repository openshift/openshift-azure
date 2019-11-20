package plugin

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	pluginapi "github.com/openshift/openshift-azure/pkg/api/plugin"
	"github.com/openshift/openshift-azure/pkg/cluster"
	"github.com/openshift/openshift-azure/pkg/cluster/names"
	"github.com/openshift/openshift-azure/pkg/config"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_cluster"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_config"
)

func expectUpdate(cs *api.OpenShiftManagedCluster, clusterUpgrader *mock_cluster.MockUpgrader, c **gomock.Call) {
	if *c == nil {
		*c = clusterUpgrader.EXPECT().CreateOrUpdateConfigStorageAccount(nil).Return(nil)
	} else {
		*c = clusterUpgrader.EXPECT().CreateOrUpdateConfigStorageAccount(nil).Return(nil).After(*c)
	}
	*c = clusterUpgrader.EXPECT().GenerateARM(nil, "", true, gomock.Any()).Return(nil, nil).After(*c)
	*c = clusterUpgrader.EXPECT().GetNameserversFromVnet(nil, gomock.Any(), cs.Properties.AzProfile.SubscriptionID, cs.Properties.AzProfile.ResourceGroup).After(*c)
	*c = clusterUpgrader.EXPECT().WriteStartupBlobs().Return(nil).After(*c)
	*c = clusterUpgrader.EXPECT().EnrichCertificatesFromVault(nil).Return(nil)
	*c = clusterUpgrader.EXPECT().EnrichStorageAccountKeys(nil).Return(nil)
	*c = clusterUpgrader.EXPECT().SortedAgentPoolProfilesForRole(api.AgentPoolProfileRoleMaster).Return([]api.AgentPoolProfile{cs.Properties.AgentPoolProfiles[0]}).After(*c)
	*c = clusterUpgrader.EXPECT().UpdateMasterAgentPool(nil, &cs.Properties.AgentPoolProfiles[0]).Return(nil).After(*c)
	*c = clusterUpgrader.EXPECT().SortedAgentPoolProfilesForRole(api.AgentPoolProfileRoleInfra).Return([]api.AgentPoolProfile{cs.Properties.AgentPoolProfiles[2]}).After(*c)
	*c = clusterUpgrader.EXPECT().UpdateWorkerAgentPool(nil, &cs.Properties.AgentPoolProfiles[2], gomock.Any()).Return(nil).After(*c)
	*c = clusterUpgrader.EXPECT().SortedAgentPoolProfilesForRole(api.AgentPoolProfileRoleCompute).Return([]api.AgentPoolProfile{cs.Properties.AgentPoolProfiles[1]}).After(*c)
	*c = clusterUpgrader.EXPECT().UpdateWorkerAgentPool(nil, &cs.Properties.AgentPoolProfiles[1], gomock.Any()).Return(nil).After(*c)
	*c = clusterUpgrader.EXPECT().CreateOrUpdateSyncPod(nil).Return(nil).After(*c)
	*c = clusterUpgrader.EXPECT().WaitForReadySyncPod(nil).Return(nil).After(*c)
	*c = clusterUpgrader.EXPECT().HealthCheck(nil).Return(nil).After(*c)
}

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
	deployer := func(ctx context.Context, azuretemplate map[string]interface{}) (*string, error) {
		return nil, nil
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
			clusterUpgrader := mock_cluster.NewMockUpgrader(gmc)
			if tt.isUpdate {
				var c *gomock.Call
				clusterUpgrader.EXPECT().BackupCluster(nil, gomock.Any())
				expectUpdate(cs, clusterUpgrader, &c)
			} else {
				c := clusterUpgrader.EXPECT().CreateOrUpdateConfigStorageAccount(nil).Return(nil)
				c = clusterUpgrader.EXPECT().GenerateARM(nil, "", tt.isUpdate, gomock.Any()).Return(nil, nil).After(c)
				c = clusterUpgrader.EXPECT().WriteStartupBlobs().Return(nil).After(c)
				c = clusterUpgrader.EXPECT().EnrichCertificatesFromVault(nil).Return(nil)
				c = clusterUpgrader.EXPECT().EnrichStorageAccountKeys(nil).Return(nil)
				c = clusterUpgrader.EXPECT().InitializeUpdateBlob(gomock.Any()).Return(nil).After(c)
				c = clusterUpgrader.EXPECT().WaitForHealthzStatusOk(nil).Return(nil).After(c)
				c = clusterUpgrader.EXPECT().CreateOrUpdateSyncPod(nil).Return(nil).After(c)
				c = clusterUpgrader.EXPECT().SortedAgentPoolProfilesForRole(api.AgentPoolProfileRoleMaster).Return([]api.AgentPoolProfile{cs.Properties.AgentPoolProfiles[0]}).After(c)
				c = clusterUpgrader.EXPECT().WaitForNodesInAgentPoolProfile(nil, &cs.Properties.AgentPoolProfiles[0], gomock.Any()).Return(nil).After(c)
				c = clusterUpgrader.EXPECT().SortedAgentPoolProfilesForRole(api.AgentPoolProfileRoleInfra).Return([]api.AgentPoolProfile{cs.Properties.AgentPoolProfiles[2]}).After(c)
				c = clusterUpgrader.EXPECT().WaitForNodesInAgentPoolProfile(nil, &cs.Properties.AgentPoolProfiles[2], gomock.Any()).Return(nil).After(c)
				c = clusterUpgrader.EXPECT().SortedAgentPoolProfilesForRole(api.AgentPoolProfileRoleCompute).Return([]api.AgentPoolProfile{cs.Properties.AgentPoolProfiles[1]}).After(c)
				c = clusterUpgrader.EXPECT().WaitForNodesInAgentPoolProfile(nil, &cs.Properties.AgentPoolProfiles[1], gomock.Any()).Return(nil).After(c)
				c = clusterUpgrader.EXPECT().WaitForReadySyncPod(nil).Return(nil).After(c)
				c = clusterUpgrader.EXPECT().HealthCheck(nil).Return(nil).After(c)
			}
			p := &plugin{
				upgraderFactory: func(ctx context.Context, log *logrus.Entry, cs *api.OpenShiftManagedCluster, initializeStorageClients, disableKeepAlives bool, testConfig api.TestConfig) (cluster.Upgrader, error) {
					return clusterUpgrader, nil
				},
				log: logrus.NewEntry(logrus.StandardLogger()),
				now: time.Now,
			}
			if err := p.CreateOrUpdate(nil, cs, tt.isUpdate, deployer); err != nil {
				t.Errorf("plugin.CreateOrUpdate(%s) error = %v", tt.name, err)
			}
		})
	}
}

func TestListEtcdBackups(t *testing.T) {
	gmc := gomock.NewController(t)
	defer gmc.Finish()

	clusterUpgrader := mock_cluster.NewMockUpgrader(gmc)
	clusterUpgrader.EXPECT().EtcdListBackups(nil).Return([]storage.Blob{{Name: "test-backup"}}, nil)

	p := &plugin{
		upgraderFactory: func(ctx context.Context, log *logrus.Entry, cs *api.OpenShiftManagedCluster, initializeStorageClients, disableKeepAlives bool, testConfig api.TestConfig) (cluster.Upgrader, error) {
			return clusterUpgrader, nil
		},
		log: logrus.NewEntry(logrus.StandardLogger()),
	}

	backups, err := p.ListEtcdBackups(nil, nil)
	if err != nil {
		t.Errorf("plugin.ListEtcdBackups error = %v", err)
	}
	if len(backups) != 1 || backups[0].Name != "test-backup" {
		t.Errorf("plugin.ListEtcdBackups unexpected response %v", backups)
	}
}

func TestRecoverEtcdCluster(t *testing.T) {
	gmc := gomock.NewController(t)
	defer gmc.Finish()

	deployer := func(ctx context.Context, azuretemplate map[string]interface{}) (*string, error) {
		return nil, nil
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
	c := clusterUpgrader.EXPECT().GetNameserversFromVnet(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	c = clusterUpgrader.EXPECT().GenerateARM(nil, gomock.Any(), true, gomock.Any()).Return(nil, nil).After(c)
	c = clusterUpgrader.EXPECT().EtcdListBackups(nil).Return([]storage.Blob{{Name: "test-backup"}}, nil).After(c)
	c = clusterUpgrader.EXPECT().EtcdRestoreDeleteMasterScaleSet(nil).Return(nil).After(c)

	// deploy masters
	c = clusterUpgrader.EXPECT().EtcdRestoreDeleteMasterScaleSetHashes(nil).Return(nil).After(c)
	c = clusterUpgrader.EXPECT().WaitForHealthzStatusOk(nil).Return(nil).After(c)
	c = clusterUpgrader.EXPECT().SortedAgentPoolProfilesForRole(api.AgentPoolProfileRoleMaster).Return([]api.AgentPoolProfile{cs.Properties.AgentPoolProfiles[0]}).After(c)
	c = clusterUpgrader.EXPECT().WaitForNodesInAgentPoolProfile(nil, &cs.Properties.AgentPoolProfiles[0], "").Return(nil).After(c)
	// update
	expectUpdate(cs, clusterUpgrader, &c)

	p := &plugin{
		upgraderFactory: func(ctx context.Context, log *logrus.Entry, cs *api.OpenShiftManagedCluster, initializeStorageClients, disableKeepAlives bool, testConfig api.TestConfig) (cluster.Upgrader, error) {
			return clusterUpgrader, nil
		},
		log: logrus.NewEntry(logrus.StandardLogger()),
		now: time.Now,
	}

	if err := p.RecoverEtcdCluster(nil, cs, deployer, "test-backup"); err != nil {
		t.Errorf("plugin.RecoverEtcdCluster error = %v", err)
	}
}

func TestRotateClusterSecrets(t *testing.T) {
	gmc := gomock.NewController(t)
	defer gmc.Finish()

	deployer := func(ctx context.Context, azuretemplate map[string]interface{}) (*string, error) {
		return nil, nil
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

	configInterface := mock_config.NewMockInterface(gmc)
	clusterUpgrader := mock_cluster.NewMockUpgrader(gmc)

	c := clusterUpgrader.EXPECT().GetNameserversFromVnet(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	c = configInterface.EXPECT().InvalidateSecrets().Return(nil).After(c)
	c = configInterface.EXPECT().Generate(gomock.Any(), false).Return(nil).After(c)
	expectUpdate(cs, clusterUpgrader, &c)

	p := &plugin{
		upgraderFactory: func(ctx context.Context, log *logrus.Entry, cs *api.OpenShiftManagedCluster, initializeStorageClients, disableKeepAlives bool, testConfig api.TestConfig) (cluster.Upgrader, error) {
			return clusterUpgrader, nil
		},
		configInterfaceFactory: func(cs *api.OpenShiftManagedCluster) (config.Interface, error) {
			return configInterface, nil
		},
		log:          logrus.NewEntry(logrus.StandardLogger()),
		pluginConfig: &pluginapi.Config{},
		now:          time.Now,
	}

	if err := p.RotateClusterSecrets(nil, cs, deployer); err != nil {
		t.Errorf("plugin.RotateClusterSecrets error = %v", err)
	}
}

func TestForceUpdate(t *testing.T) {
	gmc := gomock.NewController(t)
	defer gmc.Finish()

	deployer := func(ctx context.Context, azuretemplate map[string]interface{}) (*string, error) {
		return nil, nil
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

	c := clusterUpgrader.EXPECT().ResetUpdateBlob().Return(nil)
	c = clusterUpgrader.EXPECT().BackupCluster(nil, gomock.Any()).After(c)
	expectUpdate(cs, clusterUpgrader, &c)

	p := &plugin{
		upgraderFactory: func(ctx context.Context, log *logrus.Entry, cs *api.OpenShiftManagedCluster, initializeStorageClients, disableKeepAlives bool, testConfig api.TestConfig) (cluster.Upgrader, error) {
			return clusterUpgrader, nil
		},
		log: logrus.NewEntry(logrus.StandardLogger()),
		now: time.Now,
	}

	if err := p.ForceUpdate(nil, cs, deployer); err != nil {
		t.Errorf("plugin.ForceUpdate error = %v", err)
	}
}

func TestListClusterVMs(t *testing.T) {
	gmc := gomock.NewController(t)
	defer gmc.Finish()

	clusterUpgrader := mock_cluster.NewMockUpgrader(gmc)

	clusterUpgrader.EXPECT().ListVMHostnames(nil).Return(nil, nil)

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
	cs := &api.OpenShiftManagedCluster{
		Properties: api.Properties{
			AzProfile: api.AzProfile{
				SubscriptionID: "foo",
				ResourceGroup:  "bar",
			},
			NetworkProfile: api.NetworkProfile{
				Nameservers: []string{"1.2.3.4"},
			},
			RefreshCluster: to.BoolPtr(false),
		},
	}
	logger := logrus.NewEntry(logrus.StandardLogger())
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clusterUpgrader := mock_cluster.NewMockUpgrader(gmc)

			scaleset, instanceID, err := names.GetScaleSetNameAndInstanceID(tt.hostname)
			if err != nil {
				t.Fatal(err)
			}
			c := clusterUpgrader.EXPECT().GetNameserversFromVnet(nil, logger, cs.Properties.AzProfile.SubscriptionID, cs.Properties.AzProfile.ResourceGroup).Return([]string{"1.2.3.4"}, nil)
			c = clusterUpgrader.EXPECT().Reimage(nil, scaleset, instanceID).Return(nil).After(c)

			if tt.isMaster {
				c = clusterUpgrader.EXPECT().WaitForReadyMaster(nil, tt.hostname).Return(nil).After(c)
			} else {
				c = clusterUpgrader.EXPECT().WaitForReadyWorker(nil, tt.hostname).Return(nil).After(c)
			}

			p := &plugin{
				upgraderFactory: func(ctx context.Context, log *logrus.Entry, cs *api.OpenShiftManagedCluster, initializeStorageClients, disableKeepAlives bool, testConfig api.TestConfig) (cluster.Upgrader, error) {
					return clusterUpgrader, nil
				},
				log: logger,
			}

			if err := p.Reimage(nil, cs, tt.hostname); err != nil {
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

func errorsContains(errs []error, substr string) bool {
	for _, err := range errs {
		if strings.Contains(err.Error(), substr) {
			return true
		}
	}
	return false
}

func TestValidateUpdateBlock(t *testing.T) {
	ctx := context.Background()
	gmc := gomock.NewController(t)
	defer gmc.Finish()
	const currentVersion = "current"
	props := api.Properties{
		AzProfile: api.AzProfile{
			SubscriptionID: "foo",
			ResourceGroup:  "bar",
		},
		NetworkProfile: api.NetworkProfile{
			Nameservers: []string{"1.2.3.4"},
		},
		RefreshCluster: to.BoolPtr(true),
	}
	logger := logrus.NewEntry(logrus.StandardLogger())
	clusterUpgrader := mock_cluster.NewMockUpgrader(gmc)
	p := &plugin{
		log:          logger,
		pluginConfig: &pluginapi.Config{PluginVersion: currentVersion},
		configInterfaceFactory: func(cs *api.OpenShiftManagedCluster) (c config.Interface, err error) {
			if cs.Config.PluginVersion != currentVersion {
				err = fmt.Errorf("bad")
			}
			return
		},
		upgraderFactory: func(ctx context.Context, log *logrus.Entry, cs *api.OpenShiftManagedCluster, initializeStorageClients, disableKeepAlives bool, testConfig api.TestConfig) (cluster.Upgrader, error) {
			return clusterUpgrader, nil
		},
	}
	tests := []struct {
		name      string
		cs        *api.OpenShiftManagedCluster
		expectErr bool
	}{
		{
			name: "current",
			cs: &api.OpenShiftManagedCluster{
				Config: api.Config{
					PluginVersion: p.pluginConfig.PluginVersion,
				},
				Properties: props,
			},
		},
		{
			name: "old",
			cs: &api.OpenShiftManagedCluster{
				Config: api.Config{
					PluginVersion: "old",
				},
				Properties: props,
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := p.Validate(ctx, tt.cs, tt.cs, true)
			if tt.expectErr != errorsContains(errs, "cannot be updated by resource provider") {
				if tt.expectErr {
					t.Errorf("expected %s cluster to fail update validation", tt.name)
				} else {
					t.Errorf("expected %s cluster to pass update validation %v", tt.name, errs)
				}
			}
		})
	}
}

func TestCheckIfClusterWillRefresh(t *testing.T) {
	ctx := context.Background()
	gmc := gomock.NewController(t)
	defer gmc.Finish()
	logger := logrus.NewEntry(logrus.StandardLogger())
	origNS := []string{"1.2.3.4"}
	oldCs := &api.OpenShiftManagedCluster{
		Properties: api.Properties{
			NetworkProfile: api.NetworkProfile{
				Nameservers: origNS,
			},
		},
	}
	clusterUpgrader := mock_cluster.NewMockUpgrader(gmc)
	p := &plugin{
		log: logger,
		upgraderFactory: func(ctx context.Context, log *logrus.Entry, cs *api.OpenShiftManagedCluster, initializeStorageClients, disableKeepAlives bool, testConfig api.TestConfig) (cluster.Upgrader, error) {
			return clusterUpgrader, nil
		},
	}
	tests := []struct {
		name      string
		f         func(*api.OpenShiftManagedCluster)
		expectErr bool
	}{
		{
			name: "no refresh, no nameserver change",
			f: func(oc *api.OpenShiftManagedCluster) {
				clusterUpgrader.EXPECT().GetNameserversFromVnet(ctx, logger, oldCs.Properties.AzProfile.SubscriptionID, oldCs.Properties.AzProfile.ResourceGroup).Return(origNS, nil)
				oc.Properties.RefreshCluster = to.BoolPtr(false)
			},
		},
		{
			name: "no refresh, nameserver change",
			f: func(oc *api.OpenShiftManagedCluster) {
				clusterUpgrader.EXPECT().GetNameserversFromVnet(ctx, logger, oldCs.Properties.AzProfile.SubscriptionID, oldCs.Properties.AzProfile.ResourceGroup).Return([]string{"3.2.4.5"}, nil)
				oc.Properties.RefreshCluster = to.BoolPtr(false)
			},
			expectErr: true,
		},
		{
			name: "refresh, no nameserver change",
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.RefreshCluster = to.BoolPtr(true)
			},
		},
		{
			name: "refresh, nameserver change",
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.RefreshCluster = to.BoolPtr(true)
			},
		},
		{
			name: "nil refresh, nameserver change",
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.RefreshCluster = nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := oldCs.DeepCopy()
			tt.f(cs)
			err := p.checkIfClusterWillRefresh(ctx, cs)
			if tt.expectErr != (err != nil) {
				if tt.expectErr {
					t.Errorf("%s] expected cluster to fail update validation %v", tt.name, err)
				} else {
					t.Errorf("%s] expected cluster to pass update validation", tt.name)
				}
			}
		})
	}
}
