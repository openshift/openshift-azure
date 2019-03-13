package ready

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestIsPodReady(t *testing.T) {
	tests := []struct {
		name       string
		kubeclient *fake.Clientset
		want       bool
	}{
		{
			name: "ready",
			want: true,
			kubeclient: fake.NewSimpleClientset(&corev1.Pod{
				Status: corev1.PodStatus{
					Conditions: []corev1.PodCondition{
						{
							Type:   corev1.PodInitialized,
							Status: corev1.ConditionTrue,
						},
						{
							Type:   corev1.PodReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
			}),
		},
		{
			name: "missing",
			want: false,
			kubeclient: fake.NewSimpleClientset(&corev1.Pod{
				Status: corev1.PodStatus{
					Conditions: []corev1.PodCondition{
						{
							Type:   corev1.PodInitialized,
							Status: corev1.ConditionFalse,
						},
					},
				},
			}),
		},
	}
	for _, tt := range tests {
		if got, err := PodIsReady(tt.kubeclient.CoreV1().Pods(""), "")(); err != nil || got != tt.want {
			t.Errorf("isPodReady(%s) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestNodeIsReady(t *testing.T) {
	tests := []struct {
		name     string
		kc       *fake.Clientset
		hostname string
		want     bool
		wantErr  bool
	}{
		{
			name:     "not found",
			hostname: "master-000000",
			wantErr:  false,
			want:     false,
			kc:       fake.NewSimpleClientset(),
		},
		{
			name:     "not ready",
			hostname: "master-000000",
			wantErr:  false,
			want:     false,
			kc: fake.NewSimpleClientset(&corev1.Node{
				TypeMeta: metav1.TypeMeta{
					Kind: "node",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "master-000000",
				},
			}),
		},
		{
			name:     "ready",
			hostname: "master-00000a",
			wantErr:  false,
			want:     true,
			kc: fake.NewSimpleClientset(&corev1.Node{
				TypeMeta: metav1.TypeMeta{
					Kind: "node",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "master-00000a",
				},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{
							Type:   corev1.NodeReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
			}),
		},
	}
	for _, tt := range tests {
		got, err := NodeIsReady(tt.kc.CoreV1().Nodes(), tt.hostname)()
		if (err != nil) != tt.wantErr {
			t.Errorf("nodeIsReady() error = %v, wantErr %v", err, tt.wantErr)
			return
		}
		if got != tt.want {
			t.Errorf("%s: nodeIsReady() = %v, want %v", tt.name, got, tt.want)
		}
	}
}
