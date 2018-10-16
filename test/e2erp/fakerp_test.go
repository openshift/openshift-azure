//+build e2erp

package e2erp

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Resource provider e2e tests [Fake]", func() {
	defer GinkgoRecover()

	It("dummy fake test", func() {
		var err error
		Expect(err).NotTo(HaveOccurred())
	})
})
