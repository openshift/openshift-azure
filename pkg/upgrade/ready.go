package upgrade

import (
	"context"
	"errors"
	"time"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/log"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/managedcluster"
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
		Name:      "apiserver",
		Namespace: "kube-service-catalog",
	},
	{
		Name:      "controller-manager",
		Namespace: "kube-service-catalog",
	},
	{
		Name:      "bootstrap-autoapprover",
		Namespace: "openshift-infra",
	},
	{
		Name:      "asb",
		Namespace: "openshift-ansible-service-broker",
	},
	{
		Name:      "apiserver",
		Namespace: "openshift-template-service-broker",
	},
	{
		Name:      "bootstrap-autoapprover",
		Namespace: "openshift-infra",
	},
	{
		Name:      "webconsole",
		Namespace: "openshift-web-console",
	},
}

var daemonsetWhitelist = []struct {
	Name      string
	Namespace string
}{
	{
		Name:      "prometheus-node-exporter",
		Namespace: "openshift-metrics",
	},
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
}

func WaitForNodes(ctx context.Context, cs *api.OpenShiftManagedCluster, kc *kubernetes.Clientset) error {
	config := api.PluginConfig{AcceptLanguages: []string{"en-us"}}
	authorizer, err := azureclient.NewAuthorizerFromContext(ctx)
	if err != nil {
		return err
	}
	vmc := azureclient.NewVirtualMachineScaleSetVMsClient(cs.Properties.AzProfile.SubscriptionID, authorizer, config.AcceptLanguages)

	for _, role := range []api.AgentPoolProfileRole{api.AgentPoolProfileRoleMaster, api.AgentPoolProfileRoleInfra, api.AgentPoolProfileRoleCompute} {
		vms, err := ListVMs(ctx, cs, vmc, role)
		if err != nil {
			return err
		}
		for _, vm := range vms {
			log.Infof("waiting for %s to be ready", *vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
			err = WaitForReady(ctx, cs, role, *vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// WaitForInfraServices verifies daemonsets, statefulsets
func WaitForInfraServices(ctx context.Context, kc *kubernetes.Clientset) error {
	for _, app := range daemonsetWhitelist {
		log.Infof("checking daemonset %s/%s", app.Namespace, app.Name)

		err := wait.PollImmediateUntil(time.Second, func() (bool, error) {
			ds, err := kc.AppsV1().DaemonSets(app.Namespace).Get(app.Name, metav1.GetOptions{})
			switch {
			case kerrors.IsNotFound(err):
				return false, nil
			case err == nil:
				return ds.Status.DesiredNumberScheduled == ds.Status.CurrentNumberScheduled &&
					ds.Status.DesiredNumberScheduled == ds.Status.NumberReady &&
					ds.Status.DesiredNumberScheduled == ds.Status.UpdatedNumberScheduled &&
					ds.Generation == ds.Status.ObservedGeneration, nil
			default:
				return false, err
			}
		}, ctx.Done())
		if err != nil {
			return err
		}
	}

	for _, app := range deploymentWhitelist {
		log.Infof("checking deployment %s/%s", app.Namespace, app.Name)

		err := wait.PollImmediateUntil(time.Second, func() (bool, error) {
			d, err := kc.AppsV1().Deployments(app.Namespace).Get(app.Name, metav1.GetOptions{})
			switch {
			case kerrors.IsNotFound(err):
				return false, nil
			case err == nil:
				specReplicas := int32(1)
				if d.Spec.Replicas != nil {
					specReplicas = *d.Spec.Replicas
				}

				return specReplicas == d.Status.Replicas &&
					specReplicas == d.Status.ReadyReplicas &&
					specReplicas == d.Status.AvailableReplicas &&
					specReplicas == d.Status.UpdatedReplicas &&
					d.Generation == d.Status.ObservedGeneration, nil
			default:
				return false, err
			}
		}, ctx.Done())
		if err != nil {
			return err
		}
	}

	return nil
}

func WaitForReady(ctx context.Context, cs *api.OpenShiftManagedCluster, role api.AgentPoolProfileRole, nodeName string) error {
	switch role {
	case api.AgentPoolProfileRoleMaster:
		return masterWaitForReady(ctx, cs, nodeName)
	case api.AgentPoolProfileRoleInfra, api.AgentPoolProfileRoleCompute:
		return nodeWaitForReady(ctx, cs, nodeName)
	default:
		return errors.New("unrecognised role")
	}
}

func masterWaitForReady(ctx context.Context, cs *api.OpenShiftManagedCluster, nodeName string) error {
	kc, err := managedcluster.ClientsetFromV1Config(cs.Config.AdminKubeconfig)
	if err != nil {
		return err
	}

	return wait.PollImmediateUntil(time.Second, func() (bool, error) {
		return masterIsReady(kc, nodeName)
	}, ctx.Done())
}

func masterIsReady(kc *kubernetes.Clientset, nodeName string) (bool, error) {
	ready, err := nodeIsReady(kc, nodeName)
	if !ready || err != nil {
		return ready, err
	}

	etcdPod, err := kc.CoreV1().Pods("kube-system").Get("etcd-"+nodeName, metav1.GetOptions{})
	switch {
	case err == nil:
	case kerrors.IsNotFound(err):
		return false, nil
	default:
		return false, err
	}

	apiPod, err := kc.CoreV1().Pods("kube-system").Get("api-"+nodeName, metav1.GetOptions{})
	switch {
	case err == nil:
	case kerrors.IsNotFound(err):
		return false, nil
	default:
		return false, err
	}

	cmPod, err := kc.CoreV1().Pods("kube-system").Get("controllers-"+nodeName, metav1.GetOptions{})
	switch {
	case err == nil:
	case kerrors.IsNotFound(err):
		return false, nil
	default:
		return false, err
	}

	return isPodReady(etcdPod) && isPodReady(apiPod) && isPodReady(cmPod), nil
}

func nodeWaitForReady(ctx context.Context, cs *api.OpenShiftManagedCluster, nodeName string) error {
	kc, err := managedcluster.ClientsetFromV1Config(cs.Config.AdminKubeconfig)
	if err != nil {
		return err
	}

	err = wait.PollImmediateUntil(time.Second, func() (bool, error) {
		return nodeIsReady(kc, nodeName)
	}, ctx.Done())
	if err != nil {
		return err
	}

	return setUnschedulable(ctx, kc, nodeName, false)
}

func nodeIsReady(kc *kubernetes.Clientset, nodeName string) (bool, error) {
	node, err := kc.CoreV1().Nodes().Get(nodeName, metav1.GetOptions{})
	switch {
	case err == nil:
	case kerrors.IsNotFound(err):
		return false, nil
	default:
		return false, err
	}

	return isNodeReady(node), nil
}

func isPodReady(pod *corev1.Pod) bool {
	for _, c := range pod.Status.Conditions {
		if c.Type == corev1.PodReady {
			return c.Status == corev1.ConditionTrue
		}
	}
	return false
}

func isNodeReady(node *corev1.Node) bool {
	for _, c := range node.Status.Conditions {
		if c.Type == corev1.NodeReady {
			return c.Status == corev1.ConditionTrue
		}
	}
	return false
}
