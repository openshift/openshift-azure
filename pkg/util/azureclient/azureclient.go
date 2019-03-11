package azureclient

//go:generate go get github.com/golang/mock/gomock
//go:generate go install github.com/golang/mock/mockgen
//go:generate mockgen -destination=../../util/mocks/mock_$GOPACKAGE/azureclient.go github.com/openshift/openshift-azure/pkg/util/$GOPACKAGE AccountsClient,ActivityLogsClient,ApplicationsClient,Client,DeploymentsClient,GroupsClient,KeyVaultClient,MarketPlaceAgreementsClient,RBACApplicationsClient,RBACGroupsClient,RecordSetsClient,ResourcesClient,ServicePrincipalsClient,VaultMgmtClient,VirtualMachineScaleSetExtensionsClient,VirtualMachineScaleSetsClient,VirtualMachineScaleSetVMsClient,VirtualNetworksClient,VirtualNetworksPeeringsClient,ZonesClient
//go:generate gofmt -s -l -w ../../util/mocks/mock_$GOPACKAGE/azureclient.go
//go:generate goimports -local=github.com/openshift/openshift-azure -e -w ../../util/mocks/mock_$GOPACKAGE/azureclient.go
//go:generate mockgen -destination=../../util/mocks/mock_$GOPACKAGE/mock_storage/storage.go github.com/openshift/openshift-azure/pkg/util/$GOPACKAGE/storage Client,BlobStorageClient,Container,Blob
//go:generate gofmt -s -l -w ../../util/mocks/mock_$GOPACKAGE/mock_storage/storage.go
//go:generate goimports -local=github.com/openshift/openshift-azure -e -w ../../util/mocks/mock_$GOPACKAGE/mock_storage/storage.go

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
	"time"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"

	"github.com/openshift/openshift-azure/pkg/api"
)

const KeyVaultEndpoint = "https://vault.azure.net" // beware of the leopard

// Client returns the Client
type Client interface {
	Client() autorest.Client
}

func addAcceptLanguages(acceptLanguages []string) autorest.PrepareDecorator {
	return func(p autorest.Preparer) autorest.Preparer {
		return autorest.PreparerFunc(func(r *http.Request) (*http.Request, error) {
			r, err := p.Prepare(r)
			if err != nil {
				return r, err
			}
			for _, language := range acceptLanguages {
				r.Header.Add("Accept-Language", language)
			}
			return r, nil
		})
	}
}

type loggingSender struct {
	autorest.Sender
}

func (ls *loggingSender) Do(req *http.Request) (*http.Response, error) {
	b, _ := httputil.DumpRequestOut(req, true)
	fmt.Printf("%s\n\n", string(b))
	resp, err := ls.Sender.Do(req)
	if resp != nil {
		b, _ = httputil.DumpResponse(resp, true)
		fmt.Printf("%s\n\n", string(b))
	}
	return resp, err
}

func setupClient(ctx context.Context, client *autorest.Client, authorizer autorest.Authorizer) {
	// if context does not provide languages (sync pod, tests) - use default
	var languages []string
	if ctx.Value(api.ContextAcceptLanguages) != nil {
		languages = ctx.Value(api.ContextAcceptLanguages).([]string)
	}

	client.Authorizer = authorizer
	client.RequestInspector = addAcceptLanguages(languages)
	client.PollingDelay = 10 * time.Second
	// client.Sender = &loggingSender{client.Sender}
}

func NewAuthorizer(clientID, clientSecret, tenantID, resource string) (autorest.Authorizer, error) {
	if resource == azure.PublicCloud.KeyVaultEndpoint {
		return nil, fmt.Errorf("resource azure.PublicCloud.KeyVaultEndpoint doesn't work: use azureclient.KeyVaultEndpoint")
	}

	config := auth.NewClientCredentialsConfig(clientID, clientSecret, tenantID)
	if resource != "" {
		config.Resource = resource
	}
	return config.Authorizer()
}

func GetAuthorizerFromContext(ctx context.Context, key api.ContextKey) (autorest.Authorizer, error) {
	authorizer, ok := ctx.Value(key).(autorest.Authorizer)
	if !ok {
		return nil, fmt.Errorf("failed to get authorizer, key %s not found within context", key)
	}
	return authorizer, nil
}

func NewAuthorizerFromEnvironment(resource string) (autorest.Authorizer, error) {
	return NewAuthorizer(os.Getenv("AZURE_CLIENT_ID"), os.Getenv("AZURE_CLIENT_SECRET"), os.Getenv("AZURE_TENANT_ID"), resource)
}
