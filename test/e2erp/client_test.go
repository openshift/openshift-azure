//+build e2erp

package e2erp

import (
	"fmt"
	"os"
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

type testClient struct {
	gc    resources.GroupsClient
	rpc   sdk.OpenShiftManagedClustersClient
	ssc   azureclient.VirtualMachineScaleSetsClient
	ssvmc azureclient.VirtualMachineScaleSetVMsClient
	ssec  azureclient.VirtualMachineScaleSetExtensionsClient

	resourceGroup string
	location      string
}

func newTestClient(resourceGroup string) *testClient {
	authorizer, err := azureclient.NewAuthorizer(os.Getenv("AZURE_CLIENT_ID"), os.Getenv("AZURE_CLIENT_SECRET"), os.Getenv("AZURE_TENANT_ID"))
	if err != nil {
		panic(err)
	}
	subID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	gc := resources.NewGroupsClient(subID)
	gc.Authorizer = authorizer

	var rpc sdk.OpenShiftManagedClustersClient
	focus := []byte(config.GinkgoConfig.FocusString)
	switch {
	case fakeRe.Match(focus):
		fmt.Println("Creating a cluster using the fake resource provider")
		// rpc = sdk.NewOpenShiftManagedClustersClientWithBaseURI("http://localhost:8080", subID)
		panic("not implemented yet")
	case realRe.Match(focus):
		fmt.Println("Creating a cluster using the real resource provider")
		rpc = sdk.NewOpenShiftManagedClustersClient(subID)
	default:
		panic(fmt.Sprintf("invalid focus %q - need to -ginkgo.focus=\\[Fake\\] or -ginkgo.focus=\\[Real\\]", config.GinkgoConfig.FocusString))
	}
	rpc.Authorizer = authorizer
	ssc := azureclient.NewVirtualMachineScaleSetsClient(subID, authorizer, []string{"en-us"})
	ssvmc := azureclient.NewVirtualMachineScaleSetVMsClient(subID, authorizer, []string{"en-us"})
	ssec := azureclient.NewVirtualMachineScaleSetExtensionsClient(subID, authorizer, []string{"en-us"})

	return &testClient{
		gc:            gc,
		rpc:           rpc,
		ssc:           ssc,
		ssvmc:         ssvmc,
		ssec:          ssec,
		resourceGroup: resourceGroup,
		location:      os.Getenv("AZURE_REGION"),
	}
}
