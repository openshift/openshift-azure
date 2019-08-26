package fakerp

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	internalapi "github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/fakerp/client"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/resources"
)

type clients struct {
	aadMgr      *aadManager
	dnsMgr      *dnsManager
	vaultMgr    *vaultManager
	groupClient resources.GroupsClient
}

func newClients(ctx context.Context, log *logrus.Entry, cs *api.OpenShiftManagedCluster, testConfig api.TestConfig, conf *client.Config) (*clients, error) {
	var err error
	c := &clients{}
	c.aadMgr, err = newAADManager(ctx, log, cs, testConfig)
	if err != nil {
		return nil, err
	}
	c.dnsMgr, err = newDNSManager(ctx, log, cs.Properties.AzProfile.SubscriptionID, conf.DNSResourceGroup, conf.DNSDomain)
	if err != nil {
		return nil, err
	}
	c.vaultMgr, err = newVaultManager(ctx, log, cs.Properties.AzProfile.SubscriptionID, cs.Properties.AzProfile.TenantID)
	if err != nil {
		return nil, err
	}
	authorizer, err := azureclient.GetAuthorizerFromContext(ctx, internalapi.ContextKeyClientAuthorizer)
	if err != nil {
		return nil, err
	}
	c.groupClient = resources.NewGroupsClient(ctx, log, cs.Properties.AzProfile.SubscriptionID, authorizer)
	if err != nil {
		return nil, err
	}

	return c, nil
}
