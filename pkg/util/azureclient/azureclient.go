package azureclient

//go:generate go get github.com/golang/mock/gomock
//go:generate go install github.com/golang/mock/mockgen
//go:generate mockgen -destination=../../util/mocks/mock_$GOPACKAGE/azureclient.go github.com/openshift/openshift-azure/pkg/util/$GOPACKAGE Client,VirtualMachineScaleSetsClient,VirtualMachineScaleSetVMsClient,VirtualMachineScaleSetExtensionsClient,ApplicationsClient,MarketPlaceAgreementsClient,DeploymentsClient,AccountsClient
//go:generate gofmt -s -l -w ../../util/mocks/mock_$GOPACKAGE/azureclient.go
//go:generate goimports -local=github.com/openshift/openshift-azure -e -w ../../util/mocks/mock_$GOPACKAGE/azureclient.go
//go:generate mockgen -destination=../../util/mocks/mock_$GOPACKAGE/mock_storage/storage.go github.com/openshift/openshift-azure/pkg/util/$GOPACKAGE/storage Client,BlobStorageClient,Container,Blob
//go:generate gofmt -s -l -w ../../util/mocks/mock_$GOPACKAGE/mock_storage/storage.go
//go:generate goimports -local=github.com/openshift/openshift-azure -e -w ../../util/mocks/mock_$GOPACKAGE/mock_storage/storage.go

import (
	"context"
	"net/http"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"

	"github.com/openshift/openshift-azure/pkg/api"
)

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
			if acceptLanguages != nil {
				for _, language := range acceptLanguages {
					r.Header.Add("Accept-Language", language)
				}
			}
			return r, nil
		})
	}
}

func NewAuthorizer(clientID, clientSecret, tenantID string) (autorest.Authorizer, error) {
	return auth.NewClientCredentialsConfig(clientID, clientSecret, tenantID).Authorizer()
}

func NewAuthorizerFromContext(ctx context.Context) (autorest.Authorizer, error) {
	return NewAuthorizer(ctx.Value(api.ContextKeyClientID).(string), ctx.Value(api.ContextKeyClientSecret).(string), ctx.Value(api.ContextKeyTenantID).(string))
}
