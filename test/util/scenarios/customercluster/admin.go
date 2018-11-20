//+build e2e

package customercluster

import (
	"errors"
	"time"

	. "github.com/onsi/gomega"
	rbacv1 "k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/openshift/openshift-azure/test/util/client/kubernetes"
)

func CheckCannotGetNode(kc *kubernetes.Client) {
	_, err := kc.GetNode("master-000000", nil)
	Expect(kerrors.IsForbidden(err)).To(Equal(true))
}

func CheckCannotListInfraSecrets(kc *kubernetes.Client) {
	// list all namespaces. should not see default
	_, err := kc.ListSecrets("default", nil)
	Expect(kerrors.IsForbidden(err)).To(Equal(true))
}

func CheckCannotQueryCustomerAdminGroup(kc *kubernetes.Client) {
	_, err := kc.GetGroup("customer-admins", nil)
	Expect(kerrors.IsForbidden(err)).To(Equal(true))
}

func CheckCannotEscalatePrivilegeToClusterAdmin(kc *kubernetes.Client) {
	roleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-cluster-admin",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "User",
				Name: "customer-cluster-admin",
			},
		},
		RoleRef: rbacv1.RoleRef{
			Name: "cluster-admin",
			Kind: "ClusterRole",
		},
	}
	_, err := kc.CreateClusterRoleBinding(roleBinding)
	Expect(kerrors.IsForbidden(err)).To(Equal(true))
}

func CheckFullAccessToNonInfraNamespaces(kc *kubernetes.Client) {
	// Create project as normal user
	namespace := kc.GenerateRandomName("e2e-test-")
	kc.CreateProject(namespace)
	defer kc.CleanupProject(10 * time.Minute)

	err := wait.PollImmediate(2*time.Second, 5*time.Minute, func() (bool, error) {
		rb, err := kc.GetRoleBinding(namespace, "osa-customer-admin", nil)
		if err != nil {
			// still waiting for namespace
			if kerrors.IsNotFound(err) {
				return false, nil
			}
			// still waiting for reconciler and permissions
			if kerrors.IsForbidden(err) {
				return false, nil
			}
			return false, err
		}
		for _, subject := range rb.Subjects {
			if subject.Kind == "Group" && subject.Name == "osa-customer-admins" {
				return true, nil
			}
		}
		return false, errors.New("customer-admins rolebinding does not bind to customer-admins group")
	})
	Expect(err).ToNot(HaveOccurred())
	// get namespace created by user
	_, err = kc.GetProject(namespace, nil)
	Expect(err).ToNot(HaveOccurred())
	// attempt to delete namespace
	err = kc.DeleteProject(namespace, nil)
	Expect(err).NotTo(HaveOccurred())
}
