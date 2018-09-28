package upgrade

import (
	"context"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/managedcluster"
)

func (u *simpleUpgrader) Deploy(ctx context.Context, cs *api.OpenShiftManagedCluster, azuredeploy []byte, deployFn api.DeployFn) error {
	err := deployFn(ctx, cs, azuredeploy)
	if err != nil {
		return err
	}

	err = u.InitializeCluster(ctx, cs)
	if err != nil {
		return err
	}
	kc, err := managedcluster.ClientSetFromV1Config(ctx, cs.Config.AdminKubeconfig)
	if err != nil {
		return err
	}

	// ensure that all nodes are ready
	err = WaitForNodes(ctx, cs, kc)
	if err != nil {
		return err
	}

	// Wait for infrastructure services to be healthy
	return WaitForInfraServices(ctx, kc)
}
