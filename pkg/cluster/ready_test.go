package cluster

import (
	"context"
	"fmt"
	"testing"

	compute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	gomock "github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/log"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_azureclient"
)

func TestIsPodReady(t *testing.T) {
	tests := []struct {
		name string
		pod  *corev1.Pod
		want bool
	}{
		{
			name: "ready",
			want: true,
			pod: &corev1.Pod{
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
			},
		},
		{
			name: "missing",
			want: false,
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					Conditions: []corev1.PodCondition{
						{
							Type:   corev1.PodInitialized,
							Status: corev1.ConditionFalse,
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		if got := isPodReady(tt.pod); got != tt.want {
			t.Errorf("isPodReady(%s) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestNodeIsReady(t *testing.T) {
	tests := []struct {
		name     string
		kc       *fake.Clientset
		nodeName string
		want     bool
		wantErr  bool
	}{
		{
			name:     "not found",
			nodeName: "master-000000",
			wantErr:  false,
			want:     false,
			kc:       fake.NewSimpleClientset(),
		},
		{
			name:     "not ready",
			nodeName: "master-000000",
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
			nodeName: "master-000000",
			wantErr:  false,
			want:     true,
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
	}
	for _, tt := range tests {
		got, err := nodeIsReady(tt.kc, tt.nodeName)
		if (err != nil) != tt.wantErr {
			t.Errorf("nodeIsReady() error = %v, wantErr %v", err, tt.wantErr)
			return
		}
		if got != tt.want {
			t.Errorf("nodeIsReady() = %v, want %v", got, tt.want)
		}
	}
}

func TestMasterIsReady(t *testing.T) {
	tests := []struct {
		name     string
		kc       kubernetes.Interface
		nodeName string
		want     bool
		wantErr  bool
	}{
		{
			name:     "node not found",
			nodeName: "master-000000",
			wantErr:  false,
			want:     false,
			kc:       fake.NewSimpleClientset(),
		},
		{
			name:     "node ready, pods not found",
			nodeName: "master-000000",
			wantErr:  false,
			want:     false,
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
			name:     "node ready, pods ready",
			nodeName: "master-000000",
			wantErr:  false,
			want:     true,
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
			}, &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					Kind: "pod",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "etcd-master-000000",
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
					Name:      "api-master-000000",
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
					Name:      "controllers-master-000000",
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
		got, err := masterIsReady(tt.kc, tt.nodeName)
		if (err != nil) != tt.wantErr {
			t.Errorf("masterIsReady() error = %v, wantErr %v", err, tt.wantErr)
			return
		}
		if got != tt.want {
			t.Errorf("masterIsReady() = %v, want %v", got, tt.want)
		}
	}
}

func TestUpgraderWaitForNodes(t *testing.T) {
	vmListErr := fmt.Errorf("vm list failed")
	nodeGetErr := fmt.Errorf("node get failed")
	testRg := "myrg"
	tests := []struct {
		name        string
		kubeclient  *fake.Clientset
		cs          *api.OpenShiftManagedCluster
		expect      map[string][]compute.VirtualMachineScaleSetVM
		wantErr     bool
		expectedErr error
		reactors    []struct {
			verb     string
			reaction clienttesting.ReactionFunc
		}
	}{
		{
			name:       "nothing to wait for",
			kubeclient: fake.NewSimpleClientset(),
			cs: &api.OpenShiftManagedCluster{
				Properties: &api.Properties{
					AzProfile: &api.AzProfile{ResourceGroup: testRg},
				},
			},
			wantErr: false,
		},
		{
			name:       "list vm error",
			kubeclient: fake.NewSimpleClientset(),
			cs: &api.OpenShiftManagedCluster{
				Properties: &api.Properties{
					AzProfile: &api.AzProfile{ResourceGroup: testRg},
				},
			},
			wantErr:     true,
			expectedErr: vmListErr,
		},
		{
			name:       "node get error",
			kubeclient: fake.NewSimpleClientset(),
			cs: &api.OpenShiftManagedCluster{
				Properties: &api.Properties{
					AzProfile: &api.AzProfile{ResourceGroup: testRg},
				},
			},
			wantErr:     true,
			expectedErr: nodeGetErr,
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
			expect: map[string][]compute.VirtualMachineScaleSetVM{
				"master": {
					{
						Name: to.StringPtr("ss-master"),
						VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
							OsProfile: &compute.OSProfile{
								ComputerName: to.StringPtr("master-000000"),
							},
						},
					},
				},
				"infra": {
					{
						Name: to.StringPtr("ss-infra"),
						VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
							OsProfile: &compute.OSProfile{
								ComputerName: to.StringPtr("infra-000000"),
							},
						},
					},
				},
				"compute": {
					{
						Name: to.StringPtr("ss-compute"),
						VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
							OsProfile: &compute.OSProfile{
								ComputerName: to.StringPtr("compute-000000"),
							},
						},
					},
				},
			},
		},
		{
			name: "all ready",
			kubeclient: fake.NewSimpleClientset(&corev1.Node{
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
			}, &corev1.Node{
				TypeMeta: metav1.TypeMeta{
					Kind: "node",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "infra-000000",
				},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{
							Type:   corev1.NodeReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
			}, &corev1.Node{
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
			}, &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					Kind: "pod",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "etcd-master-000000",
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
					Name:      "api-master-000000",
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
					Name:      "controllers-master-000000",
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
			cs: &api.OpenShiftManagedCluster{
				Properties: &api.Properties{
					AzProfile: &api.AzProfile{ResourceGroup: testRg},
				},
			},
			expect: map[string][]compute.VirtualMachineScaleSetVM{
				"master": {
					{
						Name: to.StringPtr("ss-master"),
						VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
							OsProfile: &compute.OSProfile{
								ComputerName: to.StringPtr("master-000000"),
							},
						},
					},
				},
				"infra": {
					{
						Name: to.StringPtr("ss-infra"),
						VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
							OsProfile: &compute.OSProfile{
								ComputerName: to.StringPtr("infra-000000"),
							},
						},
					},
				},
				"compute": {
					{
						Name: to.StringPtr("ss-compute"),
						VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
							OsProfile: &compute.OSProfile{
								ComputerName: to.StringPtr("compute-000000"),
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}

	log.New(logrus.NewEntry(logrus.New()))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			gmc := gomock.NewController(t)
			virtualMachineScaleSetsClient := mock_azureclient.NewMockVirtualMachineScaleSetsClient(gmc)
			virtualMachineScaleSetVMsClient := mock_azureclient.NewMockVirtualMachineScaleSetVMsClient(gmc)

			mPage := mock_azureclient.NewMockVirtualMachineScaleSetVMListResultPage(gmc)
			iPage := mock_azureclient.NewMockVirtualMachineScaleSetVMListResultPage(gmc)
			cPage := mock_azureclient.NewMockVirtualMachineScaleSetVMListResultPage(gmc)

			for _, react := range tt.reactors {
				tt.kubeclient.PrependReactor(react.verb, "nodes", react.reaction)
			}

			if len(tt.expect) > 0 {
				mPage.EXPECT().Values().Return(tt.expect["master"])
				mPage.EXPECT().Next()
				iPage.EXPECT().Values().Return(tt.expect["infra"])
				iPage.EXPECT().Next()
				cPage.EXPECT().Values().Return(tt.expect["compute"])
				cPage.EXPECT().Next()
			}
			callTimes := func(vms []compute.VirtualMachineScaleSetVM) int {
				if len(vms) > 0 {
					// NotDone gets called twice once for yes, there is data, and once more for no data
					return 2
				}
				// NotDone gets called once for there is no data
				return 1
			}
			mNotDone := len(tt.expect["master"]) > 0
			mPage.EXPECT().NotDone().Times(callTimes(tt.expect["master"])).DoAndReturn(func() bool {
				ret := mNotDone
				mNotDone = false
				return ret
			})
			iNotDone := len(tt.expect["infra"]) > 0
			iPage.EXPECT().NotDone().Times(callTimes(tt.expect["infra"])).DoAndReturn(func() bool {
				ret := iNotDone
				iNotDone = false
				return ret
			})
			cNotDone := len(tt.expect["compute"]) > 0
			cPage.EXPECT().NotDone().Times(callTimes(tt.expect["compute"])).DoAndReturn(func() bool {
				ret := cNotDone
				cNotDone = false
				return ret
			})

			if tt.wantErr && len(tt.reactors) == 0 {
				virtualMachineScaleSetVMsClient.EXPECT().List(ctx, testRg, "ss-master", "", "", "").Return(nil, tt.expectedErr)
			} else {
				virtualMachineScaleSetVMsClient.EXPECT().List(ctx, testRg, "ss-master", "", "", "").Return(mPage, nil)
				virtualMachineScaleSetVMsClient.EXPECT().List(ctx, testRg, "ss-infra", "", "", "").Return(iPage, nil)
				virtualMachineScaleSetVMsClient.EXPECT().List(ctx, testRg, "ss-compute", "", "", "").Return(cPage, nil)
			}
			u := &simpleUpgrader{
				vmc:        virtualMachineScaleSetVMsClient,
				ssc:        virtualMachineScaleSetsClient,
				kubeclient: tt.kubeclient,
			}
			err := u.waitForNodes(context.Background(), tt.cs)
			if tt.wantErr && tt.expectedErr != err {
				t.Errorf("simpleUpgrader.waitForNodes() wrong error got = %v, expected %v", err, tt.expectedErr)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("simpleUpgrader.waitForNodes() unexpected error = %v", err)
			}
		})
	}
}
