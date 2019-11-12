package arm

//go:generate go get github.com/go-bindata/go-bindata/go-bindata
//go:generate go-bindata -nometadata -pkg $GOPACKAGE -prefix data data/...
//go:generate gofmt -s -l -w bindata.go

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	armconst "github.com/openshift/openshift-azure/pkg/arm/constants"
	"github.com/openshift/openshift-azure/pkg/util/arm"
	"github.com/openshift/openshift-azure/pkg/util/resourceid"
)

type simpleGenerator struct {
	testConfig api.TestConfig
	log        *logrus.Entry
	cs         *api.OpenShiftManagedCluster
}

func New(ctx context.Context, log *logrus.Entry, cs *api.OpenShiftManagedCluster, testConfig api.TestConfig) *simpleGenerator {
	return &simpleGenerator{
		testConfig: testConfig,
		log:        log,
		cs:         cs,
	}
}

func (g *simpleGenerator) Generate(ctx context.Context, backupBlob string, isUpdate bool, suffix string) (map[string]interface{}, error) {
	t := arm.Template{
		Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
		ContentVersion: "1.0.0.0",
		Resources: []interface{}{
			g.ipAPIServer(),
			g.lbAPIServer(),
			g.storageAccount(g.cs.Config.RegistryStorageAccount, map[string]*string{
				"type": to.StringPtr("registry"),
			}),
			g.storageAccount(g.cs.Config.AzureFileStorageAccount, map[string]*string{
				"type": to.StringPtr("storage"),
			}),
			g.nsgMaster(),
		},
	}
	if g.cs.Properties.PrivateAPIServer {
		t.Resources = append(t.Resources, g.ilbAPIServer())
	}
	if !isUpdate {
		t.Resources = append(t.Resources, g.vnet(), g.ipOutbound(), g.lbKubernetes(), g.nsgWorker())
	}

	for _, app := range g.cs.Properties.AgentPoolProfiles {
		if app.Role == api.AgentPoolProfileRoleMaster || !isUpdate {
			vmss, err := g.Vmss(&app, backupBlob, suffix)
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

	err = arm.FixupAPIVersions(azuretemplate, versionMap)
	if err != nil {
		return nil, err
	}

	ignoreMap := map[string]struct{}{}
	if isUpdate {
		vnetID := resourceid.ResourceID(g.cs.Properties.AzProfile.SubscriptionID, g.cs.Properties.AzProfile.ResourceGroup, "Microsoft.Network/virtualNetworks", armconst.VnetName)
		ignoreMap[vnetID] = struct{}{}
	}
	arm.FixupDepends(g.cs.Properties.AzProfile.SubscriptionID, g.cs.Properties.AzProfile.ResourceGroup, azuretemplate, ignoreMap)

	// HACK: Current SDK version is v24. Private link support comes into the version v28+.
	// To use PLS/PE we need to set configurables on vnet and create PLS itself
	// search for PrivateEndpointNetworkPolicies in https://raw.githubusercontent.com/Azure/azure-sdk-for-go/master/services/network/mgmt/2019-04-01/network/models.go
	// To have these configs in our we need to modify the object after we generate them
	arm.FixupSDKMismatch(azuretemplate)

	return azuretemplate, nil
}

func (g *simpleGenerator) Hash(app *api.AgentPoolProfile) ([]byte, error) {
	hash := sha256.New()

	// the hash is invariant of name, suffix, count...
	appCopy := *app
	appCopy.Count = 0
	appCopy.Name = ""

	// ...and also the SAS URIs
	cs := g.cs.DeepCopy()
	cs.Config.MasterStartupSASURI = ""
	cs.Config.WorkerStartupSASURI = ""

	vmss, err := vmss(cs, &appCopy, "", "", g.testConfig) // TODO: backupBlob is rather a layering violation here
	if err != nil {
		return nil, err
	}

	err = json.NewEncoder(hash).Encode(vmss)
	if err != nil {
		return nil, err
	}

	if g.testConfig.DebugHashFunctions {
		b, err := json.Marshal(vmss)
		if err != nil {
			return nil, err
		}
		err = ioutil.WriteFile(fmt.Sprintf("vmss-%s-%d", app.Role, time.Now().UnixNano()), b, 0666)
		if err != nil {
			return nil, err
		}
	}

	return hash.Sum(nil), nil
}
