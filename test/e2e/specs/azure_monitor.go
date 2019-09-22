package specs

import (
	"context"
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Azure/azure-sdk-for-go/services/preview/operationalinsights/mgmt/2015-11-01-preview/operationalinsights"
	azresources "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest/to"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/openshift/openshift-azure/pkg/util/random"
	"github.com/openshift/openshift-azure/pkg/util/ready"
	"github.com/openshift/openshift-azure/test/clients/azure"
	"github.com/openshift/openshift-azure/test/manifests"
	"github.com/openshift/openshift-azure/test/sanity"
)

const (
	azureMonitorProject = "openshift-azure-logging"
	nodeAgentName       = "log-analytics-node-agent"
	clusterAgentName    = "log-analytics-cluster-agent"
)

var _ = Describe("Azure Red Hat OpenShift e2e tests for Azure Monitor Integration [EveryPR]", func() {
	withAzureMonitorIt("should be possible for a customer to update the azure monitor configuration for their cluster [EndUser]", func() {
		ctx := context.Background()
		rgName, err := random.LowerCaseAlphanumericString(5)
		Expect(err).ToNot(HaveOccurred())
		rgName = "e2e-workspace-" + rgName

		By("Reading the cluster config before the azure monitor checks")
		clusterConfig, err := azure.RPClient.OpenShiftManagedClusters.Get(ctx, os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"))
		Expect(err).NotTo(HaveOccurred())
		Expect(clusterConfig).NotTo(BeNil())
		Expect(*clusterConfig.Properties.MonitorProfile.Enabled).To(Equal(true))
		originalWorkapceId := *clusterConfig.Properties.MonitorProfile.WorkspaceResourceID

		By(fmt.Sprintf("Creating workspace resource group %s", rgName))
		workspaceGroup, err := azure.RPClient.Groups.CreateOrUpdate(ctx, rgName, azresources.Group{
			Location: clusterConfig.Location,
			Tags: map[string]*string{
				"now": to.StringPtr(fmt.Sprintf("%d", time.Now().Unix())),
				"ttl": to.StringPtr("3h"),
			},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(workspaceGroup).NotTo(BeNil())
		defer azure.RPClient.Groups.Delete(ctx, rgName)

		By(fmt.Sprintf("Creating workspace %s in resource group %s", rgName, rgName))
		workspace, err := azure.RPClient.Workspaces.CreateOrUpdate(ctx, rgName, rgName, operationalinsights.Workspace{
			Location: clusterConfig.Location,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(workspace).NotTo(BeNil())

		By("Waiting for the log analytics agents to get ready")
		err = wait.PollImmediate(2*time.Second, 5*time.Minute, ready.CheckDaemonSetIsReady(sanity.Checker.Client.Admin.AppsV1.DaemonSets(azureMonitorProject), nodeAgentName))
		Expect(err).NotTo(HaveOccurred())
		err = wait.PollImmediate(2*time.Second, 5*time.Minute, ready.CheckDeploymentIsReady(sanity.Checker.Client.Admin.AppsV1.Deployments(azureMonitorProject), clusterAgentName))
		Expect(err).NotTo(HaveOccurred())

		By("Update cluster config to disable azure monitor and update the workspace resource id")
		*clusterConfig.Properties.MonitorProfile.Enabled = false
		newWorkSpaceId := fmt.Sprintf("/subscriptions/225e02bc-43d0-43d1-a01a-17e584a4ef69/resourcegroups/"+
			"%s/providers/microsoft.operationalinsights/workspaces/%s",
			rgName, rgName)
		*clusterConfig.Properties.MonitorProfile.WorkspaceResourceID = newWorkSpaceId

		By("Executing a cluster update")
		firstUpdate, err := azure.RPClient.OpenShiftManagedClusters.CreateOrUpdateAndWait(ctx, os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"), clusterConfig)
		Expect(err).NotTo(HaveOccurred())
		Expect(firstUpdate).NotTo(BeNil())
		Expect(*firstUpdate.Properties.MonitorProfile.WorkspaceResourceID).To(Equal(newWorkSpaceId))

		By("Verifying that the log analytics agents are deleted")
		err = wait.PollImmediate(2*time.Second, 5*time.Minute, func() (done bool, err error) {
			_, err = sanity.Checker.Client.Admin.AppsV1.Deployments(azureMonitorProject).Get(clusterAgentName, metav1.GetOptions{})
			switch {
			case errors.IsNotFound(err):
				return true, nil
			case err != nil:
				return false, err
			}
			return false, nil
		})
		Expect(err).NotTo(HaveOccurred())
		err = wait.PollImmediate(2*time.Second, 5*time.Minute, func() (done bool, err error) {
			_, err = sanity.Checker.Client.Admin.AppsV1.DaemonSets(azureMonitorProject).Get(nodeAgentName, metav1.GetOptions{})
			switch {
			case errors.IsNotFound(err):
				return true, nil
			case err != nil:
				return false, err
			}
			return false, nil
		})
		Expect(err).NotTo(HaveOccurred())

		By("Update cluster config to re-enable azure monitor and the original workspace")
		*firstUpdate.Properties.MonitorProfile.Enabled = true
		*firstUpdate.Properties.MonitorProfile.WorkspaceResourceID = originalWorkapceId

		By("Executing a cluster update")
		secondUpdate, err := azure.RPClient.OpenShiftManagedClusters.CreateOrUpdateAndWait(ctx, os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"), firstUpdate)
		Expect(err).NotTo(HaveOccurred())
		Expect(*secondUpdate.Properties.MonitorProfile.Enabled).To(Equal(true))
		Expect(*secondUpdate.Properties.MonitorProfile.WorkspaceResourceID).To(Equal(originalWorkapceId))

		By("Waiting for the log analytics agents to get ready")
		err = wait.PollImmediate(2*time.Second, 5*time.Minute, ready.CheckDaemonSetIsReady(sanity.Checker.Client.Admin.AppsV1.DaemonSets(azureMonitorProject), nodeAgentName))
		Expect(err).NotTo(HaveOccurred())
		err = wait.PollImmediate(2*time.Second, 5*time.Minute, ready.CheckDeploymentIsReady(sanity.Checker.Client.Admin.AppsV1.Deployments(azureMonitorProject), clusterAgentName))
		Expect(err).NotTo(HaveOccurred())
	})
})

// withAzureMonitorIt wraps It blocks to make sure the tests are run on plugin versions supporting Azure Monitor (>= v9.0).
func withAzureMonitorIt(description string, f interface{}) {
	RegisterFailHandler(Fail)
	internal, err := manifests.InternalConfig()
	Expect(err).NotTo(HaveOccurred())

	var major, minor int
	_, err = fmt.Sscanf(internal.Config.PluginVersion, "v%d.%d", &major, &minor)
	Expect(err).NotTo(HaveOccurred())

	if major >= 9 {
		It(description, f)
	} else {
		By(fmt.Sprintf("Skipping azure monitor test for plugin major version %d", major))
		PIt(description, f)
	}
}
