package fakerp

import (
	"context"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Azure/go-autorest/autorest/to"

	"github.com/openshift/openshift-azure/pkg/cluster/updateblob"
	"github.com/openshift/openshift-azure/test/clients/azure"
	"github.com/openshift/openshift-azure/test/sanity"
)

// This test also fulfills the need to test upgrade reentrancy (Jira-221):
// Create an OSA cluster
// Fetch the .config.configStorageAccount update blob locally
// Do an update in the master startup script (pkg/arm/data/master-startup.sh
// Update the OSA cluster
// Refetch the blob and ensure it the master VM hashes between the local copy from the 2nd step and the new blob are different
var _ = Describe("Change OpenShift Component Log Level E2E tests [ChangeLogLevel][LongRunning]", func() {
	var (
		ctx = context.Background()
	)
	It("should be possible for an SRE to update the OpenShift component log level of a cluster", func() {
		By("Reading the internal config before the log level update")
		before, err := azure.RPClient.OpenShiftManagedClustersAdmin.Get(ctx, os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"))
		Expect(err).NotTo(HaveOccurred())
		Expect(before).NotTo(BeNil())

		By("Reading the update blob before the update")
		ubs := updateblob.NewBlobService(azure.RPClient.BlobStorage)
		beforeBlob, err := ubs.Read()
		Expect(err).ToNot(HaveOccurred())
		Expect(beforeBlob).NotTo(BeNil())
		Expect(len(beforeBlob.HostnameHashes)).To(Equal(3)) // one per master instance
		Expect(len(beforeBlob.ScalesetHashes)).To(Equal(2)) // one per worker scaleset

		By("Executing a cluster update with updated log levels.")
		before.Config.ComponentLogLevel.APIServer = to.IntPtr(*before.Config.ComponentLogLevel.APIServer - 2)
		before.Config.ComponentLogLevel.ControllerManager = to.IntPtr(*before.Config.ComponentLogLevel.ControllerManager - 2)
		before.Config.ComponentLogLevel.Node = to.IntPtr(*before.Config.ComponentLogLevel.Node - 2)
		update, err := azure.RPClient.OpenShiftManagedClustersAdmin.CreateOrUpdate(ctx, os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"), before)
		Expect(err).NotTo(HaveOccurred())
		Expect(update).NotTo(BeNil())

		By("Reading the internal config after the log level update")
		after, err := azure.RPClient.OpenShiftManagedClustersAdmin.Get(ctx, os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"))
		Expect(err).NotTo(HaveOccurred())
		Expect(after).NotTo(BeNil())

		By("Verifying that the cluster log level has been updated")
		Expect(after.Config.ComponentLogLevel).To(Equal(before.Config.ComponentLogLevel))

		By("Reading the update blob after the update")
		afterBlob, err := ubs.Read()
		Expect(err).ToNot(HaveOccurred())
		Expect(afterBlob).NotTo(BeNil())

		By("Verifying that the instance hashes of the update blob are different (masters)")
		for key := range beforeBlob.HostnameHashes {
			Expect(afterBlob.HostnameHashes).ToNot(HaveKey(key))
		}

		By("Verifying that the scaleset hashes of the update blob are identical (workers)")
		for key, val := range beforeBlob.ScalesetHashes {
			Expect(afterBlob.ScalesetHashes).To(HaveKey(key))
			Expect(val).To(Equal(afterBlob.ScalesetHashes[key]))
		}

		By("Validating the cluster")
		errs := sanity.Checker.ValidateCluster(context.Background())
		Expect(errs).To(BeEmpty())
	})
})
