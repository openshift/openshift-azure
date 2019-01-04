package plugin

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_arm"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_cluster"
	"github.com/openshift/openshift-azure/test/util/tls"
)

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
			errStep:  api.PluginStepUpdateMasterAgentPoolListVMs,
		},
		{
			name:     "update in place: read blob error",
			isUpdate: true,
			wantErr:  true,
			errStep:  api.PluginStepUpdateMasterAgentPoolReadBlob,
		},
		{
			name:     "update in place: drain error",
			isUpdate: true,
			wantErr:  true,
			errStep:  api.PluginStepUpdateMasterAgentPoolDrain,
		},
		{
			name:     "update in place: deallocate error",
			isUpdate: true,
			wantErr:  true,
			errStep:  api.PluginStepUpdateMasterAgentPoolDeallocate,
		},
		{
			name:     "update in place: update VMs error",
			isUpdate: true,
			wantErr:  true,
			errStep:  api.PluginStepUpdateMasterAgentPoolUpdateVMs,
		},
		{
			name:     "update in place: reimage error",
			isUpdate: true,
			wantErr:  true,
			errStep:  api.PluginStepUpdateMasterAgentPoolReimage,
		},
		{
			name:     "update in place: start error",
			isUpdate: true,
			wantErr:  true,
			errStep:  api.PluginStepUpdateMasterAgentPoolStart,
		},
		{
			name:     "update in place: wait for ready error",
			isUpdate: true,
			wantErr:  true,
			errStep:  api.PluginStepUpdateMasterAgentPoolWaitForReady,
		},
		{
			name:     "update in place: update blob error",
			isUpdate: true,
			wantErr:  true,
			errStep:  api.PluginStepUpdateMasterAgentPoolUpdateBlob,
		},
		{
			name:     "update plus one: list VMs error",
			isUpdate: true,
			wantErr:  true,
			errStep:  api.PluginStepUpdateWorkerAgentPoolListVMs,
		},
		{
			name:     "update plus one: read blob error",
			isUpdate: true,
			wantErr:  true,
			errStep:  api.PluginStepUpdateWorkerAgentPoolReadBlob,
		},
		{
			name:     "update plus one: wait for ready error",
			isUpdate: true,
			wantErr:  true,
			errStep:  api.PluginStepUpdateWorkerAgentPoolWaitForReady,
		},
		{
			name:     "update plus one: update blob error",
			isUpdate: true,
			wantErr:  true,
			errStep:  api.PluginStepUpdateWorkerAgentPoolUpdateBlob,
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
		mockGen := mock_arm.NewMockGenerator(mockCtrl)
		mockGen.EXPECT().Generate(nil, nil, "", tt.isUpdate, gomock.Any()).Return(nil, nil)
		if tt.wantErr {
			err := &api.PluginError{Err: fmt.Errorf("test error"), Step: tt.errStep}
			switch tt.errStep {
			case api.PluginStepDeploy, api.PluginStepInitialize,
				api.PluginStepWaitForConsoleHealth, api.PluginStepWaitForNodes,
				api.PluginStepUpdateMasterAgentPoolListVMs,
				api.PluginStepUpdateMasterAgentPoolReadBlob, api.PluginStepUpdateMasterAgentPoolDrain,
				api.PluginStepUpdateMasterAgentPoolDeallocate, api.PluginStepUpdateMasterAgentPoolUpdateVMs,
				api.PluginStepUpdateMasterAgentPoolReimage, api.PluginStepUpdateMasterAgentPoolStart,
				api.PluginStepUpdateMasterAgentPoolWaitForReady, api.PluginStepUpdateMasterAgentPoolUpdateBlob,
				api.PluginStepUpdateWorkerAgentPoolListVMs, api.PluginStepUpdateWorkerAgentPoolReadBlob,
				api.PluginStepUpdateWorkerAgentPoolWaitForReady, api.PluginStepUpdateWorkerAgentPoolUpdateBlob:
				if tt.isUpdate {
					mockUp.EXPECT().Update(nil, nil, nil, nil, gomock.Any()).Return(err)
				} else {
					mockUp.EXPECT().Deploy(nil, nil, nil, nil, gomock.Any()).Return(err)
				}
			case api.PluginStepWaitForWaitForOpenShiftAPI:
				if tt.isUpdate {
					mockUp.EXPECT().Update(nil, nil, nil, nil, gomock.Any()).Return(nil)
				} else {
					mockUp.EXPECT().Deploy(nil, nil, nil, nil, gomock.Any()).Return(nil)
				}
				mockUp.EXPECT().WaitForInfraServices(nil, nil).Return(nil)
				mockUp.EXPECT().HealthCheck(nil, nil).Return(err)
			case api.PluginStepWaitForInfraDaemonSets, api.PluginStepWaitForInfraDeployments:
				if tt.isUpdate {
					mockUp.EXPECT().Update(nil, nil, nil, nil, gomock.Any()).Return(nil)
				} else {
					mockUp.EXPECT().Deploy(nil, nil, nil, nil, gomock.Any()).Return(nil)
				}
				mockUp.EXPECT().WaitForInfraServices(nil, nil).Return(err)
			}
		} else {
			if tt.isUpdate {
				mockUp.EXPECT().Update(nil, nil, nil, nil, gomock.Any()).Return(nil)
			} else {
				mockUp.EXPECT().Deploy(nil, nil, nil, nil, gomock.Any()).Return(nil)
			}
			mockUp.EXPECT().WaitForInfraServices(nil, nil).Return(nil)
			mockUp.EXPECT().HealthCheck(nil, nil).Return(nil)
		}
		mockUp.EXPECT().CreateClients(nil, nil).Return(nil)
		p := &plugin{
			clusterUpgrader: mockUp,
			armGenerator:    mockGen,
			log:             logrus.NewEntry(logrus.StandardLogger()),
		}
		if err := p.CreateOrUpdate(nil, nil, tt.isUpdate, nil); (err != nil) != tt.wantErr {
			t.Errorf("plugin.CreateOrUpdate(%s) error = %v, wantErr %v", tt.name, err, tt.wantErr)
		}
	}
}

func TestNewPlugin(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	tests := map[string]struct {
		f            func(*api.PluginConfig)
		expectedErrs []error
	}{
		"empty syncImage": {
			f: func(p *api.PluginConfig) {
				p.SyncImage = ""
			},
			expectedErrs: []error{
				errors.New(`syncImage cannot be empty`),
			},
		},
		"empty imagePullSecret": {
			f: func(p *api.PluginConfig) {
				p.GenevaConfig.ImagePullSecret = []byte{}
			},
			expectedErrs: []error{
				errors.New(`imagePullSecret cannot be empty`),
			},
		},
		"empty LoggingSector": {
			f: func(p *api.PluginConfig) {
				p.GenevaConfig.LoggingSector = ""
			},
			expectedErrs: []error{
				errors.New(`loggingSector cannot be empty`),
			},
		},
		"empty LoggingImage": {
			f: func(p *api.PluginConfig) {
				p.GenevaConfig.LoggingImage = ""
			},
			expectedErrs: []error{
				errors.New(`loggingImage cannot be empty`),
			},
		},
		"empty tdAgentImage": {
			f: func(p *api.PluginConfig) {
				p.GenevaConfig.TDAgentImage = ""
			},
			expectedErrs: []error{
				errors.New(`tdAgentImage cannot be empty`),
			},
		},
		"nil loggingKey": {
			f: func(p *api.PluginConfig) {
				p.GenevaConfig.LoggingKey = nil
			},
			expectedErrs: []error{
				errors.New(`loggingKey cannot be nil`),
			},
		},
		"nil loggingCert": {
			f: func(p *api.PluginConfig) {
				p.GenevaConfig.LoggingCert = nil
			},
			expectedErrs: []error{
				errors.New(`loggingCert cannot be nil`),
			},
		},
		"empty loggingControlPlaneAccount": {
			f: func(p *api.PluginConfig) {
				p.GenevaConfig.LoggingControlPlaneAccount = ""
			},
			expectedErrs: []error{
				errors.New(`loggingControlPlaneAccount cannot be empty`),
			},
		},
		"nil metricsCert": {
			f: func(p *api.PluginConfig) {
				p.GenevaConfig.MetricsCert = nil
			},
			expectedErrs: []error{
				errors.New(`metricsCert cannot be nil`),
			},
		},
		"nil metricsKey": {
			f: func(p *api.PluginConfig) {
				p.GenevaConfig.MetricsKey = nil
			},
			expectedErrs: []error{
				errors.New(`metricsKey cannot be nil`),
			},
		},
		"empty metricsBridge": {
			f: func(p *api.PluginConfig) {
				p.GenevaConfig.MetricsBridge = ""
			},
			expectedErrs: []error{
				errors.New(`metricsBridge cannot be empty`),
			},
		},
		"empty statsdImage": {
			f: func(p *api.PluginConfig) {
				p.GenevaConfig.StatsdImage = ""
			},
			expectedErrs: []error{
				errors.New(`statsdImage cannot be empty`),
			},
		},
		"empty metricsEndpoint": {
			f: func(p *api.PluginConfig) {
				p.GenevaConfig.MetricsEndpoint = ""
			},
			expectedErrs: []error{
				errors.New(`metricsEndpoint cannot be empty`),
			},
		},
		"empty metricsAccount": {
			f: func(p *api.PluginConfig) {
				p.GenevaConfig.MetricsAccount = ""
			},
			expectedErrs: []error{
				errors.New(`metricsAccount cannot be empty`),
			},
		},
	}

	for name, test := range tests {
		pCfg, err := getDummyPluginConfig()
		if err != nil {
			t.Fatal(err)
		}

		test.f(pCfg)
		_, errs := NewPlugin(log, pCfg)
		if !reflect.DeepEqual(errs, test.expectedErrs) {
			t.Errorf("%s expected errors:", name)
			for _, err := range test.expectedErrs {
				t.Errorf("\t%v", err)
			}
			t.Error("received errors:")
			for _, err := range errs {
				t.Errorf("\t%v", err)
			}
		}
	}
}

func getDummyPluginConfig() (*api.PluginConfig, error) {
	// dummy config
	return &api.PluginConfig{
		SyncImage: "syncImage",
		GenevaConfig: api.GenevaConfig{
			ImagePullSecret: []byte("imagePullSecret"),

			LoggingSector:              "loggingSector",
			LoggingAccount:             "loggingAccount",
			LoggingNamespace:           "loggingNamespace",
			LoggingControlPlaneAccount: "loggingControlPlaneAccount",
			LoggingCert:                tls.GetDummyCertificate(),
			LoggingKey:                 tls.GetDummyPrivateKey(),
			TDAgentImage:               "tdAgentImage",
			LoggingImage:               "loggingImage",

			MetricsCert:     tls.GetDummyCertificate(),
			MetricsKey:      tls.GetDummyPrivateKey(),
			MetricsBridge:   "metricsBridge",
			StatsdImage:     "statsdImage",
			MetricsAccount:  "metricsAccount",
			MetricsEndpoint: "metricsEndpoint",
		},
	}, nil
}

func TestRecoverEtcdCluster(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	testData := map[string]interface{}{"test": "data"}
	testDataWithBackup := map[string]interface{}{"test": "backup"}
	mockGen := mock_arm.NewMockGenerator(mockCtrl)
	mockUp := mock_cluster.NewMockUpgrader(mockCtrl)
	gomock.InOrder(
		mockGen.EXPECT().Generate(nil, nil, gomock.Any(), true, gomock.Any()).Return(testDataWithBackup, nil),
		mockGen.EXPECT().Generate(nil, nil, gomock.Any(), true, gomock.Any()).Return(testData, nil),
	)
	mockUp.EXPECT().CreateClients(nil, nil).Times(2).Return(nil)
	mockUp.EXPECT().EtcdRestore(nil, nil, testDataWithBackup, nil).Return(nil)
	mockUp.EXPECT().Update(nil, nil, testData, nil, gomock.Any()).Return(nil)
	mockUp.EXPECT().WaitForInfraServices(nil, nil).Return(nil)
	mockUp.EXPECT().HealthCheck(nil, nil).Return(nil)
	p := &plugin{
		clusterUpgrader: mockUp,
		armGenerator:    mockGen,
		log:             logrus.NewEntry(logrus.StandardLogger()),
	}

	if err := p.RecoverEtcdCluster(nil, nil, nil, "test-backup"); err != nil {
		t.Errorf("plugin.RecoverEtcdCluster error = %v", err)
	}
}
