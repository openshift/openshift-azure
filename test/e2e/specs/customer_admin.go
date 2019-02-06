package specs

import (
	"errors"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	rbacv1 "k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/openshift/openshift-azure/pkg/util/randomstring"
	"github.com/openshift/openshift-azure/test/clients/openshift"
)

var _ = Describe("Openshift on Azure customer-admin e2e tests [CustomerAdmin][Fake]", func() {
	var (
		cli      *openshift.Client
		admincli *openshift.Client
	)

	BeforeEach(func() {
		var err error
		cli, err = openshift.NewEndUserClient()
		Expect(err).ToNot(HaveOccurred())
		admincli, err = openshift.NewCustomerAdminClient()
		Expect(err).ToNot(HaveOccurred())
	})

	It("should not read nodes", func() {
		_, err := admincli.CoreV1.Nodes().Get("master-000000", metav1.GetOptions{})
		Expect(kerrors.IsForbidden(err)).To(Equal(true))
	})

	It("should have full access on all non-infrastructure namespaces", func() {
		// Create project as normal user
		namespace, err := randomstring.RandomString("abcdefghijklmnopqrstuvwxyz0123456789", 5)
		Expect(err).ToNot(HaveOccurred())
		namespace = "e2e-test-" + namespace
		err = cli.CreateProject(namespace)
		Expect(err).ToNot(HaveOccurred())
		defer cli.CleanupProject(namespace)

		err = wait.PollImmediate(2*time.Second, 5*time.Minute, func() (bool, error) {
			rb, err := admincli.RbacV1.RoleBindings(namespace).Get("osa-customer-admin", metav1.GetOptions{})
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
		_, err = admincli.ProjectV1.Projects().Get(namespace, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		// attempt to delete namespace
		err = admincli.ProjectV1.Projects().Delete(namespace, &metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())
	})

	It("should not list infra namespace secrets", func() {
		// list all secrets in a namespace. should not see any in openshift-azure-logging
		_, err := admincli.CoreV1.Secrets("openshift-azure-logging").List(metav1.ListOptions{})
		Expect(kerrors.IsForbidden(err)).To(Equal(true))
	})

	It("should not list default namespace secrets", func() {
		// list all secrets in a namespace. should not see any in default
		_, err := admincli.CoreV1.Secrets("default").List(metav1.ListOptions{})
		Expect(kerrors.IsForbidden(err)).To(Equal(true))
	})

	It("should not able to query groups", func() {
		_, err := admincli.UserV1.Groups().Get("customer-admins", metav1.GetOptions{})
		Expect(kerrors.IsForbidden(err)).To(Equal(true))
	})

	It("should not be able to escalate privileges", func() {
		_, err := admincli.RbacV1.ClusterRoleBindings().Create(&rbacv1.ClusterRoleBinding{
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
		})
		Expect(kerrors.IsForbidden(err)).To(Equal(true))
	})

	// Placeholder to test that a ded admin cannot delete pods in the default or openshift- namespaces
})
