package upgrade

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	policy "k8s.io/api/policy/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/log"
)

type Upgrader interface {
	Drain(ctx context.Context, cs *api.OpenShiftManagedCluster, role api.AgentPoolProfileRole, nodeName string) error
	WaitForReady(ctx context.Context, cs *api.OpenShiftManagedCluster, role api.AgentPoolProfileRole, nodeName string) error
}

type simpleUpgrader struct{}

var _ Upgrader = &simpleUpgrader{}

func NewSimpleUpgrader(entry *logrus.Entry) Upgrader {
	log.New(entry)
	return &simpleUpgrader{}
}

func (u *simpleUpgrader) Drain(ctx context.Context, cs *api.OpenShiftManagedCluster, role api.AgentPoolProfileRole, nodeName string) error {
	kc, err := newClientset(cs)
	if err != nil {
		return err
	}

	_, err = kc.CoreV1().Nodes().Get(nodeName, metav1.GetOptions{})
	switch {
	case err == nil:
	case kerrors.IsNotFound(err):
		log.Info("drain: node not found, skipping")
		return nil
	default:
		return err
	}

	switch role {
	case api.AgentPoolProfileRoleMaster:
		// no-op for now

	case api.AgentPoolProfileRoleInfra, api.AgentPoolProfileRoleCompute:
		err := setUnschedulable(ctx, kc, nodeName, true)
		if err != nil {
			return err
		}

		err = evictPods(ctx, kc, nodeName)
		if err != nil {
			return err
		}

	default:
		return errors.New("unrecognised role")
	}

	return kc.CoreV1().Nodes().Delete(nodeName, &metav1.DeleteOptions{})
}

func setUnschedulable(ctx context.Context, kc *kubernetes.Clientset, nodeName string, unschedulable bool) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		node, err := kc.CoreV1().Nodes().Get(nodeName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		node.Spec.Unschedulable = unschedulable
		_, err = kc.CoreV1().Nodes().Update(node)
		return err
	})
}

func getControllerRef(pod *v1.Pod) *metav1.OwnerReference {
	for _, ref := range pod.OwnerReferences {
		if ref.Controller != nil && *ref.Controller {
			return &ref
		}
	}
	return nil
}

func max(i, j time.Duration) time.Duration {
	if i > j {
		return i
	}
	return j
}

func evictPods(ctx context.Context, kc *kubernetes.Clientset, nodeName string) error {
	podList, err := kc.CoreV1().Pods(metav1.NamespaceAll).List(metav1.ListOptions{
		FieldSelector: fields.SelectorFromSet(fields.Set{"spec.nodeName": nodeName}).String(),
	})
	if err != nil {
		return err
	}

	pods := map[*v1.Pod]struct{}{}
	duration := time.Duration(0)
	for i, pod := range podList.Items {
		if _, isMirror := pod.Annotations[v1.MirrorPodAnnotationKey]; isMirror {
			continue
		}

		controllerRef := getControllerRef(&pod)
		if controllerRef != nil && controllerRef.Kind == "DaemonSet" {
			continue
		}

		err = kc.CoreV1().Pods(pod.Namespace).Evict(&policy.Eviction{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pod.Name,
				Namespace: pod.Namespace,
			},
		})
		switch {
		case err == nil:
			d := 30 * time.Second
			if pod.Spec.TerminationGracePeriodSeconds != nil {
				d = 3 * time.Duration(*pod.Spec.TerminationGracePeriodSeconds+2) * time.Second
			}
			duration = max(duration, d)

		case apierrors.IsNotFound(err):
		default:
			// TODO: handle 429

			return err
		}

		pods[&podList.Items[i]] = struct{}{}
	}

	t := time.NewTimer(duration)
	defer t.Stop()
	return wait.PollUntil(time.Second, func() (bool, error) {
		for pod := range pods {
			p, err := kc.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
			switch {
			case apierrors.IsNotFound(err) || (p != nil && p.ObjectMeta.UID != pod.ObjectMeta.UID):
				delete(pods, pod)
			case err == nil:
			default:
				return false, err
			}
		}
		if len(pods) == 0 {
			return true, nil
		}
		select {
		case <-t.C:
			return true, nil
		default:
			return false, nil
		}
	}, ctx.Done())
}
