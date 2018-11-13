package cluster

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/ready"
	"github.com/openshift/openshift-azure/pkg/util/wait"
)

var deploymentWhitelist = []struct {
	Name      string
	Namespace string
}{
	{
		Name:      "docker-registry",
		Namespace: "default",
	},
	{
		Name:      "router",
		Namespace: "default",
	},
	{
		Name:      "registry-console",
		Namespace: "default",
	},
	{
		Name:      "customer-admin-controller",
		Namespace: "openshift-infra",
	},
	{
		Name:      "asb",
		Namespace: "openshift-ansible-service-broker",
	},
	{
		Name:      "webconsole",
		Namespace: "openshift-web-console",
	},
	{
		Name:      "console",
		Namespace: "openshift-console",
	},
	{
		Name:      "cluster-monitoring-operator",
		Namespace: "openshift-monitoring",
	},
	{
		Name:      "mdm",
		Namespace: "openshift-azure-monitoring",
	},
	{
		Name:      "prom-mdm-converter",
		Namespace: "openshift-azure-monitoring",
	},
}

var daemonsetWhitelist = []struct {
	Name      string
	Namespace string
}{
	{
		Name:      "sync",
		Namespace: "openshift-node",
	},
	{
		Name:      "ovs",
		Namespace: "openshift-sdn",
	},
	{
		Name:      "sdn",
		Namespace: "openshift-sdn",
	},
	{
		Name:      "apiserver",
		Namespace: "kube-service-catalog",
	},
	{
		Name:      "controller-manager",
		Namespace: "kube-service-catalog",
	},
	{
		Name:      "apiserver",
		Namespace: "openshift-template-service-broker",
	},
	{
		Name:      "mdsd",
		Namespace: "openshift-azure-logging",
	},
	{
		Name:      "td-agent",
		Namespace: "openshift-azure-logging",
	},
}

var statefulsetWhitelist = []struct {
	Name      string
	Namespace string
}{
	{
		Name:      "bootstrap-autoapprover",
		Namespace: "openshift-infra",
	},
	{
		Name:      "prometheus-forwarder",
		Namespace: "openshift-azure-monitoring",
	},
}

type computerName string

func (computerName computerName) toKubernetes() string {
	return strings.ToLower(string(computerName))
}

func (u *simpleUpgrader) waitForNodes(ctx context.Context, cs *api.OpenShiftManagedCluster) error {
	for _, role := range []api.AgentPoolProfileRole{api.AgentPoolProfileRoleMaster, api.AgentPoolProfileRoleInfra, api.AgentPoolProfileRoleCompute} {
		vms, err := u.listVMs(ctx, cs, role)
		if err != nil {
			return err
		}
		for _, vm := range vms {
			computerName := computerName(*vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
			u.log.Infof("waiting for %s to be ready", computerName)
			err = u.waitForReady(ctx, cs, role, computerName)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (u *simpleUpgrader) WaitForInfraServices(ctx context.Context, cs *api.OpenShiftManagedCluster) *api.PluginError {
	for _, app := range daemonsetWhitelist {
		u.log.Infof("checking daemonset %s/%s", app.Namespace, app.Name)

		err := wait.PollImmediateUntil(time.Second, ready.DaemonSetIsReady(u.kubeclient.AppsV1().DaemonSets(app.Namespace), app.Name), ctx.Done())
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepWaitForInfraDaemonSets}
		}
	}

	for _, app := range statefulsetWhitelist {
		u.log.Infof("checking statefulset %s/%s", app.Namespace, app.Name)

		err := wait.PollImmediateUntil(time.Second, ready.StatefulSetIsReady(u.kubeclient.AppsV1().StatefulSets(app.Namespace), app.Name), ctx.Done())
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepWaitForInfraStatefulSets}
		}
	}

	for _, app := range deploymentWhitelist {
		u.log.Infof("checking deployment %s/%s", app.Namespace, app.Name)

		err := wait.PollImmediateUntil(time.Second, ready.DeploymentIsReady(u.kubeclient.AppsV1().Deployments(app.Namespace), app.Name), ctx.Done())
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepWaitForInfraDeployments}
		}
	}

	return nil
}

func (u *simpleUpgrader) waitForReady(ctx context.Context, cs *api.OpenShiftManagedCluster, role api.AgentPoolProfileRole, computerName computerName) error {
	switch role {
	case api.AgentPoolProfileRoleMaster:
		return u.masterWaitForReady(ctx, cs, computerName)
	case api.AgentPoolProfileRoleInfra, api.AgentPoolProfileRoleCompute:
		return u.nodeWaitForReady(ctx, cs, computerName)
	default:
		return errors.New("unrecognised role")
	}
}

func (u *simpleUpgrader) masterWaitForReady(ctx context.Context, cs *api.OpenShiftManagedCluster, computerName computerName) error {
	return wait.PollImmediateUntil(time.Second, func() (bool, error) { return u.masterIsReady(computerName) }, ctx.Done())
}

func (u *simpleUpgrader) masterIsReady(computerName computerName) (bool, error) {
	r, err := ready.NodeIsReady(u.kubeclient.CoreV1().Nodes(), computerName.toKubernetes())()
	if !r || err != nil {
		return r, err
	}

	r, err = ready.PodIsReady(u.kubeclient.CoreV1().Pods("kube-system"), "master-etcd-"+computerName.toKubernetes())()
	if !r || err != nil {
		return r, err
	}

	r, err = ready.PodIsReady(u.kubeclient.CoreV1().Pods("kube-system"), "master-api-"+computerName.toKubernetes())()
	if !r || err != nil {
		return r, err
	}

	r, err = ready.PodIsReady(u.kubeclient.CoreV1().Pods("kube-system"), "controllers-"+computerName.toKubernetes())()
	if !r || err != nil {
		return r, err
	}

	return true, nil
}

func (u *simpleUpgrader) nodeWaitForReady(ctx context.Context, cs *api.OpenShiftManagedCluster, computerName computerName) error {
	err := wait.PollImmediateUntil(time.Second, ready.NodeIsReady(u.kubeclient.CoreV1().Nodes(), computerName.toKubernetes()), ctx.Done())
	if err != nil {
		return err
	}

	return u.setUnschedulable(ctx, computerName, false)
}
