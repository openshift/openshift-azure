package specs

import (
	"errors"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/openshift/openshift-azure/pkg/util/randomstring"
	"github.com/openshift/openshift-azure/test/clients/openshift"
)

var _ = Describe("Openshift on Azure customer-reader e2e tests [CustomerAdmin][Fake]", func() {
	var (
		cli       *openshift.Client
		readercli *openshift.Client
	)

	BeforeEach(func() {
		var err error
		cli, err = openshift.NewEndUserClient()
		Expect(err).ToNot(HaveOccurred())
		readercli, err = openshift.NewCustomerReaderClient()
		Expect(err).ToNot(HaveOccurred())
	})

	It("should not read nodes", func() {
		_, err := readercli.CoreV1.Nodes().Get("master-000000", metav1.GetOptions{})
		Expect(kerrors.IsForbidden(err)).To(Equal(true))
	})

	It("should have read-only on all non-infrastructure namespaces", func() {
		// create project as enduser
		namespace, err := randomstring.RandomString("abcdefghijklmnopqrstuvwxyz0123456789", 5)
		Expect(err).ToNot(HaveOccurred())
		namespace = "e2e-test-" + namespace
		err = cli.CreateProject(namespace)
		Expect(err).ToNot(HaveOccurred())
		defer cli.CleanupProject(namespace)

		err = wait.PollImmediate(2*time.Second, 5*time.Minute, func() (bool, error) {
			rb, err := readercli.CoreV1.Namespaces().Get(namespace, metav1.GetOptions{})
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
			if rb != nil && rb.Name == namespace {
				return true, nil
			}
			return false, errors.New("namespace retrieved is incorrect")
		})
		Expect(err).ToNot(HaveOccurred())
		// get project created by user
		ns, err := readercli.ProjectV1.Projects().Get(namespace, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(ns).NotTo(Equal(nil))
		// attempt to delete namespace
		err = readercli.ProjectV1.Projects().Delete(namespace, &metav1.DeleteOptions{})
		Expect(kerrors.IsForbidden(err)).To(Equal(true))
	})

	It("should not list infra namespace secrets", func() {
		// list all namespaces. should not see default
		_, err := readercli.CoreV1.Secrets("default").List(metav1.ListOptions{})
		Expect(kerrors.IsForbidden(err)).To(Equal(true))
	})

	It("should not be able to query groups", func() {
		_, err := readercli.UserV1.Groups().Get("customer-readers", metav1.GetOptions{})
		Expect(kerrors.IsForbidden(err)).To(Equal(true))
	})
})
