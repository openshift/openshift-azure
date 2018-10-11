//+build e2erp

package e2erp

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Resource provider e2e tests [Real]", func() {
	defer GinkgoRecover()

	It("dummy real test", func() {
		var err error
		Expect(err).NotTo(HaveOccurred())
	})
})
