package arm

import (
	"context"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/jsonpath"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_azureclient/mock_storage"
	"github.com/openshift/openshift-azure/test/util/populate"
	"github.com/openshift/openshift-azure/test/util/tls"
)

func TestGenerate(t *testing.T) {
	gmc := gomock.NewController(t)
	defer gmc.Finish()

	cs := &api.OpenShiftManagedCluster{
		Properties: api.Properties{
			AgentPoolProfiles: []api.AgentPoolProfile{
				{Role: api.AgentPoolProfileRoleCompute, Name: "compute"},
			},
		},
		Config: api.Config{
			SSHKey: tls.GetDummyPrivateKey(),
			Certificates: api.CertificateConfig{
				Ca:            api.CertKeyPair{Cert: tls.GetDummyCertificate(), Key: tls.GetDummyPrivateKey()},
				NodeBootstrap: api.CertKeyPair{Cert: tls.GetDummyCertificate(), Key: tls.GetDummyPrivateKey()},
			},
		},
	}

	storageClient := mock_storage.NewMockClient(gmc)
	bsc := mock_storage.NewMockBlobStorageClient(gmc)
	container := mock_storage.NewMockContainer(gmc)
	blob := mock_storage.NewMockBlob(gmc)
	c := storageClient.EXPECT().GetBlobService().Return(bsc)
	c = bsc.EXPECT().GetContainerReference("config").Return(container).After(c)
	c = container.EXPECT().GetBlobReference("master-startup").Return(blob).After(c)
	c = blob.EXPECT().GetSASURI(gomock.Any()).Return("", nil).After(c)
	c = container.EXPECT().GetBlobReference("worker-startup").Return(blob).After(c)
	c = blob.EXPECT().GetSASURI(gomock.Any()).Return("", nil).After(c)
	c = storageClient.EXPECT().GetBlobService().Return(bsc).After(c)
	c = bsc.EXPECT().GetContainerReference("config").Return(container).After(c)
	c = container.EXPECT().GetBlobReference("master-startup").Return(blob).After(c)
	c = blob.EXPECT().GetSASURI(gomock.Any()).Return("", nil).After(c)
	c = container.EXPECT().GetBlobReference("worker-startup").Return(blob).After(c)
	c = blob.EXPECT().GetSASURI(gomock.Any()).Return("", nil).After(c)

	sg := simpleGenerator{
		storageClient: storageClient,
		cs:            cs,
	}

	armtemplate, err := sg.Generate(context.Background(), "", false, "")
	if err != nil {
		t.Fatal(err)
	}

	if len(jsonpath.MustCompile("$.resources[?(@.type='Microsoft.Network/networkSecurityGroups')]").Get(armtemplate)) != 2 {
		t.Error("expected to find two networkSecurityGroups on create")
	}

	armtemplate, err = sg.Generate(context.Background(), "", true, "")
	if err != nil {
		t.Fatal(err)
	}

	if len(jsonpath.MustCompile("$.resources[?(@.type='Microsoft.Network/networkSecurityGroups')]").Get(armtemplate)) != 1 {
		t.Error("expected to find one networkSecurityGroup on update")
	}
}

func TestHash(t *testing.T) {
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
		sg := simpleGenerator{
			cs: &cs,
		}
		baseline, err := sg.Hash(&api.AgentPoolProfile{
			Role: role,
		})
		if err != nil {
			t.Errorf("%s: unexpected error: %v", role, err)
		}
		sg.cs = cs.DeepCopy()
		sg.cs.Config.MasterStartupSASURI = "foo"
		sg.cs.Config.WorkerStartupSASURI = "foo"
		second, err := sg.Hash(&api.AgentPoolProfile{
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
