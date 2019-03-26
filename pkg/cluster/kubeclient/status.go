package kubeclient

import (
	"context"
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (u *kubeclient) GetControlPlanePods(ctx context.Context) ([]v1.Pod, error) {
	namespaces, err := u.client.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var pods []v1.Pod
	for _, namespace := range namespaces.Items {
		if IsControlPlaneNamespace(namespace.Name) {
			list, err := u.client.CoreV1().Pods(namespace.Name).List(metav1.ListOptions{IncludeUninitialized: true})
			if err != nil {
				return nil, err
			}
			pods = append(pods, list.Items...)
		}
	}
	return pods, nil
}

func IsControlPlaneNamespace(namespace string) bool {
	if namespace == "default" || namespace == "openshift" {
		return true
	}
	if strings.HasPrefix(namespace, "kube-") || strings.HasPrefix(namespace, "openshift-") {
		return true
	}
	return false
}
