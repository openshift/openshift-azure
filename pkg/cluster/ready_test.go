package cluster

import (
	"context"
	"fmt"
	"strings"
	"testing"

	compute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_azureclient"
)

func mockListVMs(ctx context.Context, gmc *gomock.Controller, virtualMachineScaleSetVMsClient *mock_azureclient.MockVirtualMachineScaleSetVMsClient, cs *api.OpenShiftManagedCluster, role api.AgentPoolProfileRole, rg string, outVMS []compute.VirtualMachineScaleSetVM, outErr error) {
	mPage := mock_azureclient.NewMockVirtualMachineScaleSetVMListResultPage(gmc)
	if len(outVMS) > 0 {
		mPage.EXPECT().Values().Return(outVMS)
		mPage.EXPECT().Next()
	}
	callTimes := func(vms []compute.VirtualMachineScaleSetVM) int {
		if len(vms) > 0 {
			// NotDone gets called twice once for yes, there is data, and once more for no data
			return 2
		}
		// NotDone gets called once for there is no data
		return 1
	}
	if outErr == nil {
		mNotDone := len(outVMS) > 0
		mPage.EXPECT().NotDone().Times(callTimes(outVMS)).DoAndReturn(func() bool {
			ret := mNotDone
			mNotDone = false
			return ret
		})
	}
	scalesetName := strings.TrimSpace("ss-" + string(role))
	if outErr != nil {
		virtualMachineScaleSetVMsClient.EXPECT().List(ctx, rg, scalesetName, "", "", "").Return(nil, outErr)
	} else {
		virtualMachineScaleSetVMsClient.EXPECT().List(ctx, rg, scalesetName, "", "", "").Return(mPage, nil)
	}
}

func TestMasterIsReady(t *testing.T) {
	tests := []struct {
		name         string
		kc           kubernetes.Interface
		computerName computerName
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
		u := &simpleUpgrader{kubeclient: tt.kc}
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
				Properties: api.Properties{
					AzProfile: api.AzProfile{ResourceGroup: testRg},
				},
			},
			wantErr: false,
		},
		{
			name:       "list vm error",
			kubeclient: fake.NewSimpleClientset(),
			cs: &api.OpenShiftManagedCluster{
				Properties: api.Properties{
					AzProfile: api.AzProfile{ResourceGroup: testRg},
				},
			},
			wantErr:     true,
			expectedErr: vmListErr,
		},
		{
			name:       "node get error",
			kubeclient: fake.NewSimpleClientset(),
			cs: &api.OpenShiftManagedCluster{
				Properties: api.Properties{
					AzProfile: api.AzProfile{ResourceGroup: testRg},
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
			cs: &api.OpenShiftManagedCluster{
				Properties: api.Properties{
					AzProfile: api.AzProfile{ResourceGroup: testRg},
				},
			},
			expect: map[string][]compute.VirtualMachineScaleSetVM{
				"master": {
					{
						Name: to.StringPtr("ss-master"),
						VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
							OsProfile: &compute.OSProfile{
								ComputerName: to.StringPtr("master-00000A"),
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			gmc := gomock.NewController(t)
			defer gmc.Finish()
			virtualMachineScaleSetsClient := mock_azureclient.NewMockVirtualMachineScaleSetsClient(gmc)
			virtualMachineScaleSetVMsClient := mock_azureclient.NewMockVirtualMachineScaleSetVMsClient(gmc)

			if tt.wantErr {
				mockListVMs(ctx, gmc, virtualMachineScaleSetVMsClient, tt.cs, "master", testRg, nil, tt.expectedErr)
			} else {
				mockListVMs(ctx, gmc, virtualMachineScaleSetVMsClient, tt.cs, "master", testRg, tt.expect["master"], nil)
				mockListVMs(ctx, gmc, virtualMachineScaleSetVMsClient, tt.cs, "infra", testRg, tt.expect["infra"], nil)
				mockListVMs(ctx, gmc, virtualMachineScaleSetVMsClient, tt.cs, "compute", testRg, tt.expect["compute"], nil)
			}

			for _, react := range tt.reactors {
				tt.kubeclient.PrependReactor(react.verb, "nodes", react.reaction)
			}

			u := &simpleUpgrader{
				vmc:        virtualMachineScaleSetVMsClient,
				ssc:        virtualMachineScaleSetsClient,
				kubeclient: tt.kubeclient,
				log:        logrus.NewEntry(logrus.StandardLogger()),
			}
			err := u.waitForNodes(ctx, tt.cs)
			if tt.wantErr && tt.expectedErr != err {
				t.Errorf("simpleUpgrader.waitForNodes() wrong error got = %v, expected %v", err, tt.expectedErr)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("simpleUpgrader.waitForNodes() unexpected error = %v", err)
			}
		})
	}
}
