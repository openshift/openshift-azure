package kubeclient

import (
	"context"
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (u *Kubeclientset) GetControlPlanePods(ctx context.Context) ([]v1.Pod, error) {
	namespaces, err := u.Client.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var pods []v1.Pod
	for _, namespace := range namespaces.Items {
		if IsControlPlaneNamespace(namespace.Name) {
			list, err := u.Client.CoreV1().Pods(namespace.Name).List(metav1.ListOptions{IncludeUninitialized: true})
			if err != nil {
				return nil, err
			}
			pods = append(pods, list.Items...)
		}
	}
	return pods, nil
}

func (u *Kubeclientset) GetLiveClusterInfo(ctx context.Context) ([]v1.Pod, error) {
	namespaces, err := u.Client.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var pods []v1.Pod
	for _, namespace := range namespaces.Items {
		if IsControlPlaneNamespace(namespace.Name) {
			list, err := u.Client.CoreV1().Pods(namespace.Name).List(metav1.ListOptions{IncludeUninitialized: true, LabelSelector: "app=sync"})
			if err != nil {
				return nil, err
			}
			pods = append(pods, list.Items...)
		}
	}
	for _, namespace := range namespaces.Items {
		if IsControlPlaneNamespace(namespace.Name) {
			list, err := u.Client.CoreV1().Pods(namespace.Name).List(metav1.ListOptions{IncludeUninitialized: true, LabelSelector: "app=openshift-web-console"})
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
