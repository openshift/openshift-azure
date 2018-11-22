//+build e2e

package azure

import (
	"context"
	"fmt"
	"regexp"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/onsi/ginkgo/config"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/fakerp"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	sdk "github.com/openshift/openshift-azure/pkg/util/azureclient/osa-go-sdk/services/containerservice/mgmt/2018-09-30-preview/containerservice"
	"github.com/openshift/openshift-azure/pkg/util/log"
)

var (
	fakeRe = regexp.MustCompile("Fake")
	realRe = regexp.MustCompile("Real")
)

type Client struct {
	gc    resources.GroupsClient
	rpc   sdk.OpenShiftManagedClustersClient
	ssc   azureclient.VirtualMachineScaleSetsClient
	ssvmc azureclient.VirtualMachineScaleSetVMsClient
	ssec  azureclient.VirtualMachineScaleSetExtensionsClient
	appsc azureclient.ApplicationsClient

	resourceGroup string
	location      string
	log           *logrus.Entry
	ctx           context.Context
}

func NewClient() *Client {
	logrus.SetLevel(log.SanitizeLogLevel("Debug"))
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	log := logrus.NewEntry(logrus.StandardLogger())
	conf, err := fakerp.NewConfig(log)
	if err != nil {
		panic(err)
	}
	log = logrus.WithFields(logrus.Fields{"location": conf.Region, "resourceGroup": conf.ResourceGroup})

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

	ctx := context.Background()
	ctx = context.WithValue(ctx, api.ContextKeyClientID, conf.ClientID)
	ctx = context.WithValue(ctx, api.ContextKeyClientSecret, conf.ClientSecret)
	ctx = context.WithValue(ctx, api.ContextKeyTenantID, conf.TenantID)

	return &Client{
		gc:            gc,
		rpc:           rpc,
		ssc:           ssc,
		ssvmc:         ssvmc,
		ssec:          ssec,
		appsc:         appsc,
		resourceGroup: conf.ResourceGroup,
		location:      conf.Region,
		ctx:           ctx,
		log:           log,
	}
}
