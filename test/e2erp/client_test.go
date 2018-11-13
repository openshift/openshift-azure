//+build e2erp

package e2erp

import (
	"fmt"
	"regexp"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/onsi/ginkgo/config"

	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	sdk "github.com/openshift/openshift-azure/pkg/util/azureclient/osa-go-sdk/services/containerservice/mgmt/2018-09-30-preview/containerservice"
)

var (
	fakeRe = regexp.MustCompile("Fake")
	realRe = regexp.MustCompile("Real")
)

type AzureConfig struct {
	SubscriptionID  string   `envconfig:"AZURE_SUBSCRIPTION_ID" required:"true"`
	TenantID        string   `envconfig:"AZURE_TENANT_ID" required:"true"`
	Region          string   `envconfig:"AZURE_REGION" required:"true"`
	ClientID        string   `envconfig:"AZURE_CLIENT_ID" required:"true"`
	ClientSecret    string   `envconfig:"AZURE_CLIENT_SECRET" required:"true"`
	ResourceGroup   string   `envconfig:"RESOURCEGROUP" required:"true"`
	AcceptLanguages []string `envconfig:"ACCEPT_LANGUAGES" default:"en-us"`
}

type testClient struct {
	gc    resources.GroupsClient
	rpc   sdk.OpenShiftManagedClustersClient
	ssc   azureclient.VirtualMachineScaleSetsClient
	ssvmc azureclient.VirtualMachineScaleSetVMsClient
	ssec  azureclient.VirtualMachineScaleSetExtensionsClient
	appsc azureclient.ApplicationsClient

	resourceGroup string
	location      string
}

func newTestClient(conf AzureConfig) *testClient {
	authorizer, err := azureclient.NewAuthorizer(conf.ClientID, conf.ClientSecret, conf.TenantID)
	if err != nil {
		panic(err)
	}
	subID := conf.SubscriptionID
	gc := resources.NewGroupsClient(subID)
	gc.Authorizer = authorizer

	var rpc sdk.OpenShiftManagedClustersClient
	focus := []byte(config.GinkgoConfig.FocusString)
	switch {
	case fakeRe.Match(focus):
		fmt.Println("Creating a cluster using the fake resource provider")
		rpc = sdk.NewOpenShiftManagedClustersClientWithBaseURI("http://localhost:8080", subID)
	case realRe.Match(focus):
		fmt.Println("Creating a cluster using the real resource provider")
		rpc = sdk.NewOpenShiftManagedClustersClient(subID)
	default:
		panic(fmt.Sprintf("invalid focus %q - need to -ginkgo.focus=\\[Fake\\] or -ginkgo.focus=\\[Real\\]", config.GinkgoConfig.FocusString))
	}
	rpc.Authorizer = authorizer
	ssc := azureclient.NewVirtualMachineScaleSetsClient(subID, authorizer, conf.AcceptLanguages)
	ssvmc := azureclient.NewVirtualMachineScaleSetVMsClient(subID, authorizer, conf.AcceptLanguages)
	ssec := azureclient.NewVirtualMachineScaleSetExtensionsClient(subID, authorizer, conf.AcceptLanguages)
	appsc := azureclient.NewApplicationsClient(subID, authorizer, conf.AcceptLanguages)

	return &testClient{
		gc:            gc,
		rpc:           rpc,
		ssc:           ssc,
		ssvmc:         ssvmc,
		ssec:          ssec,
		appsc:         appsc,
		resourceGroup: conf.ResourceGroup,
		location:      conf.Region,
	}
}
