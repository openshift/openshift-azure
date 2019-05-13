package fakerp

import (
	"context"
	"fmt"
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

var _ = Describe("Command tests [Command][Fake][LongRunning]", func() {
	It("should be possible for an SRE to restart system services on vms", func() {
		vm := "master-000000"

		startTime := time.Now()

		err := azure.RPClient.OpenShiftManagedClustersAdmin.RunCommand(context.Background(), os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"), vm, "RestartKubelet")
		Expect(err).NotTo(HaveOccurred())

		err = azure.RPClient.OpenShiftManagedClustersAdmin.RunCommand(context.Background(), os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"), vm, "RestartDocker")
		Expect(err).NotTo(HaveOccurred())

		err = azure.RPClient.OpenShiftManagedClustersAdmin.RunCommand(context.Background(), os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"), vm, "RestartNetworkManager")
		Expect(err).NotTo(HaveOccurred())

		scaleset, _, err := names.GetScaleSetNameAndInstanceID(vm)
		Expect(err).NotTo(HaveOccurred())

		wait.PollImmediate(10*time.Second, 2*time.Minute, func() (bool, error) {
			By("Verifying through azure activity logs that the command ran")
			logs, err := azure.RPClient.ActivityLogs.List(
				context.Background(),
				fmt.Sprintf("eventTimestamp ge '%s' and resourceUri eq %s",
					startTime.Format(time.RFC3339),
					resourceid.ResourceID(os.Getenv("AZURE_SUBSCRIPTION_ID"), os.Getenv("RESOURCEGROUP"), "Microsoft.Compute/virtualMachineScaleSets", scaleset),
				),
				"status,operationName")
			if err != nil {
				return false, err
			}

			var count int
			for logs.NotDone() {
				for _, log := range logs.Values() {
					if *log.OperationName.Value == "Microsoft.Compute/virtualMachineScaleSets/virtualmachines/runCommand/action" &&
						*log.Status.Value == "Succeeded" {
						count++
					}
				}
				err = logs.Next()
				if err != nil {
					return false, err
				}
			}

			return count == 3, nil
		})
		Expect(err).NotTo(HaveOccurred())

		By("Validating the cluster")
		errs := sanity.Checker.ValidateCluster(context.Background())
		Expect(errs).To(BeEmpty())
	})
})
