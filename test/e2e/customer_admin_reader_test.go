//+build e2e

package e2e

import (
	"errors"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

var _ = Describe("Openshift on Azure customer-cluster-reader e2e tests [CustomerAdmin]", func() {
	defer GinkgoRecover()

	It("should not read nodes", func() {
		_, err := creader.kc.CoreV1().Nodes().Get("master-000000", metav1.GetOptions{})
		Expect(kerrors.IsForbidden(err)).To(Equal(true))
	})

	It("should have read-only on all non-infrastructure namespaces", func() {
		// create project as enduser
		namespace := nameGen.generate("e2e-test-")
		c.createProject(namespace)
		defer c.cleanupProject(10 * time.Minute)

		err := wait.PollImmediate(2*time.Second, 5*time.Minute, func() (bool, error) {
			rb, err := creader.kc.CoreV1().Namespaces().Get(c.namespace, metav1.GetOptions{})
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
			if rb != nil && rb.Name == c.namespace {
				return true, nil
			}
			return false, errors.New("namespace retrieved is incorrect")
		})
		Expect(err).ToNot(HaveOccurred())
		// get project created by user
		ns, err := creader.pc.Projects().Get(c.namespace, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(ns).NotTo(Equal(nil))
		// attempt to delete namespace
		err = creader.pc.Projects().Delete(c.namespace, &metav1.DeleteOptions{})
		Expect(kerrors.IsForbidden(err)).To(Equal(true))
	})

	It("should not list infra namespace secrets", func() {
		// list all namespaces. should not see default
		_, err := creader.kc.CoreV1().Secrets("default").List(metav1.ListOptions{})
		Expect(kerrors.IsForbidden(err)).To(Equal(true))
	})

	It("should not be able to query groups", func() {
		_, err := creader.uc.Groups().Get("customer-readers", metav1.GetOptions{})
		Expect(kerrors.IsForbidden(err)).To(Equal(true))
	})
})
