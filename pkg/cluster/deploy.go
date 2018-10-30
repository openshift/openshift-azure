package cluster

import (
	"context"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/arm"
	"github.com/openshift/openshift-azure/pkg/log"
	"github.com/openshift/openshift-azure/pkg/util/managedcluster"
)

func (u *simpleUpgrader) Deploy(ctx context.Context, cs *api.OpenShiftManagedCluster, azuretemplate map[string]interface{}, deployFn api.DeployFn) error {
	err := u.createClients(ctx, cs)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepClientCreation}
	}
	err = deployFn(ctx, azuretemplate)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepDeploy}
	}
	err = u.Initialize(ctx, cs)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepInitialize}
	}
	err = managedcluster.WaitForHealthz(ctx, cs.Config.AdminKubeconfig)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepWaitForWaitForOpenShiftAPI}
	}
	err = u.waitForNodes(ctx, cs)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepWaitForNodes}
	}
	err = u.copyVmHashesInCluster(ctx, cs, azuretemplate)
	if err != nil {
		log.Warnf("could not copy vm hashes into the cluster: %v", err)
	}
	return nil
}

func (u *simpleUpgrader) copyVmHashesInCluster(ctx context.Context, cs *api.OpenShiftManagedCluster, azuretemplate map[string]interface{}) error {
	updateTracker := make(map[string]string)
	for _, profile := range cs.Properties.AgentPoolProfiles {
		vms, err := listVMs(ctx, cs, u.vmc, profile.Role)
		if err != nil {
			return err
		}
		if hash := arm.GetTag(string(profile.Role), arm.HashKey, azuretemplate); hash != "" {
			for _, vm := range vms {
				updateTracker[*vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName] = hash
			}
		} else {
			// TODO: Warn
		}
	}

	return u.updateConfigMap(updateTracker)
}

func (u *simpleUpgrader) updateConfigMap(data map[string]string) error {
	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: "upgrade-tracker",
		},
		Data: data,
	}

	// TODO: Retry on conflicts
	_, err := u.kubeclient.CoreV1().ConfigMaps("openshift-azure").Update(cm)
	return err
}

func (u *simpleUpgrader) readConfigMap() (*v1.ConfigMap, error) {
	return u.kubeclient.CoreV1().ConfigMaps("openshift-azure").Get("upgrade-tracker", metav1.GetOptions{})
}
