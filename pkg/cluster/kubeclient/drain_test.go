package kubeclient

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestDrainAndDeleteWorker(t *testing.T) {
	tests := []struct {
		name            string
		kubeclient      *fake.Clientset
		hostname        string
		wantErr         error
		expectedActions [][]string
	}{
		{
			name:     "compute-empty",
			hostname: "compute-000000",
			expectedActions: [][]string{
				{"get", "nodes"},
			},
			kubeclient: fake.NewSimpleClientset(),
		},
		{
			name:     "compute-no-pods",
			hostname: "compute-000000",
			expectedActions: [][]string{
				{"get", "nodes"},
				{"update", "nodes"},
				{"list", "pods"},
				{"delete", "nodes"},
			},
			kubeclient: fake.NewSimpleClientset(&corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "compute-000000",
				},
			}),
		},
		{
			name:     "compute-with-a-pod",
			hostname: "kubernetes",
			expectedActions: [][]string{
				{"get", "nodes"},
				{"update", "nodes"},
				{"list", "pods"},
				{"delete", "pods"},
				{"get", "pods"},
				{"delete", "nodes"},
			},
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
	for _, tt := range tests {
		u := &kubeclient{
			client: tt.kubeclient,
			log:    logrus.NewEntry(logrus.StandardLogger()),
		}
		if err := u.DrainAndDeleteWorker(context.Background(), tt.hostname); err != tt.wantErr {
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
