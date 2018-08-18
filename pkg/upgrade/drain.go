package upgrade

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	policy "k8s.io/api/policy/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/wait"
)

const cordonMaxRetries = 5

func (u *VMSSUpgrader) updateNode(nodeName string, cordon bool) error {
	node, err := u.kc.CoreV1().Nodes().Get(nodeName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	node.Spec.Unschedulable = cordon
	_, err = u.kc.CoreV1().Nodes().Update(node)
	return err
}

func (u *VMSSUpgrader) uncordon(nodeName string) error {
	var updated bool
	for i := 0; i < cordonMaxRetries; i++ {
		if err := u.updateNode(nodeName, false); err != nil {
			log.Info(err)
			time.Sleep(2 * time.Second)
			continue
		}
		updated = true
		break
	}
	if !updated {
		return fmt.Errorf("failed to uncordon node %q", nodeName)
	}
	log.Infof("Node %q has been marked schedulable.", nodeName)
	return nil
}

func (u *VMSSUpgrader) drain(nodeName string) error {
	var cordoned bool
	for i := 0; i < cordonMaxRetries; i++ {
		if err := u.updateNode(nodeName, true); err != nil {
			log.Info(err)
			time.Sleep(2 * time.Second)
			continue
		}
		cordoned = true
		break
	}
	if !cordoned {
		return fmt.Errorf("failed to cordon node %q", nodeName)
	}
	log.Infof("Node %q has been marked unschedulable.", nodeName)

	// Evict pods in the node.
	return u.evictPods(nodeName)
}

type podFilter func(v1.Pod) bool

func isMirrorPod(pod v1.Pod) bool {
	_, found := pod.Annotations[v1.MirrorPodAnnotationKey]
	return found
}

func getControllerRef(pod *v1.Pod) *metav1.OwnerReference {
	for _, ref := range pod.OwnerReferences {
		if ref.Controller != nil && *ref.Controller {
			return &ref
		}
	}
	return nil
}

func isDaemonSetPod(pod v1.Pod) bool {
	controllerRef := getControllerRef(&pod)
	// Don't delete/evict daemonsets as they will just come back
	// and deleting/evicting them can cause service disruptions.
	return controllerRef != nil && controllerRef.Kind == "DaemonSet"
}

func (u *VMSSUpgrader) evictPods(nodeName string) error {
	podList, err := u.kc.CoreV1().Pods(metav1.NamespaceAll).List(metav1.ListOptions{
		FieldSelector: fields.SelectorFromSet(fields.Set{"spec.nodeName": nodeName}).String()})
	if err != nil {
		return err
	}

	var pods []v1.Pod
	for _, pod := range podList.Items {
		podOk := true
		for _, filter := range []podFilter{isMirrorPod, isDaemonSetPod} {
			podOk = podOk && !filter(pod)
		}
		if podOk {
			pods = append(pods, pod)
		}
	}

	if len(pods) == 0 {
		return nil
	}

	doneCh := make(chan bool, len(pods))
	errCh := make(chan error, 1)

	for _, pod := range pods {
		go func(pod v1.Pod, doneCh chan bool, errCh chan error) {
			for {
				err = u.kc.CoreV1().Pods(pod.Namespace).Evict(&policy.Eviction{ObjectMeta: metav1.ObjectMeta{Name: pod.Name, Namespace: pod.Namespace}})
				if err == nil {
					break
				} else if apierrors.IsNotFound(err) {
					doneCh <- true
					return
				} else if apierrors.IsTooManyRequests(err) {
					time.Sleep(5 * time.Second)
				} else {
					errCh <- errors.Wrapf(err, "error when evicting pod %q", pod.Name)
					return
				}
			}
			// TODO: Make these configurable?
			timeout := 3 * time.Duration(*pod.Spec.TerminationGracePeriodSeconds+2) * time.Second
			if err := u.waitPodDeletion(&pod, 3*time.Second, timeout); err == nil {
				doneCh <- true
			} else {
				errCh <- errors.Wrapf(err, "error when waiting for pod %q terminating", pod.Name)
			}
		}(pod, doneCh, errCh)
	}

	// TODO: Make this configurable?
	timeout := 30 * time.Minute
	doneCount := 0
	for {
		select {
		case err := <-errCh:
			return err
		case <-doneCh:
			doneCount++
			if doneCount == len(pods) {
				return nil
			}
		case <-time.After(timeout):
			return errors.Errorf("Drain did not complete within %v", timeout)
		}
	}

	return nil
}

func (u *VMSSUpgrader) waitPodDeletion(pod *v1.Pod, interval, timeout time.Duration) error {
	return wait.PollImmediate(interval, timeout, func() (bool, error) {
		p, err := u.kc.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) || (p != nil && p.ObjectMeta.UID != pod.ObjectMeta.UID) {
			log.Infof("pod %q successfully evicted", pod.Name)
			return true, nil
		} else if err != nil {
			return false, err
		}
		// need to wait for that deletion..
		return false, nil
	})
}
