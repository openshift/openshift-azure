package upgrade

import (
	"context"
	"errors"
	"time"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	acsapi "github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/managedcluster"
	"github.com/openshift/openshift-azure/pkg/util/wait"
)

func WaitForReady(ctx context.Context, cs *acsapi.OpenShiftManagedCluster, role acsapi.AgentPoolProfileRole, nodeName string) error {
	switch role {
	case acsapi.AgentPoolProfileRoleMaster:
		return masterWaitForReady(ctx, cs, nodeName)
	case acsapi.AgentPoolProfileRoleInfra, acsapi.AgentPoolProfileRoleCompute:
		return nodeWaitForReady(ctx, cs, nodeName)
	default:
		return errors.New("unrecognised role")
	}
}

func masterWaitForReady(ctx context.Context, cs *acsapi.OpenShiftManagedCluster, nodeName string) error {
	kc, err := managedcluster.ClientsetFromConfig(cs)
	if err != nil {
		return err
	}

	return wait.PollImmediateUntil(time.Second, func() (bool, error) {
		return masterIsReady(kc, nodeName)
	}, ctx.Done())
}

func masterIsReady(kc *kubernetes.Clientset, nodeName string) (bool, error) {
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

func nodeWaitForReady(ctx context.Context, cs *acsapi.OpenShiftManagedCluster, nodeName string) error {
	kc, err := managedcluster.ClientsetFromConfig(cs)
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
