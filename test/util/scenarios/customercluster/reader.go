//+build e2e

package customercluster

import (
	"errors"
	"time"

	. "github.com/onsi/gomega"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/openshift/openshift-azure/test/util/client/kubernetes"
)

func CheckCannotQueryCustomerReaderGroup(kc *kubernetes.Client) {
	_, err := kc.GetGroup("customer-readers", nil)
	Expect(kerrors.IsForbidden(err)).To(Equal(true))
}

func CheckReadOnlyAccessToNonInfraNamespaces(kc *kubernetes.Client) {
	// create project as enduser
	namespace := kc.GenerateRandomName("e2e-test-")
	kc.CreateProject(namespace)
	defer kc.CleanupProject(10 * time.Minute)

	err := wait.PollImmediate(2*time.Second, 5*time.Minute, func() (bool, error) {
		rb, err := kc.GetNamespace(namespace, nil)
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
	ns, err := kc.GetProject(namespace, nil)
	Expect(err).ToNot(HaveOccurred())
	Expect(ns).NotTo(Equal(nil))
	// attempt to delete namespace
	err = kc.DeleteProject(namespace, nil)
	Expect(kerrors.IsForbidden(err)).To(Equal(true))
}
