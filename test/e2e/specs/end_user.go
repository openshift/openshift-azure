package specs

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openshift/openshift-azure/test/e2e/standard"
)

var _ = Describe("Openshift on Azure end user e2e tests [EndUser][Fake]", func() {
	var (
		cli *standard.SanityChecker
	)

	BeforeEach(func() {
		var err error
		cli, err = standard.NewDefaultSanityChecker()
		Expect(err).NotTo(HaveOccurred())
		Expect(cli).ToNot(BeNil())
	})

	It("should create and validate test apps", func() {
		ctx := context.Background()
		By("creating test app")
		namespace, errs := cli.CreateTestApp(ctx)
		Expect(len(errs)).To(Equal(0))
		defer func() {
			By("deleting test app")
			_ = cli.DeleteTestApp(ctx, namespace)
		}()

		By("validating test app")
		errs = cli.ValidateTestApp(ctx, namespace)
		Expect(len(errs)).To(Equal(0))
	})

	It("should validate the cluster", func() {
		errs := cli.ValidateCluster(context.Background())
		Expect(len(errs)).To(Equal(0))
	})
})
