package kubeclient

import (
	"context"
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetControlPlanePods(t *testing.T) {
	tests := []struct {
		name       string
		kc         kubernetes.Interface
		namespaces []string
		wantResult []corev1.Pod
		wantErr    bool
	}{
		{
			name:       "control plane namespace not found",
			namespaces: []string{"test"},
			wantErr:    false,
			kc:         fake.NewSimpleClientset(),
		},
		{
			name:       "control plane namespace found, no pods found",
			namespaces: []string{"kube-system"},
			wantErr:    false,
			kc: fake.NewSimpleClientset(&corev1.Namespace{
				TypeMeta: metav1.TypeMeta{
					Kind: "namespace",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "kube-system",
				},
				Status: corev1.NamespaceStatus{
					Phase: corev1.NamespaceActive,
				},
			}),
		},
		{
			name:       "control plane namespaces found, pods found",
			namespaces: []string{"kube-system", "test", "openshift-node"},
			wantErr:    false,
			wantResult: []corev1.Pod{
				{
					TypeMeta: metav1.TypeMeta{
						Kind: "pod",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "master-etcd-master-00000a",
						Namespace: "kube-system",
					},
				},
				{
					TypeMeta: metav1.TypeMeta{
						Kind: "pod",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "sync-test",
						Namespace: "openshift-node",
					},
				},
			},
			kc: fake.NewSimpleClientset(&corev1.Namespace{
				TypeMeta: metav1.TypeMeta{
					Kind: "namespace",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "kube-system",
				},
				Status: corev1.NamespaceStatus{
					Phase: corev1.NamespaceActive,
				},
			}, &corev1.Namespace{
				TypeMeta: metav1.TypeMeta{
					Kind: "namespace",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Status: corev1.NamespaceStatus{
					Phase: corev1.NamespaceActive,
				},
			}, &corev1.Namespace{
				TypeMeta: metav1.TypeMeta{
					Kind: "namespace",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "openshift-node",
				},
				Status: corev1.NamespaceStatus{
					Phase: corev1.NamespaceActive,
				},
			}, &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					Kind: "pod",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "master-etcd-master-00000a",
					Namespace: "kube-system",
				},
			}, &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					Kind: "pod",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test",
				},
			}, &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					Kind: "pod",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sync-test",
					Namespace: "openshift-node",
				},
			}),
		},
	}
	for _, tt := range tests {
		u := &Kubeclientset{Client: tt.kc}
		got, err := u.GetControlPlanePods(context.Background())
		if (err != nil) != tt.wantErr {
			t.Errorf("GetControlPlanePods() error = %v, wantErr %v. Test: %v", err, tt.wantErr, tt.name)
			return
		}
		if !reflect.DeepEqual(got, tt.wantResult) {
			t.Errorf("GetControlPlanePods() = %v, want %v. Test: %v", got, tt.wantResult, tt.name)
		}
	}
}
