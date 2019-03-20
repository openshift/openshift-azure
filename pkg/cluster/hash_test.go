package cluster

import (
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/startup"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_startup"
	"github.com/openshift/openshift-azure/test/util/populate"
)

func TestHashScaleSet(t *testing.T) {
	gmc := gomock.NewController(t)
	defer gmc.Finish()

	mockStartup := mock_startup.NewMockInterface(gmc)
	c := mockStartup.EXPECT().Hash(api.AgentPoolProfileRoleMaster).Return(nil, nil).Times(2)
	c = mockStartup.EXPECT().Hash(api.AgentPoolProfileRoleCompute).Return(nil, nil).Times(2).After(c)

	prepare := func(v reflect.Value) {
		switch v.Interface().(type) {
		case []api.IdentityProvider:
			// set the Provider to AADIdentityProvider
			v.Set(reflect.ValueOf([]api.IdentityProvider{{Provider: &api.AADIdentityProvider{Kind: "AADIdentityProvider"}}}))
		}
	}

	var cs api.OpenShiftManagedCluster
	populate.Walk(&cs, prepare)
	cs.Properties.AgentPoolProfiles = []api.AgentPoolProfile{
		{
			Role: api.AgentPoolProfileRoleMaster,
		},
		{
			Role:   api.AgentPoolProfileRoleCompute,
			VMSize: api.StandardD2sV3,
		},
	}

	for _, role := range []api.AgentPoolProfileRole{api.AgentPoolProfileRoleMaster, api.AgentPoolProfileRoleCompute} {
		h := hasher{
			startupFactory: func(*logrus.Entry, *api.OpenShiftManagedCluster) (startup.Interface, error) { return mockStartup, nil },
		}
		baseline, err := h.HashScaleSet(&cs, &api.AgentPoolProfile{
			Role: role,
		})
		if err != nil {
			t.Errorf("%s: unexpected error: %v", role, err)
		}
		csCopy := cs.DeepCopy()
		csCopy.Config.MasterStartupSASURI = "foo"
		csCopy.Config.WorkerStartupSASURI = "foo"
		second, err := h.HashScaleSet(csCopy, &api.AgentPoolProfile{
			Name:  "foo",
			Role:  role,
			Count: 1,
		})
		if err != nil {
			t.Errorf("%s: unexpected error: %v", role, err)
		}
		if !reflect.DeepEqual(baseline, second) {
			t.Errorf("%s: expected:\n%#v\ngot:\n%#v", role, baseline, second)
		}
	}
}
