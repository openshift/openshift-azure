package fakerp

import (
	"context"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openshift/openshift-azure/pkg/cluster/updateblob"
	"github.com/openshift/openshift-azure/test/clients/azure"
	"github.com/openshift/openshift-azure/test/sanity"
)

var _ = Describe("Force Update E2E tests [ForceUpdate][Fake][LongRunning]", func() {
	var (
		azurecli *azure.Client
	)

	BeforeEach(func() {
		var err error
		azurecli, err = azure.NewClientFromEnvironment(true)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should be possible for an SRE to force update a cluster", func() {
		By("Reading the update blob before the force update")
		ubs := updateblob.NewBlobService(azurecli.BlobStorage)
		before, err := ubs.Read()
		Expect(err).ToNot(HaveOccurred())
		Expect(before).NotTo(BeNil())
		Expect(len(before.HostnameHashes)).To(Equal(3)) // one per master instance
		Expect(len(before.ScalesetHashes)).To(Equal(2)) // one per worker scaleset

		By("Executing force update on the cluster.")
		err = azurecli.OpenShiftManagedClustersAdmin.ForceUpdate(context.Background(), os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"))
		Expect(err).NotTo(HaveOccurred())

		By("Reading the update blob after the force update")
		after, err := ubs.Read()
		Expect(err).ToNot(HaveOccurred())
		Expect(after).NotTo(BeNil())

		By("Verifying that the instance hashes of the update blob are identical (masters)")
		for key, val := range before.HostnameHashes {
			Expect(after.HostnameHashes).To(HaveKey(key))
			Expect(val).To(Equal(after.HostnameHashes[key]))
		}

		By("Verifying that the scaleset hashes of the update blob are different (workers)")
		for key := range before.ScalesetHashes {
			Expect(after.ScalesetHashes).NotTo(HaveKey(key))
		}

		By("Validating the cluster")
		errs := sanity.Checker.ValidateCluster(context.Background())
		Expect(errs).To(BeEmpty())
	})
})
