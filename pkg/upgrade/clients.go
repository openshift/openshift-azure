package upgrade

import (
	"context"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

func (u *simpleUpgrader) getClients(ctx context.Context, cs *api.OpenShiftManagedCluster) (*azureclient.AzureClients, error) {
	if u.clients == nil {
		var err error
		u.clients, err = azureclient.NewAzureClients(ctx, cs, u.pluginConfig)
		if err != nil {
			return nil, err
		}
	}

	return u.clients, nil
}
