package cluster

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/log"
)

func TestUpgraderDrain(t *testing.T) {
	tests := []struct {
		name            string
		kubeclient      *fake.Clientset
		role            api.AgentPoolProfileRole
		nodeName        string
		wantErr         error
		expectedActions [][]string
	}{
		{
			name:     "master-empty",
			role:     api.AgentPoolProfileRoleMaster,
			nodeName: "master-000000",
			expectedActions: [][]string{
				{"get", "nodes"}},
			kubeclient: fake.NewSimpleClientset(),
		},
		{
			name:     "unknown-role",
			role:     "cant-find-this",
			nodeName: "master-000000",
			expectedActions: [][]string{
				{"get", "nodes"}},
			wantErr: errUnrecognisedRole,
			kubeclient: fake.NewSimpleClientset(&corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "master-000000",
				},
			}),
		},
		{
			name:     "master-no-pods",
			role:     api.AgentPoolProfileRoleMaster,
			nodeName: "master-000000",
			expectedActions: [][]string{
				{"get", "nodes"},
				{"delete", "nodes"}},
			kubeclient: fake.NewSimpleClientset(&corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "master-000000",
				},
			}),
		},
		{
			name:     "compute-with-a-pod",
			role:     api.AgentPoolProfileRoleCompute,
			nodeName: "kubernetes",
			expectedActions: [][]string{
				{"get", "nodes"},
				{"get", "nodes"},
				{"update", "nodes"},
				{"list", "pods"},
				{"delete", "pods"},
				{"get", "pods"},
				{"delete", "nodes"}},
			kubeclient: fake.NewSimpleClientset(&corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "kubernetes",
				},
			}, &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pod",
				},
			}),
		},
	}
	log.New(logrus.NewEntry(logrus.New()))
	for _, tt := range tests {
		u := &simpleUpgrader{
			kubeclient: tt.kubeclient,
		}
		if err := u.drain(context.Background(), nil, tt.role, tt.nodeName); err != tt.wantErr {
			t.Errorf("[%v] simpleUpgrader.drain() error = %v, wantErr %v", tt.name, err, tt.wantErr)
		}
		actions := tt.kubeclient.Actions()
		if len(actions) != len(tt.expectedActions) {
			t.Errorf("[%v] Expected %d actions, got %d : %v", tt.name, len(tt.expectedActions), len(actions), actions)
		}
		for i, action := range tt.expectedActions {
			if !actions[i].Matches(action[0], action[1]) {
				t.Errorf("[%v] unexpected action: %v, expected %v", tt.name, actions[i], action)
			}
		}
	}
}
