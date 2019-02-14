package fakerp

import (
	"context"
	"math/rand"
	"net/http"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openshift/openshift-azure/pkg/config"
	"github.com/openshift/openshift-azure/test/clients/azure"
	"github.com/openshift/openshift-azure/test/e2e/standard"
)

var _ = Describe("Reimage VM E2E tests [ReimageVM][Fake][LongRunning]", func() {
	var (
		azurecli *azure.Client
		cli      *standard.SanityChecker
	)

	BeforeEach(func() {
		var err error
		azurecli, err = azure.NewClientFromEnvironment(false)
		Expect(err).NotTo(HaveOccurred())
		Expect(azurecli).NotTo(BeNil())
		cli, err = standard.NewDefaultSanityChecker()
		Expect(err).NotTo(HaveOccurred())
		Expect(cli).NotTo(BeNil())
	})

	It("should be possible for an SRE to reimage a VM in a scale set", func() {
		By("Reading the cluster state")
		before, err := azurecli.OpenShiftManagedClustersAdmin.Get(context.Background(), os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"))
		Expect(err).NotTo(HaveOccurred())
		Expect(before).NotTo(BeNil())

		By("Executing reimage on a vm in the cluster")
		vms, err := azurecli.VirtualMachineScaleSetVMs.List(context.Background(), os.Getenv("RESOURCEGROUP"), config.MasterScalesetName, "", "", "")
		Expect(err).NotTo(HaveOccurred())
		rand.Seed(time.Now().Unix())
		vm := vms[rand.Intn(len(vms))]
		startTime := time.Now()
		update, err := azurecli.OpenShiftManagedClustersAdmin.ReimageAndWait(context.Background(), os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"), *vm.OsProfile.ComputerName)
		endTime := time.Now()
		Expect(err).NotTo(HaveOccurred())
		Expect(update.StatusCode).To(Equal(http.StatusOK))
		Expect(update).NotTo(BeNil())

		By("Verifying through azure activity logs that the reimage happened")
		logs, err := azurecli.ActivityLogsByResourceIdAndStatus(context.Background(), *vm.ID, startTime, endTime, azure.ActivitySucceeded)
		Expect(err).NotTo(HaveOccurred())
		Expect(logs).NotTo(BeEmpty())
		loggedOperations := make(map[string]bool)
		for _, log := range logs {
			loggedOperations[*log.OperationName.Value] = true
		}
		Expect(loggedOperations).To(HaveKey("Microsoft.Compute/virtualMachineScaleSets/virtualmachines/reimage/action"))

		By("Validating the cluster")
		errs := cli.ValidateCluster(context.Background())
		Expect(errs).To(BeEmpty())
	})
})
