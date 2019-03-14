package kubeclient

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (u *kubeclient) DeletePod(ctx context.Context, namespace, name string) error {
	return u.client.CoreV1().Pods(namespace).Delete(name, &metav1.DeleteOptions{})
}
