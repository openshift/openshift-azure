package upgrade

import (
	"context"

	"github.com/openshift/openshift-azure/pkg/api"
)

func (u *simpleUpgrader) Deploy(ctx context.Context, cs *api.OpenShiftManagedCluster, azuredeploy []byte, deployFn api.DeployFn) error {
	err := deployFn(ctx, cs, azuredeploy)
	if err != nil {
		return err
	}

	return u.InitializeCluster(ctx, cs)
}
