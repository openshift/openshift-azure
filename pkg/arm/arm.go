package arm

//go:generate go get github.com/go-bindata/go-bindata/go-bindata
//go:generate go-bindata -nometadata -pkg $GOPACKAGE -prefix data data/...
//go:generate gofmt -s -l -w bindata.go
//go:generate go get github.com/golang/mock/gomock
//go:generate go install github.com/golang/mock/mockgen
//go:generate mockgen -destination=../util/mocks/mock_$GOPACKAGE/arm.go -package=mock_$GOPACKAGE -source arm.go
//go:generate gofmt -s -l -w ../util/mocks/mock_$GOPACKAGE/arm.go
//go:generate goimports -local=github.com/openshift/openshift-azure -e -w ../util/mocks/mock_$GOPACKAGE/arm.go

import (
	"context"
	"encoding/json"
	"time"

	azstorage "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/storage"
)

type Generator interface {
	Generate(ctx context.Context, cs *api.OpenShiftManagedCluster, backupBlob string, isUpdate bool, suffix string) (map[string]interface{}, error)
}

type simpleGenerator struct {
	testConfig     api.TestConfig
	accountsClient azureclient.AccountsClient
	storageClient  storage.Client
}

var _ Generator = &simpleGenerator{}

type Template struct {
	Schema         string        `json:"$schema,omitempty"`
	ContentVersion string        `json:"contentVersion,omitempty"`
	Parameters     struct{}      `json:"parameters,omitempty"`
	Variables      struct{}      `json:"variables,omitempty"`
	Resources      []interface{} `json:"resources,omitempty"`
	Outputs        struct{}      `json:"outputs,omitempty"`
}

// NewSimpleGenerator create a new SimpleGenerator
func NewSimpleGenerator(ctx context.Context, cs *api.OpenShiftManagedCluster, testConfig api.TestConfig) (Generator, error) {
	authorizer, err := azureclient.GetAuthorizerFromContext(ctx, api.ContextKeyClientAuthorizer)
	if err != nil {
		return nil, err
	}

	g := &simpleGenerator{
		testConfig:     testConfig,
		accountsClient: azureclient.NewAccountsClient(ctx, cs.Properties.AzProfile.SubscriptionID, authorizer),
	}

	if cs.Config.ConfigStorageAccountKey == "" {
		keys, err := g.accountsClient.ListKeys(ctx, cs.Properties.AzProfile.ResourceGroup, cs.Config.ConfigStorageAccount)
		if err != nil {
			return nil, err
		}
		cs.Config.ConfigStorageAccountKey = *(*keys.Keys)[0].Value
	}

	g.storageClient, err = storage.NewClient(cs.Config.ConfigStorageAccount, cs.Config.ConfigStorageAccountKey, storage.DefaultBaseURL, storage.DefaultAPIVersion, true)
	if err != nil {
		return nil, err
	}

	return g, nil
}

func GetConfigSASURI(storageClient storage.Client, app *api.AgentPoolProfile) (string, error) {
	now := time.Now().Add(-time.Hour)

	bsc := storageClient.GetBlobService()
	c := bsc.GetContainerReference("config") // TODO: should be using consts, need to merge packages
	var b storage.Blob
	switch app.Role {
	case api.AgentPoolProfileRoleMaster:
		b = c.GetBlobReference("master-startup")
	default:
		b = c.GetBlobReference("worker-startup")
	}
	return b.GetSASURI(azstorage.BlobSASOptions{
		BlobServiceSASPermissions: azstorage.BlobServiceSASPermissions{
			Read: true,
		},
		SASOptions: azstorage.SASOptions{
			APIVersion: "2015-04-05",
			Start:      now,
			Expiry:     now.AddDate(5, 0, 0),
			UseHTTPS:   true,
		},
	})
}

func (g *simpleGenerator) Generate(ctx context.Context, cs *api.OpenShiftManagedCluster, backupBlob string, isUpdate bool, suffix string) (map[string]interface{}, error) {
	t := Template{
		Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
		ContentVersion: "1.0.0.0",
		Resources: []interface{}{
			vnet(cs),
			ipAPIServer(cs),
			lbAPIServer(cs),
			storageAccount(cs.Config.RegistryStorageAccount, cs, map[string]*string{
				"type": to.StringPtr("registry"),
			}),
			storageAccount(cs.Config.AzureFileStorageAccount, cs, map[string]*string{
				"type": to.StringPtr("storage"),
			}),
			nsgMaster(cs),
		},
	}
	if !isUpdate {
		t.Resources = append(t.Resources, ipOutbound(cs), lbKubernetes(cs), nsgWorker(cs))
	}
	for _, app := range cs.Properties.AgentPoolProfiles {
		if app.Role == api.AgentPoolProfileRoleMaster || !isUpdate {
			blobURI, err := GetConfigSASURI(g.storageClient, &app)
			if err != nil {
				return nil, err
			}

			vmss, err := Vmss(cs, &app, blobURI, backupBlob, suffix, g.testConfig)
			if err != nil {
				return nil, err
			}
			t.Resources = append(t.Resources, vmss)
		}
	}

	b, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}

	var azuretemplate map[string]interface{}
	err = json.Unmarshal(b, &azuretemplate)
	if err != nil {
		return nil, err
	}

	FixupAPIVersions(azuretemplate)
	FixupDepends(cs.Properties.AzProfile.SubscriptionID, cs.Properties.AzProfile.ResourceGroup, azuretemplate)

	return azuretemplate, nil
}
