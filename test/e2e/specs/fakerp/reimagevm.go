package fakerp

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/openshift/openshift-azure/pkg/cluster/names"
	"github.com/openshift/openshift-azure/pkg/util/resourceid"
	"github.com/openshift/openshift-azure/test/clients/azure"
	"github.com/openshift/openshift-azure/test/sanity"
)

var _ = Describe("Reimage VM E2E tests [ReimageVM][Fake][LongRunning]", func() {
	var (
		azurecli *azure.Client
	)

	BeforeEach(func() {
		var err error
		azurecli, err = azure.NewClientFromEnvironment(false)
		Expect(err).NotTo(HaveOccurred())
		Expect(azurecli).NotTo(BeNil())
	})

	It("should be possible for an SRE to reimage a VM in a scale set", func() {
		By("Reading the cluster state")
		before, err := azurecli.OpenShiftManagedClustersAdmin.Get(context.Background(), os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"))
		Expect(err).NotTo(HaveOccurred())
		Expect(before).NotTo(BeNil())

		By("Executing reimage on a vm in the cluster")
		vmlist, err := azurecli.OpenShiftManagedClustersAdmin.ListClusterVMs(context.Background(), os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"))
		Expect(err).NotTo(HaveOccurred())
		Expect(len(*vmlist.VMs)).To(BeNumerically(">=", 6))
		rand.Seed(time.Now().Unix())
		vm := (*vmlist.VMs)[rand.Intn(len(*vmlist.VMs))]
		By(fmt.Sprintf("Reimaging %s", vm))
		startTime := time.Now()
		err = azurecli.OpenShiftManagedClustersAdmin.Reimage(context.Background(), os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"), vm)
		Expect(err).NotTo(HaveOccurred())

		scaleset, instanceID, err := names.GetScaleSetNameAndInstanceID(vm)
		Expect(err).NotTo(HaveOccurred())

		wait.PollImmediate(10*time.Second, 2*time.Minute, func() (bool, error) {
			By("Verifying through azure activity logs that the reimage happened")
			logs, err := azurecli.ActivityLogs.List(
				context.Background(),
				fmt.Sprintf("eventTimestamp ge '%s' and resourceUri eq %s",
					startTime.Format(time.RFC3339),
					resourceid.ResourceID(os.Getenv("AZURE_SUBSCRIPTION_ID"), os.Getenv("RESOURCEGROUP"), "Microsoft.Compute/virtualMachineScaleSets", scaleset)+"/virtualMachines/"+instanceID,
				),
				"status,operationName")
			if err != nil {
				return false, err
			}

			var count int
			for logs.NotDone() {
				for _, log := range logs.Values() {
					if *log.OperationName.Value == "Microsoft.Compute/virtualMachineScaleSets/virtualmachines/reimage/action" &&
						*log.Status.Value == "Succeeded" {
						count++
					}
				}
				err = logs.Next()
				if err != nil {
					return false, err
				}
			}

			return count == 1, nil
		})
		Expect(err).NotTo(HaveOccurred())

		By("Validating the cluster")
		errs := sanity.Checker.ValidateCluster(context.Background())
		Expect(errs).To(BeEmpty())
	})
})
