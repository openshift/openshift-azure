package azure

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/onsi/ginkgo/config"
	"github.com/sirupsen/logrus"

	externalapi "github.com/openshift/openshift-azure/pkg/api/2018-09-30-preview/api"
	adminapi "github.com/openshift/openshift-azure/pkg/api/admin/api"
	realfakerp "github.com/openshift/openshift-azure/pkg/fakerp"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

// rpFocus represents the supported RP APIs which e2e tests use to create their azure clients,
// The client will be configured to work either against the real, fake or admin apis
type rpFocus string

var (
	adminRpFocus = rpFocus(regexp.QuoteMeta("[Admin]"))
	fakeRpFocus  = rpFocus(regexp.QuoteMeta("[Fake]"))
	realRpFocus  = rpFocus(regexp.QuoteMeta("[Real]"))
)

func (tf rpFocus) match(focusString string) bool {
	return strings.Contains(focusString, string(tf))
}

// Client is the main controller for azure client objects
type Client struct {
	Accounts                         azureclient.AccountsClient
	Applications                     azureclient.ApplicationsClient
	OpenShiftManagedClusters         externalapi.OpenShiftManagedClustersClient
	OpenShiftManagedClustersAdmin    adminapi.OpenShiftManagedClustersClient
	VirtualMachineScaleSets          azureclient.VirtualMachineScaleSetsClient
	VirtualMachineScaleSetExtensions azureclient.VirtualMachineScaleSetExtensionsClient
	VirtualMachineScaleSetVMs        azureclient.VirtualMachineScaleSetVMsClient
	Resources                        azureclient.ResourcesClient
}

// NewClientFromEnvironment creates a new azure client from environment variables
func NewClientFromEnvironment() (*Client, error) {
	authorizer, err := azureclient.NewAuthorizerFromEnvironment()
	if err != nil {
		return nil, err
	}
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")

	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	log := logrus.NewEntry(logrus.StandardLogger())

	conf, err := realfakerp.NewConfig(log, true)
	if err != nil {
		return nil, err
	}

	var rpURL string
	focus := config.GinkgoConfig.FocusString
	switch {
	case adminRpFocus.match(focus), fakeRpFocus.match(focus):
		fmt.Println("configuring the fake resource provider")
		rpURL = realfakerp.StartServer(log, conf, realfakerp.LocalHttpAddr)
	case realRpFocus.match(focus):
		fmt.Println("configuring the real resource provider")
		rpURL = externalapi.DefaultBaseURI
	default:
		panic(fmt.Sprintf("invalid focus %q - need to -ginkgo.focus=\\[Admin\\], -ginkgo.focus=\\[Fake\\] or -ginkgo.focus=\\[Real\\]", config.GinkgoConfig.FocusString))
	}

	rpc := externalapi.NewOpenShiftManagedClustersClientWithBaseURI(rpURL, subscriptionID)
	rpc.Authorizer = authorizer

	rpcAdmin := adminapi.NewOpenShiftManagedClustersClientWithBaseURI(rpURL, subscriptionID)
	rpcAdmin.Authorizer = authorizer

	return &Client{
		Accounts:                         azureclient.NewAccountsClient(subscriptionID, authorizer, nil),
		Applications:                     azureclient.NewApplicationsClient(subscriptionID, authorizer, nil),
		OpenShiftManagedClusters:         rpc,
		OpenShiftManagedClustersAdmin:    rpcAdmin,
		VirtualMachineScaleSets:          azureclient.NewVirtualMachineScaleSetsClient(subscriptionID, authorizer, nil),
		VirtualMachineScaleSetExtensions: azureclient.NewVirtualMachineScaleSetExtensionsClient(subscriptionID, authorizer, nil),
		VirtualMachineScaleSetVMs:        azureclient.NewVirtualMachineScaleSetVMsClient(subscriptionID, authorizer, nil),
		Resources:                        azureclient.NewResourcesClient(subscriptionID, authorizer, nil),
	}, nil
}
