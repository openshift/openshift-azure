package fakerp

import (
	"context"
	"net/http"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Azure/go-autorest/autorest/to"

	"github.com/openshift/openshift-azure/test/clients/azure"
	"github.com/openshift/openshift-azure/test/e2e/standard"
)

var _ = Describe("Change OpenShift Component Log Level E2E tests [ChangeLogLevel][Fake][LongRunning]", func() {
	var (
		azurecli *azure.Client
		cli      *standard.SanityChecker
	)

	BeforeEach(func() {
		var err error
		azurecli, err = azure.NewClientFromEnvironment(true)
		Expect(err).NotTo(HaveOccurred())
		cli, err = standard.NewDefaultSanityChecker()
		Expect(err).NotTo(HaveOccurred())
		Expect(cli).NotTo(BeNil())
	})

	It("should be possible for an SRE to update the OpenShift component log level of a cluster", func() {
		By("Reading the internal config before the log level update")
		before, err := azurecli.OpenShiftManagedClustersAdmin.Get(context.Background(), os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"))
		Expect(err).NotTo(HaveOccurred())
		Expect(before).NotTo(BeNil())

		By("Executing a cluster update with updated log levels.")
		before.Config.ClusterLogLevel.ApiServer = to.IntPtr(4)
		before.Config.ClusterLogLevel.ControllerManager = to.IntPtr(4)
		update, err := azurecli.OpenShiftManagedClustersAdmin.CreateOrUpdateAndWait(context.Background(), os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"), before)
		Expect(err).NotTo(HaveOccurred())
		Expect(update).NotTo(BeNil())
		Expect(update.StatusCode).To(Equal(http.StatusOK))

		By("Reading the internal config after the log level update")
		after, err := azurecli.OpenShiftManagedClustersAdmin.Get(context.Background(), os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"))
		Expect(err).NotTo(HaveOccurred())
		Expect(after).NotTo(BeNil())

		By("Verifying that the cluster log level has been updated")
		Expect(after.Config.ClusterLogLevel).To(Equal(before.Config.ClusterLogLevel))

		By("Validating the cluster")
		errs := cli.ValidateCluster(context.Background())
		Expect(errs).To(BeEmpty())
	})
})
