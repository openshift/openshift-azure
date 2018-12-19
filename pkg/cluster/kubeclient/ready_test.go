package kubeclient

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"

	"github.com/openshift/openshift-azure/pkg/api"
)

func TestMasterIsReady(t *testing.T) {
	tests := []struct {
		name         string
		kc           kubernetes.Interface
		computerName ComputerName
		want         bool
		wantErr      bool
	}{
		{
			name:         "node not found",
			computerName: "master-000000",
			wantErr:      false,
			want:         false,
			kc:           fake.NewSimpleClientset(),
		},
		{
			name:         "node ready, pods not found",
			computerName: "master-000000",
			wantErr:      false,
			want:         false,
			kc: fake.NewSimpleClientset(&corev1.Node{
				TypeMeta: metav1.TypeMeta{
					Kind: "node",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "master-000000",
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
		{
			name:         "node ready, pods ready",
			computerName: "master-00000A",
			wantErr:      false,
			want:         true,
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
			}, &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					Kind: "pod",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "master-etcd-master-00000a",
					Namespace: "kube-system",
				},
				Status: corev1.PodStatus{
					Conditions: []corev1.PodCondition{
						{
							Type:   corev1.PodReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
			}, &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					Kind: "pod",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "master-api-master-00000a",
					Namespace: "kube-system",
				},
				Status: corev1.PodStatus{
					Conditions: []corev1.PodCondition{
						{
							Type:   corev1.PodReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
			}, &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					Kind: "pod",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "controllers-master-00000a",
					Namespace: "kube-system",
				},
				Status: corev1.PodStatus{
					Conditions: []corev1.PodCondition{
						{
							Type:   corev1.PodReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
			}),
		},
	}
	for _, tt := range tests {
		u := &kubeclient{client: tt.kc}
		got, err := u.masterIsReady(tt.computerName)
		if (err != nil) != tt.wantErr {
			t.Errorf("masterIsReady() error = %v, wantErr %v. Test: %v", err, tt.wantErr, tt.name)
			return
		}
		if got != tt.want {
			t.Errorf("masterIsReady() = %v, want %v. Test: %v", got, tt.want, tt.name)
		}
	}
}

func TestWaitForReady(t *testing.T) {
	nodeGetErr := fmt.Errorf("node get failed")
	tests := []struct {
		name         string
		client       *fake.Clientset
		role         api.AgentPoolProfileRole
		computerName ComputerName
		wantErr      bool
		expectedErr  error
		reactors     []struct {
			verb     string
			reaction clienttesting.ReactionFunc
		}
	}{
		{
			name:         "node get error",
			client:       fake.NewSimpleClientset(),
			computerName: "master-000000",
			role:         api.AgentPoolProfileRoleMaster,
			wantErr:      true,
			expectedErr:  nodeGetErr,
			reactors: []struct {
				verb     string
				reaction clienttesting.ReactionFunc
			}{
				{
					verb: "get",
					reaction: func(action clienttesting.Action) (handled bool, ret runtime.Object, err error) {
						return true, nil, nodeGetErr
					},
				},
			},
		},
		{
			name:         "all ready",
			computerName: "compute-000000",
			role:         api.AgentPoolProfileRoleCompute,
			client: fake.NewSimpleClientset(&corev1.Node{
				TypeMeta: metav1.TypeMeta{
					Kind: "node",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "compute-000000",
				},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{
							Type:   corev1.NodeReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
			}, &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					Kind: "pod",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "master-etcd-master-00000a",
					Namespace: "kube-system",
				},
				Status: corev1.PodStatus{
					Conditions: []corev1.PodCondition{
						{
							Type:   corev1.PodReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
			}, &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					Kind: "pod",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "master-api-master-00000a",
					Namespace: "kube-system",
				},
				Status: corev1.PodStatus{
					Conditions: []corev1.PodCondition{
						{
							Type:   corev1.PodReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
			}, &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					Kind: "pod",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "controllers-master-00000a",
					Namespace: "kube-system",
				},
				Status: corev1.PodStatus{
					Conditions: []corev1.PodCondition{
						{
							Type:   corev1.PodReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
			}),
			wantErr: false,
		},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			gmc := gomock.NewController(t)
			defer gmc.Finish()

			for _, react := range tt.reactors {
				tt.client.PrependReactor(react.verb, "nodes", react.reaction)
			}
			u := &kubeclient{
				client: tt.client,
				log:    logrus.NewEntry(logrus.StandardLogger()),
			}

			if err := u.WaitForReady(ctx, tt.role, tt.computerName); (err != nil) != tt.wantErr {
				t.Errorf("kubeclient.WaitForReady() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
