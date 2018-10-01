//+build e2e

package e2e

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Openshift on Azure end user e2e tests [EndUser]", func() {
	defer GinkgoRecover()

	BeforeEach(func() {
		// TODO: Use a generator here
		namespace := "generateme"
		// TODO: The namespace is cached in the client so this will not
		// work with parallel tests.
		c.createNamespace(namespace)
	})

	AfterEach(func() {
		if CurrentGinkgoTestDescription().Failed {
			// TODO: Dump info from namespace
		}
		c.cleanupNamespace(10 * time.Minute)
	})

	// TODO: Add tests
	It("dummy test", func() {
		var err error
		Expect(err).NotTo(HaveOccurred())
	})
})
