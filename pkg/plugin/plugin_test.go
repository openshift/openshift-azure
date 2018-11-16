package plugin

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/arm"
	"github.com/openshift/openshift-azure/pkg/config"
	"github.com/openshift/openshift-azure/pkg/log"
	"github.com/openshift/openshift-azure/pkg/util/fixtures"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_arm"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_cluster"
)

func NewPluginWithFakeUpgrader(ctrl *gomock.Controller, entry *logrus.Entry, pluginConfig *api.PluginConfig) api.Plugin {
	log.New(entry)
	return &plugin{
		entry:           entry,
		config:          *pluginConfig,
		clusterUpgrader: mock_cluster.NewMockUpgrader(ctrl),
		configGenerator: config.NewSimpleGenerator(pluginConfig),
		armGenerator:    arm.NewSimpleGenerator(entry, pluginConfig),
	}
}

func TestMerge(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	var config = api.PluginConfig{
		SyncImage:       "sync:latest",
		AcceptLanguages: []string{"en-us"},
	}
	newCluster := fixtures.NewTestOpenShiftCluster()
	p := NewPluginWithFakeUpgrader(mockCtrl, logrus.NewEntry(logrus.New()), &config)
	oldCluster := fixtures.NewTestOpenShiftCluster()

	newCluster.Config = nil
	newCluster.Properties.AgentPoolProfiles = nil
	newCluster.Properties.RouterProfiles = nil
	newCluster.Properties.ServicePrincipalProfile = nil
	newCluster.Properties.AzProfile = nil
	newCluster.Properties.AuthProfile = nil
	newCluster.Properties.FQDN = ""

	// make old cluster go through plugin first
	armTemplate := testPluginRun(p, oldCluster, nil, t)
	if !hasResourceType(armTemplate, "Microsoft.Network/networkSecurityGroups") {
		t.Fatalf("networkSecurityGroups should be applied during cluster creation")
	}

	// should fix all of the items removed above and we should
	// be able to run through the entire plugin process.
	p.MergeConfig(context.Background(), newCluster, oldCluster)

	if newCluster.Config == nil {
		t.Errorf("new cluster config should be merged")
	}
	if len(newCluster.Properties.AgentPoolProfiles) == 0 {
		t.Errorf("new cluster agent pool profiles should be merged")
	}
	if newCluster.Properties.NetworkProfile == nil {
		t.Errorf("new cluster network profile should be merged")
	}
	if len(newCluster.Properties.RouterProfiles) == 0 {
		t.Errorf("new cluster router profiles should be merged")
	}
	if newCluster.Properties.ServicePrincipalProfile == nil {
		t.Errorf("new cluster service principal profile should be merged")
	}
	if newCluster.Properties.AzProfile == nil {
		t.Errorf("new cluster az profile should be merged")
	}
	if newCluster.Properties.AuthProfile == nil {
		t.Errorf("new cluster auth profile should be merged")
	}
	if newCluster.Properties.FQDN == "" {
		t.Errorf("new cluster fqdn should be merged")
	}

	armTemplate = testPluginRun(p, newCluster, oldCluster, t)
	if hasResourceType(armTemplate, "Microsoft.Network/networkSecurityGroups") {
		t.Fatalf("networkSecurityGroups should not be applied during cluster upgrade")
	}
}

func hasResourceType(armTemplate map[string]interface{}, resType string) bool {
	for _, res := range armTemplate["resources"].([]interface{}) {
		if res.(map[string]interface{})["type"] == resType {
			return true
		}
	}
	return false
}

func testPluginRun(p api.Plugin, newCluster *api.OpenShiftManagedCluster, oldCluster *api.OpenShiftManagedCluster, t *testing.T) (armTemplate map[string]interface{}) {
	if errs := p.Validate(context.Background(), newCluster, oldCluster, false); len(errs) != 0 {
		t.Fatalf("error validating: %s", spew.Sdump(errs))
	}

	if err := p.GenerateConfig(context.Background(), newCluster); err != nil {
		t.Fatalf("error generating config for arm generate test: %s", spew.Sdump(err))
	}

	azuretemplate, err := p.GenerateARM(context.Background(), newCluster, oldCluster != nil)
	if err != nil {
		t.Fatalf("error generating arm: %s", spew.Sdump(err))
	}
	if len(azuretemplate) == 0 {
		t.Errorf("no arm was generated")
	}
	return azuretemplate
}

func TestGenerateARM(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	testData := map[string]interface{}{"test": "data"}
	mockGen := mock_arm.NewMockGenerator(mockCtrl)
	mockGen.EXPECT().Generate(nil, nil, true).Return(testData, nil)
	p := &plugin{
		armGenerator: mockGen,
	}
	log.New(logrus.NewEntry(logrus.New()))

	got, err := p.GenerateARM(nil, nil, true)
	if err != nil {
		t.Errorf("plugin.GenerateARM() error = %v", err)
		return
	}
	if !reflect.DeepEqual(got, testData) {
		t.Errorf("plugin.GenerateARM() = %v, want %v", got, testData)
	}
}

func TestCreateOrUpdate(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockUp := mock_cluster.NewMockUpgrader(mockCtrl)
	tests := []struct {
		name     string
		isUpdate bool
		wantErr  bool
		errStep  api.PluginStep
	}{
		{
			name:     "deploy",
			isUpdate: false,
			wantErr:  false,
		},
		{
			name:     "update",
			isUpdate: true,
			wantErr:  false,
		},
		{
			name:     "deploy: deploy error",
			isUpdate: false,
			wantErr:  true,
			errStep:  api.PluginStepDeploy,
		},
		{
			name:     "deploy: initialize error",
			isUpdate: false,
			wantErr:  true,
			errStep:  api.PluginStepInitialize,
		},
		{
			name:     "deploy: openshift healthz error",
			isUpdate: false,
			wantErr:  true,
			errStep:  api.PluginStepWaitForWaitForOpenShiftAPI,
		},
		{
			name:     "deploy: nodes error",
			isUpdate: false,
			wantErr:  true,
			errStep:  api.PluginStepWaitForNodes,
		},
		{
			name:     "update: drain error",
			isUpdate: true,
			wantErr:  true,
			errStep:  api.PluginStepDrain,
		},
		{
			name:     "update: deploy error",
			isUpdate: true,
			wantErr:  true,
			errStep:  api.PluginStepDeploy,
		},
		{
			name:     "update: initialize error",
			isUpdate: true,
			wantErr:  true,
			errStep:  api.PluginStepInitialize,
		},
		{
			name:     "update: nodes error",
			isUpdate: true,
			wantErr:  true,
			errStep:  api.PluginStepWaitForNodes,
		},
		{
			name:     "update in place: list VMs error",
			isUpdate: true,
			wantErr:  true,
			errStep:  api.PluginStepUpdateInPlaceListVMs,
		},
		{
			name:     "update in place: sort masters error",
			isUpdate: true,
			wantErr:  true,
			errStep:  api.PluginStepUpdateInPlaceSortMasters,
		},
		{
			name:     "update in place: read blob error",
			isUpdate: true,
			wantErr:  true,
			errStep:  api.PluginStepUpdateInPlaceReadBlob,
		},
		{
			name:     "update in place: drain error",
			isUpdate: true,
			wantErr:  true,
			errStep:  api.PluginStepUpdateInPlaceDrain,
		},
		{
			name:     "update in place: deallocate error",
			isUpdate: true,
			wantErr:  true,
			errStep:  api.PluginStepUpdateInPlaceDeallocate,
		},
		{
			name:     "update in place: update VMs error",
			isUpdate: true,
			wantErr:  true,
			errStep:  api.PluginStepUpdateInPlaceUpdateVMs,
		},
		{
			name:     "update in place: reimage error",
			isUpdate: true,
			wantErr:  true,
			errStep:  api.PluginStepUpdateInPlaceReimage,
		},
		{
			name:     "update in place: start error",
			isUpdate: true,
			wantErr:  true,
			errStep:  api.PluginStepUpdateInPlaceStart,
		},
		{
			name:     "update in place: wait for ready error",
			isUpdate: true,
			wantErr:  true,
			errStep:  api.PluginStepUpdateInPlaceWaitForReady,
		},
		{
			name:     "update in place: update blob error",
			isUpdate: true,
			wantErr:  true,
			errStep:  api.PluginStepUpdateInPlaceUpdateBlob,
		},
		{
			name:     "update plus one: list VMs error",
			isUpdate: true,
			wantErr:  true,
			errStep:  api.PluginStepUpdatePlusOneListVMs,
		},
		{
			name:     "update plus one: read blob error",
			isUpdate: true,
			wantErr:  true,
			errStep:  api.PluginStepUpdatePlusOneReadBlob,
		},
		{
			name:     "update plus one: wait for ready error",
			isUpdate: true,
			wantErr:  true,
			errStep:  api.PluginStepUpdatePlusOneWaitForReady,
		},
		{
			name:     "update plus one: update blob error",
			isUpdate: true,
			wantErr:  true,
			errStep:  api.PluginStepUpdatePlusOneUpdateBlob,
		},
		{
			name:     "update plus one: delete VMs error",
			isUpdate: true,
			wantErr:  true,
			errStep:  api.PluginStepUpdatePlusOneDeleteVMs,
		},
		{
			name:     "waitforinfra: daemon error",
			isUpdate: true,
			wantErr:  true,
			errStep:  api.PluginStepWaitForInfraDaemonSets,
		},
		{
			name:     "waitforinfra: deployments error",
			isUpdate: true,
			wantErr:  true,
			errStep:  api.PluginStepWaitForInfraDeployments,
		},
		{
			name:     "ConsoleHealth: error",
			isUpdate: true,
			wantErr:  true,
			errStep:  api.PluginStepWaitForConsoleHealth,
		},
	}
	for _, tt := range tests {
		if tt.wantErr {
			err := &api.PluginError{Err: fmt.Errorf("test error"), Step: tt.errStep}
			switch tt.errStep {
			case api.PluginStepDrain, api.PluginStepDeploy, api.PluginStepInitialize,
				api.PluginStepWaitForConsoleHealth, api.PluginStepWaitForNodes,
				api.PluginStepUpdateInPlaceListVMs, api.PluginStepUpdateInPlaceSortMasters,
				api.PluginStepUpdateInPlaceReadBlob, api.PluginStepUpdateInPlaceDrain,
				api.PluginStepUpdateInPlaceDeallocate, api.PluginStepUpdateInPlaceUpdateVMs,
				api.PluginStepUpdateInPlaceReimage, api.PluginStepUpdateInPlaceStart,
				api.PluginStepUpdateInPlaceWaitForReady, api.PluginStepUpdateInPlaceUpdateBlob,
				api.PluginStepUpdatePlusOneListVMs, api.PluginStepUpdatePlusOneReadBlob,
				api.PluginStepUpdatePlusOneWaitForReady, api.PluginStepUpdatePlusOneUpdateBlob,
				api.PluginStepUpdatePlusOneDeleteVMs:
				if tt.isUpdate {
					mockUp.EXPECT().Update(nil, nil, nil, nil).Return(err)
				} else {
					mockUp.EXPECT().Deploy(nil, nil, nil, nil).Return(err)
				}
			case api.PluginStepWaitForWaitForOpenShiftAPI:
				if tt.isUpdate {
					mockUp.EXPECT().Update(nil, nil, nil, nil).Return(nil)
				} else {
					mockUp.EXPECT().Deploy(nil, nil, nil, nil).Return(nil)
				}
				mockUp.EXPECT().WaitForInfraServices(nil, nil).Return(nil)
				mockUp.EXPECT().HealthCheck(nil, nil).Return(err)
			case api.PluginStepWaitForInfraDaemonSets, api.PluginStepWaitForInfraDeployments:
				if tt.isUpdate {
					mockUp.EXPECT().Update(nil, nil, nil, nil).Return(nil)
				} else {
					mockUp.EXPECT().Deploy(nil, nil, nil, nil).Return(nil)
				}
				mockUp.EXPECT().WaitForInfraServices(nil, nil).Return(err)
			}
		} else {
			if tt.isUpdate {
				mockUp.EXPECT().Update(nil, nil, nil, nil).Return(nil)
			} else {
				mockUp.EXPECT().Deploy(nil, nil, nil, nil).Return(nil)
			}
			mockUp.EXPECT().WaitForInfraServices(nil, nil).Return(nil)
			mockUp.EXPECT().HealthCheck(nil, nil).Return(nil)
		}
		p := &plugin{
			clusterUpgrader: mockUp,
		}
		log.New(logrus.NewEntry(logrus.New()))
		if err := p.CreateOrUpdate(nil, nil, nil, tt.isUpdate, nil); (err != nil) != tt.wantErr {
			t.Errorf("plugin.CreateOrUpdate(%s) error = %v, wantErr %v", tt.name, err, tt.wantErr)
		}
	}
}
