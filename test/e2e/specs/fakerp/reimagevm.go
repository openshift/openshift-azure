package fakerp

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openshift/openshift-azure/pkg/config"
	"github.com/openshift/openshift-azure/pkg/util/resourceid"
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
		vmlist, err := azurecli.OpenShiftManagedClustersAdmin.ListClusterVMs(context.Background(), os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"))
		Expect(err).NotTo(HaveOccurred())
		Expect(len(vmlist.VMs)).To(BeNumerically(">=", 6))
		rand.Seed(time.Now().Unix())
		vm := vmlist.VMs[rand.Intn(len(vmlist.VMs))]
		By(fmt.Sprintf("Reimaging %s", vm))
		startTime := time.Now()
		update, err := azurecli.OpenShiftManagedClustersAdmin.ReimageAndWait(context.Background(), os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"), vm)
		endTime := time.Now()
		Expect(err).NotTo(HaveOccurred())
		Expect(update.StatusCode).To(Equal(http.StatusOK))
		Expect(update).NotTo(BeNil())

		By("Verifying through azure activity logs that the reimage happened")
		scaleset, instanceID, err := config.GetScaleSetNameAndInstanceID(vm)
		Expect(err).NotTo(HaveOccurred())

		logs, err := azurecli.ActivityLogs.List(
			context.Background(),
			fmt.Sprintf("eventTimestamp ge '%s' and eventTimestamp le '%s' and resourceUri eq %s",
				startTime.Format(time.RFC3339),
				endTime.Format(time.RFC3339),
				resourceid.ResourceID(os.Getenv("AZURE_SUBSCRIPTION_ID"), os.Getenv("RESOURCEGROUP"), "Microsoft.Compute/virtualMachineScaleSets", scaleset)+"/virtualMachines/"+instanceID,
			),
			"status,subscriptionId,resourceId,eventName,operationName,httpRequest")
		Expect(err).NotTo(HaveOccurred())

		var found bool
	out:
		for logs.NotDone() {
			for _, log := range logs.Values() {
				if *log.OperationName.Value == "Microsoft.Compute/virtualMachineScaleSets/virtualmachines/reimage/action" &&
					*log.Status.Value == "Succeeded" {
					found = true
					break out
				}
			}
			err = logs.Next()
			Expect(err).NotTo(HaveOccurred())
		}
		Expect(found).To(BeTrue())

		By("Validating the cluster")
		errs := cli.ValidateCluster(context.Background())
		Expect(errs).To(BeEmpty())
	})
})
