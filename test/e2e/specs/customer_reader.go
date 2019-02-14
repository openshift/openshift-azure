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
	"github.com/openshift/openshift-azure/test/e2e/standard"
)

var _ = Describe("Openshift on Azure customer-reader e2e tests [CustomerAdmin][Fake]", func() {
	var (
		cli *standard.SanityChecker
	)

	BeforeEach(func() {
		var err error
		cli, err = standard.NewDefaultSanityChecker()
		Expect(err).NotTo(HaveOccurred())
		Expect(cli).ToNot(BeNil())
	})

	It("should not read nodes", func() {
		_, err := cli.Client.CustomerReader.CoreV1.Nodes().Get("master-000000", metav1.GetOptions{})
		Expect(kerrors.IsForbidden(err)).To(Equal(true))
	})

	It("should have read-only on all non-infrastructure namespaces", func() {
		// create project as enduser
		namespace, err := randomstring.RandomString("abcdefghijklmnopqrstuvwxyz0123456789", 5)
		Expect(err).ToNot(HaveOccurred())
		namespace = "e2e-test-" + namespace
		err = cli.Client.EndUser.CreateProject(namespace)
		Expect(err).ToNot(HaveOccurred())
		defer cli.Client.EndUser.CleanupProject(namespace)

		err = wait.PollImmediate(2*time.Second, 5*time.Minute, func() (bool, error) {
			rb, err := cli.Client.CustomerReader.CoreV1.Namespaces().Get(namespace, metav1.GetOptions{})
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
		ns, err := cli.Client.CustomerReader.ProjectV1.Projects().Get(namespace, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(ns).NotTo(Equal(nil))
	})

	It("should not list infra namespace secrets", func() {
		// list all secrets in a namespace. should not see any in openshift-azure-logging
		_, err := cli.Client.CustomerReader.CoreV1.Secrets("openshift-azure-logging").List(metav1.ListOptions{})
		Expect(kerrors.IsForbidden(err)).To(Equal(true))
	})

	It("should not list default namespace secrets", func() {
		// list all secrets in a namespace. should not see any in default
		_, err := cli.Client.CustomerReader.CoreV1.Secrets("default").List(metav1.ListOptions{})
		Expect(kerrors.IsForbidden(err)).To(Equal(true))
	})

	It("should not be able to query groups", func() {
		_, err := cli.Client.CustomerReader.UserV1.Groups().Get("customer-readers", metav1.GetOptions{})
		Expect(kerrors.IsForbidden(err)).To(Equal(true))
	})
})
