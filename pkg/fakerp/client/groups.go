package client

import (
	"context"
	"fmt"
	"time"

	azresources "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/resources"
)

// EnsureResourceGroup creates a resource group and returns whether the
// resource group was actually created or not and any error encountered.
func EnsureResourceGroup(log *logrus.Entry, conf *Config) error {
	authorizer, err := azureclient.NewAuthorizer(conf.ClientID, conf.ClientSecret, conf.TenantID, "")
	if err != nil {
		return err
	}
	ctx := context.Background()
	gc := resources.NewGroupsClient(ctx, log, conf.SubscriptionID, authorizer)

	if _, err := gc.Get(ctx, conf.ResourceGroup); err == nil {
		return nil
	}

	tags := map[string]*string{
		"now": to.StringPtr(fmt.Sprintf("%d", time.Now().Unix())),
		"ttl": to.StringPtr("72h"),
	}
	if conf.ResourceGroupTTL != "" {
		if _, err := time.ParseDuration(conf.ResourceGroupTTL); err != nil {
			return fmt.Errorf("invalid ttl provided: %q - %v", conf.ResourceGroupTTL, err)
		}
		tags["ttl"] = &conf.ResourceGroupTTL
	}

	_, err = gc.CreateOrUpdate(ctx, conf.ResourceGroup, azresources.Group{Location: &conf.Region, Tags: tags})
	return err
}
