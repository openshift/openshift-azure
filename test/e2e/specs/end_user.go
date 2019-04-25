package specs

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openshift/openshift-azure/test/sanity"
)

var _ = Describe("Openshift on Azure end user e2e tests [EndUser][Fake][EveryPR]", func() {
	It("should create and validate test apps [EndUser][Fake][Apps]", func() {
		ctx := context.Background()
		By("creating test app")
		namespace, errs := sanity.Checker.CreateTestApp(ctx)
		Expect(errs).To(BeEmpty())
		defer func() {
			By("deleting test app")
			_ = sanity.Checker.DeleteTestApp(ctx, namespace)
		}()

		By("validating test app")
		errs = sanity.Checker.ValidateTestApp(ctx, namespace)
		Expect(errs).To(BeEmpty())
	})

	It("should validate the cluster [EndUser][Fake][Cluster]", func() {
		errs := sanity.Checker.ValidateCluster(context.Background())
		Expect(errs).To(BeEmpty())
	})
})
