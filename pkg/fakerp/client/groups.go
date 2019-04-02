package client

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

// CreateResourceGroup creates a resource group and returns whether the
// resource group was actually created or not and any error encountered.
func CreateResourceGroup(conf *Config) (bool, error) {
	authorizer, err := azureclient.NewAuthorizer(conf.ClientID, conf.ClientSecret, conf.TenantID, "")
	if err != nil {
		return false, err
	}
	gc := resources.NewGroupsClient(conf.SubscriptionID)
	gc.Authorizer = authorizer

	if _, err := gc.Get(context.Background(), conf.ResourceGroup); err == nil {
		return false, nil
	}

	tags := map[string]*string{
		"now": to.StringPtr(fmt.Sprintf("%d", time.Now().Unix())),
		"ttl": to.StringPtr("72h"),
	}
	if conf.ResourceGroupTTL != "" {
		if _, err := time.ParseDuration(conf.ResourceGroupTTL); err != nil {
			return false, fmt.Errorf("invalid ttl provided: %q - %v", conf.ResourceGroupTTL, err)
		}
		tags["ttl"] = &conf.ResourceGroupTTL
	}

	rg := resources.Group{Location: &conf.Region, Tags: tags}
	_, err = gc.CreateOrUpdate(context.Background(), conf.ResourceGroup, rg)
	return true, err
}
