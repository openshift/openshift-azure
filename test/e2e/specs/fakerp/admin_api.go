package fakerp

import (
	"context"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Azure/go-autorest/autorest/to"

	"github.com/openshift/openshift-azure/test/clients/azure"
)

var _ = Describe("Validate AdimAPI field readability [AdminAPI][Fake]", func() {
	var (
		cli *azure.Client
	)

	BeforeEach(func() {
		var err error
		cli, err = azure.NewClientFromEnvironment(false)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should not be possible to update clusterVersion using AdminAPI", func() {
		By("Reading the cluster state")
		before, err := cli.OpenShiftManagedClustersAdmin.Get(context.Background(), os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"))
		Expect(err).NotTo(HaveOccurred())
		Expect(before).NotTo(BeNil())
		Expect(before.Config.ClusterVersion).To(BeEquivalentTo(to.StringPtr("v0.0")))

		By("Updating the cluster state")
		before.Config.ClusterVersion = to.StringPtr("v0.1")
		update, err := cli.OpenShiftManagedClustersAdmin.CreateOrUpdate(context.Background(), os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"), before)
		Expect(err).NotTo(HaveOccurred())
		Expect(update).NotTo(BeNil())

		By("Reading the cluster state")
		after, err := cli.OpenShiftManagedClustersAdmin.Get(context.Background(), os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"))
		Expect(err).NotTo(HaveOccurred())
		Expect(after).NotTo(BeNil())
		Expect(after.Config.ClusterVersion).To(BeEquivalentTo(to.StringPtr("v0.0")))
	})
})
