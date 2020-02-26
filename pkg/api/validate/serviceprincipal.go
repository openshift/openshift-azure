package validate

import (
	"fmt"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/dgrijalva/jwt-go"
	"k8s.io/apimachinery/pkg/util/wait"

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

	// get a token, retrying only on AADSTS700016 errors (slow AAD propagation).
	// see: https://github.com/Azure/ARO-RP/blob/0af036fcd242c116a15bfd3dc2f4ac01b9f64534/pkg/api/validate/openshiftcluster_validatedynamic.go#L95-L108
	wait.PollImmediate(time.Second, 10*time.Second, func() (bool, error) {
		err = token.EnsureFresh()
		return err == nil || !strings.Contains(err.Error(), "AADSTS700016"), nil
	})
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
