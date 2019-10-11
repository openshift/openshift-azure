package arm

import (
	"context"
	"reflect"
	"testing"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/jsonpath"
	"github.com/openshift/openshift-azure/test/util/populate"
	"github.com/openshift/openshift-azure/test/util/tls"
)

func TestGenerate(t *testing.T) {
	sg := simpleGenerator{
		cs: &api.OpenShiftManagedCluster{
			Properties: api.Properties{
				AgentPoolProfiles: []api.AgentPoolProfile{
					{Role: api.AgentPoolProfileRoleCompute, Name: "compute"},
				},
			},
			Config: api.Config{
				SSHKey: tls.DummyPrivateKey,
				Certificates: api.CertificateConfig{
					Ca:            api.CertKeyPair{Cert: tls.DummyCertificate, Key: tls.DummyPrivateKey},
					NodeBootstrap: api.CertKeyPair{Cert: tls.DummyCertificate, Key: tls.DummyPrivateKey},
				},
				Images: api.ImageConfig{
					GenevaImagePullSecret: populate.DummyImagePullSecret("acr.azure.io"),
					ImagePullSecret:       populate.DummyImagePullSecret("registry.redhat.io"),
				},
			},
		},
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
	cs.Config.Images.GenevaImagePullSecret = populate.DummyImagePullSecret("acr.azure.io")
	cs.Config.Images.ImagePullSecret = populate.DummyImagePullSecret("registry.redhat.io")

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
