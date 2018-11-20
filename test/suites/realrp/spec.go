//+build e2e

package realrp

import (
	. "github.com/onsi/ginkgo"

	"github.com/openshift/openshift-azure/test/util/client/azure"
	"github.com/openshift/openshift-azure/test/util/scenarios/realrp"
)

var (
	az        *azure.Client
	gitCommit = "unknown"
)

var _ = Describe("Azure resource provider E2E tests [Real]", func() {
	defer GinkgoRecover()

	It("should not be possible for customer to mutate an osa scale set", func() {
		realrp.TestCustomerCannotModifyScaleSet(az)
	})
})
