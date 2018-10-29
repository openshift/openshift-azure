package upgrade

import (
	"context"

	"github.com/openshift/openshift-azure/pkg/api"
)

func (u *simpleUpgrader) Deploy(ctx context.Context, cs *api.OpenShiftManagedCluster, azuretemplate map[string]interface{}, deployFn api.DeployFn) error {
	err := deployFn(ctx, azuretemplate)
	if err != nil {
		return err
	}

	err = u.InitializeCluster(ctx, cs)
	if err != nil {
		return err
	}

	// ensure that all nodes are ready
	return u.waitForNodes(ctx, cs)
}
