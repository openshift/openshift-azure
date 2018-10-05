package upgrade

import (
	"context"
	"encoding/json"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/managedcluster"
)

func (u *simpleUpgrader) Deploy(ctx context.Context, cs *api.OpenShiftManagedCluster, azuredeploy []byte, deployFn api.DeployFn) error {
	var azuretemplate map[string]interface{}
	err := json.Unmarshal(azuredeploy, &azuretemplate)
	if err != nil {
		return err
	}
	err = deployFn(ctx, azuretemplate)
	if err != nil {
		return err
	}

	err = u.InitializeCluster(ctx, cs)
	if err != nil {
		return err
	}
	kc, err := managedcluster.ClientsetFromV1ConfigAndWait(ctx, cs.Config.AdminKubeconfig)
	if err != nil {
		return err
	}

	// ensure that all nodes are ready
	return WaitForNodes(ctx, cs, kc)
}
