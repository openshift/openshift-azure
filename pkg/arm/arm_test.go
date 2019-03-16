package arm

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/jsonpath"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_azureclient/mock_storage"
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
	c = container.EXPECT().GetBlobReference("worker-startup").Return(blob).After(c)
	c = blob.EXPECT().GetSASURI(gomock.Any()).Return("", nil).After(c)

	sg := simpleGenerator{
		storageClient: storageClient,
	}

	armtemplate, err := sg.Generate(context.Background(), cs, "", false, "")
	if err != nil {
		t.Fatal(err)
	}

	if len(jsonpath.MustCompile("$.resources[?(@.type='Microsoft.Network/networkSecurityGroups')]").Get(armtemplate)) != 2 {
		t.Error("expected to find two networkSecurityGroups on create")
	}

	armtemplate, err = sg.Generate(context.Background(), cs, "", true, "")
	if err != nil {
		t.Fatal(err)
	}

	if len(jsonpath.MustCompile("$.resources[?(@.type='Microsoft.Network/networkSecurityGroups')]").Get(armtemplate)) != 1 {
		t.Error("expected to find one networkSecurityGroup on update")
	}
}
