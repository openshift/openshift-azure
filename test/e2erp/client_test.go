//+build e2erp

package e2erp

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/ghodss/yaml"
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

func (t *testClient) setup(manifest string) error {
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Minute)
	// TODO: tags: now=$(date +%s) ttl=$ttl
	if _, err := t.gc.CreateOrUpdate(ctx, t.resourceGroup, resources.Group{Location: &t.location}); err != nil {
		return err
	}

	in, err := ioutil.ReadFile(manifest)
	if err != nil {
		return err
	}
	var oc sdk.OpenShiftManagedCluster
	if err := yaml.Unmarshal(in, &oc); err != nil {
		return err
	}

	future, err := t.rpc.CreateOrUpdate(ctx, t.resourceGroup, t.resourceGroup, oc)
	if err != nil {
		return err
	}
	if err := future.WaitForCompletionRef(ctx, t.rpc.Client); err != nil {
		return err
	}
	oc, err = future.Result(t.rpc)
	if err != nil {
		return err
	}
	if *oc.OpenShiftManagedClusterProperties.ProvisioningState != "Succeeded" {
		return fmt.Errorf("failed to provision cluster: %s", *oc.OpenShiftManagedClusterProperties.ProvisioningState)
	}
	return nil
}

func (t *testClient) teardown() error {
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Minute)
	future, err := t.rpc.Delete(ctx, t.resourceGroup, t.resourceGroup)
	if err != nil {
		return err
	}
	if err := future.WaitForCompletionRef(ctx, t.rpc.Client); err != nil {
		return err
	}
	resp, err := future.Result(t.rpc)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}
	_, err = t.gc.Delete(ctx, t.resourceGroup)
	return err
}
