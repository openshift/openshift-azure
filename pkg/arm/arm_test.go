package arm

import (
	"context"
	"testing"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/jsonpath"
	"github.com/openshift/openshift-azure/test/util/tls"
)

func TestGenerate(t *testing.T) {
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

	var sg simpleGenerator
	armtemplate, err := sg.Generate(context.Background(), cs, "", false, "", map[string]string{"config": "", "registry": ""})
	if err != nil {
		t.Fatal(err)
	}

	if len(jsonpath.MustCompile("$.resources[?(@.type='Microsoft.Network/networkSecurityGroups')]").Get(armtemplate)) != 2 {
		t.Error("expected to find two networkSecurityGroups on create")
	}

	armtemplate, err = sg.Generate(context.Background(), cs, "", true, "", map[string]string{"config": "", "registry": ""})
	if err != nil {
		t.Fatal(err)
	}

	if len(jsonpath.MustCompile("$.resources[?(@.type='Microsoft.Network/networkSecurityGroups')]").Get(armtemplate)) != 1 {
		t.Error("expected to find one networkSecurityGroup on update")
	}
}
