package fakerp

import (
	"context"
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/openshift/openshift-azure/pkg/config"
	"github.com/openshift/openshift-azure/pkg/util/resourceid"
	"github.com/openshift/openshift-azure/test/clients/azure"
	"github.com/openshift/openshift-azure/test/e2e/standard"
)

var _ = Describe("Command tests [Command][Fake][LongRunning]", func() {
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

	It("should be possible for an SRE to restart Kubelet and NetworkManager", func() {
		vm := "master-000000"

		startTime := time.Now()

		err := azurecli.OpenShiftManagedClustersAdmin.RunCommand(context.Background(), os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"), vm, "RestartKubelet")
		Expect(err).NotTo(HaveOccurred())

		err = azurecli.OpenShiftManagedClustersAdmin.RunCommand(context.Background(), os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"), vm, "RestartNetworkManager")
		Expect(err).NotTo(HaveOccurred())

		scaleset, _, err := config.GetScaleSetNameAndInstanceID(vm)
		Expect(err).NotTo(HaveOccurred())

		wait.PollImmediate(10*time.Second, 2*time.Minute, func() (bool, error) {
			By("Verifying through azure activity logs that the reimage happened")
			logs, err := azurecli.ActivityLogs.List(
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

			return count == 2, nil
		})
		Expect(err).NotTo(HaveOccurred())

		By("Validating the cluster")
		errs := cli.ValidateCluster(context.Background())
		Expect(errs).To(BeEmpty())
	})
})
