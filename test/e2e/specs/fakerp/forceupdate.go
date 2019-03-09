package fakerp

import (
	"context"
	"net/http"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openshift/openshift-azure/pkg/cluster/updateblob"
	"github.com/openshift/openshift-azure/test/clients/azure"
	"github.com/openshift/openshift-azure/test/e2e/standard"
)

var _ = Describe("Force Update E2E tests [ForceUpdate][Fake][LongRunning]", func() {
	var (
		azurecli *azure.Client
		cli      *standard.SanityChecker
	)

	BeforeEach(func() {
		var err error
		azurecli, err = azure.NewClientFromEnvironment(true)
		Expect(err).NotTo(HaveOccurred())
		cli, err = standard.NewDefaultSanityChecker(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(cli).NotTo(BeNil())
	})

	It("should be possible for an SRE to force update a cluster", func() {
		By("Reading the update blob before the force update")
		ubs, err := updateblob.NewBlobService(azurecli.BlobStorage)
		Expect(err).ToNot(HaveOccurred())
		before, err := ubs.Read()
		Expect(err).ToNot(HaveOccurred())
		Expect(before).NotTo(BeNil())
		Expect(len(before.InstanceHashes)).To(BeEquivalentTo(3)) // one per master instance
		Expect(len(before.ScalesetHashes)).To(BeEquivalentTo(2)) // one per worker scaleset

		By("Executing force update on the cluster.")
		update, err := azurecli.OpenShiftManagedClustersAdmin.ForceUpdateAndWait(context.Background(), os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"))
		Expect(err).NotTo(HaveOccurred())
		Expect(update.StatusCode).To(Equal(http.StatusOK))
		Expect(update).NotTo(BeNil())

		By("Reading the update blob after the force update")
		after, err := ubs.Read()
		Expect(err).ToNot(HaveOccurred())
		Expect(after).NotTo(BeNil())

		By("Verifying that the instance hashes of the update blob are identical (masters)")
		for key, val := range before.InstanceHashes {
			Expect(after.InstanceHashes).To(HaveKey(key))
			Expect(val).To(Equal(after.InstanceHashes[key]))
		}

		By("Verifying that the scaleset hashes of the update blob are different (workers)")
		for key := range before.ScalesetHashes {
			Expect(after.ScalesetHashes).NotTo(HaveKey(key))
		}

		By("Validating the cluster")
		errs := cli.ValidateCluster(context.Background())
		Expect(errs).To(BeEmpty())
	})
})
