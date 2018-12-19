package kubeclient

import (
	"context"
	"time"

	"github.com/pkg/errors"

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
}

func (u *kubeclient) WaitForInfraServices(ctx context.Context) *api.PluginError {
	for _, app := range daemonsetWhitelist {
		u.log.Infof("checking daemonset %s/%s", app.Namespace, app.Name)

		err := wait.PollImmediateUntil(time.Second, ready.DaemonSetIsReady(u.client.AppsV1().DaemonSets(app.Namespace), app.Name), ctx.Done())
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepWaitForInfraDaemonSets}
		}
	}

	for _, app := range statefulsetWhitelist {
		u.log.Infof("checking statefulset %s/%s", app.Namespace, app.Name)

		err := wait.PollImmediateUntil(time.Second, ready.StatefulSetIsReady(u.client.AppsV1().StatefulSets(app.Namespace), app.Name), ctx.Done())
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepWaitForInfraStatefulSets}
		}
	}

	for _, app := range deploymentWhitelist {
		u.log.Infof("checking deployment %s/%s", app.Namespace, app.Name)

		err := wait.PollImmediateUntil(time.Second, ready.DeploymentIsReady(u.client.AppsV1().Deployments(app.Namespace), app.Name), ctx.Done())
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepWaitForInfraDeployments}
		}
	}

	return nil
}

func (u *kubeclient) WaitForReady(ctx context.Context, role api.AgentPoolProfileRole, computerName ComputerName) error {
	switch role {
	case api.AgentPoolProfileRoleMaster:
		return u.masterWaitForReady(ctx, computerName)
	case api.AgentPoolProfileRoleInfra, api.AgentPoolProfileRoleCompute:
		return u.nodeWaitForReady(ctx, computerName)
	default:
		return errors.New("unrecognised role")
	}
}

func (u *kubeclient) masterWaitForReady(ctx context.Context, computerName ComputerName) error {
	return wait.PollImmediateUntil(time.Second, func() (bool, error) { return u.masterIsReady(computerName) }, ctx.Done())
}

func (u *kubeclient) masterIsReady(computerName ComputerName) (bool, error) {
	r, err := ready.NodeIsReady(u.client.CoreV1().Nodes(), computerName.toKubernetes())()
	if !r || err != nil {
		return r, err
	}

	r, err = ready.PodIsReady(u.client.CoreV1().Pods("kube-system"), "master-etcd-"+computerName.toKubernetes())()
	if !r || err != nil {
		return r, err
	}

	r, err = ready.PodIsReady(u.client.CoreV1().Pods("kube-system"), "master-api-"+computerName.toKubernetes())()
	if !r || err != nil {
		return r, err
	}

	return ready.PodIsReady(u.client.CoreV1().Pods("kube-system"), "controllers-"+computerName.toKubernetes())()
}

func (u *kubeclient) nodeWaitForReady(ctx context.Context, computerName ComputerName) error {
	err := wait.PollImmediateUntil(time.Second, ready.NodeIsReady(u.client.CoreV1().Nodes(), computerName.toKubernetes()), ctx.Done())
	if err != nil {
		return err
	}

	return u.setUnschedulable(ctx, computerName, false)
}
