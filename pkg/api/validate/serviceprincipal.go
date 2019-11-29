package validate

import (
	"fmt"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/dgrijalva/jwt-go"

	"github.com/openshift/openshift-azure/pkg/api"
)

type azureClaim struct {
	Roles []string `json:"roles,omitempty"`
}

func (*azureClaim) Valid() error {
	return fmt.Errorf("unimplemented")
}

type RoleLister interface {
	ListAADApplicationRoles(*api.AADIdentityProvider) ([]string, error)
}

type SimpleRoleLister struct{}

func (SimpleRoleLister) ListAADApplicationRoles(aad *api.AADIdentityProvider) (roles []string, err error) {
	oauthConfig, err := adal.NewOAuthConfig(azure.PublicCloud.ActiveDirectoryEndpoint, aad.TenantID)
	if err != nil {
		return nil, err
	}

	token, err := adal.NewServicePrincipalToken(*oauthConfig, aad.ClientID, aad.Secret, azure.PublicCloud.GraphEndpoint)
	if err != nil {
		return nil, err
	}

	err = token.EnsureFresh()
	if err != nil {
		return
	}

	p := &jwt.Parser{}
	c := &azureClaim{}
	_, _, err = p.ParseUnverified(token.OAuthToken(), c)
	if err != nil {
		return
	}

	return c.Roles, nil
}

type DummyRoleLister struct {
	roles []string
}

func (d DummyRoleLister) ListAADApplicationRoles(aad *api.AADIdentityProvider) ([]string, error) {
	return d.roles, nil
}
