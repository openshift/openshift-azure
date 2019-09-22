package client

import (
	"context"
	"fmt"
	"time"

	azresources "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

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

	if ready, _ := checkResourceGroupIsReady(ctx, gc, conf.ResourceGroup); ready {
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

	if _, err = gc.CreateOrUpdate(ctx, conf.ResourceGroup, azresources.Group{Location: &conf.Region, Tags: tags}); err != nil {
		return err
	}
	log.Infof("waiting for successful provision of resource group %s", conf.ResourceGroup)
	return wait.PollImmediate(5*time.Second, 5*time.Minute, func() (bool, error) { return checkResourceGroupIsReady(ctx, gc, conf.ResourceGroup) })
}

func checkResourceGroupIsReady(ctx context.Context, gc resources.GroupsClient, resourceGroup string) (bool, error) {
	g, err := gc.Get(ctx, resourceGroup)
	if err != nil {
		return false, err
	}
	return *g.Properties.ProvisioningState == "Succeeded", nil
}
