package fakerp

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/ghodss/yaml"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/fakerp"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/test/clients/azure"
	"github.com/openshift/openshift-azure/test/sanity"
	logger "github.com/openshift/openshift-azure/test/util/log"
)

var _ = Describe("Apply security updates E2E tests [ApplySecurityUpdates][Fake][LongRunning]", func() {
	var (
		ctx                    = context.Background()
		securityUpdatePackages = []string{
			"nano",
			"wget",
		}
		log = logger.GetTestLogger()
		// Do not use the internal config in other tests! This is necessary here in order to acquire
		// ssh credentials which will be used to query the sample vm for rpm lists before and
		// after cve hot patches are applied to a cluster. Usually fakerp tests should be written
		// to use the admin config wherever possible.
		internalConfig = func() (*api.OpenShiftManagedCluster, error) {
			var cs api.OpenShiftManagedCluster
			b, err := ioutil.ReadFile("../../_data/containerservice.yaml")
			if err != nil {
				return nil, err
			}
			err = yaml.Unmarshal(b, &cs)
			return &cs, err
		}
		sampleVm = "master-000000"
	)

	It("should be possible for an SRE to apply security updates to a cluster", func() {
		By("Reading the admin config before the security updates")
		beforeUpdate, err := azure.RPClient.OpenShiftManagedClustersAdmin.Get(ctx, os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"))
		Expect(err).NotTo(HaveOccurred())
		Expect(beforeUpdate).NotTo(BeNil())

		By("Reading the internal config (not converting from the admin config) for access to secrets for ssh access")
		internal, err := internalConfig()
		Expect(err).NotTo(HaveOccurred())
		Expect(internal).NotTo(BeNil())

		By("Creating ssh client connection to sample vm")
		authorizer, err := azureclient.NewAuthorizerFromEnvironment("")
		ctx = context.WithValue(ctx, api.ContextKeyClientAuthorizer, authorizer)
		ssher, err := fakerp.NewSSHer(ctx, log, internal)
		Expect(err).NotTo(HaveOccurred())
		Expect(ssher).NotTo(BeNil())
		sshcli, err := ssher.Dial(ctx, sampleVm)
		Expect(err).NotTo(HaveOccurred())
		Expect(sshcli).NotTo(BeNil())

		By("Searching for installed patch packages on sample vm before security updates")
		rpmsBeforeUpdate, err := ssher.RunRemoteCommand(sshcli, fmt.Sprintf("sudo rpm -qa | grep -E %q", strings.Join(securityUpdatePackages, "|")))
		Expect(err).NotTo(HaveOccurred())

		By("Setting list of patch RPMs on admin config")
		beforeUpdate.Config.SecurityPatchPackages = to.StringSlicePtr(securityUpdatePackages)

		By("Executing a cluster update to install patch RPMs")
		_, err = azure.RPClient.OpenShiftManagedClustersAdmin.CreateOrUpdate(ctx, os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"), beforeUpdate)
		Expect(err).NotTo(HaveOccurred())

		By("Reading the admin config after installing patch RPMs")
		afterUpdate, err := azure.RPClient.OpenShiftManagedClustersAdmin.Get(ctx, os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"))
		Expect(err).NotTo(HaveOccurred())
		Expect(afterUpdate).NotTo(BeNil())

		By("Refreshing ssh client connection to sample vm")
		sshcli, err = ssher.Dial(ctx, sampleVm)
		Expect(err).NotTo(HaveOccurred())
		Expect(sshcli).NotTo(BeNil())

		By("Searching for updated packages on sample vm after applying security updates")
		rpmsAfterUpdate, err := ssher.RunRemoteCommand(sshcli, fmt.Sprintf("sudo rpm -qa | grep -E %q", strings.Join(securityUpdatePackages, "|")))
		Expect(err).NotTo(HaveOccurred())

		By("Reading the security updates logs on sample vm")
		securityUpdateLogs, err := ssher.RunRemoteCommand(sshcli, "sudo journalctl -t master-startup.sh -t node-startup.sh")
		Expect(err).NotTo(HaveOccurred())

		By("Verifying that the security updates have been installed and present in the cluster's config")
		Expect(afterUpdate.Config.SecurityPatchPackages).To(Equal(beforeUpdate.Config.SecurityPatchPackages))
		Expect(string(rpmsBeforeUpdate)).NotTo(Equal(string(rpmsAfterUpdate)))
		for _, rpm := range securityUpdatePackages {
			Expect(string(rpmsAfterUpdate)).To(ContainSubstring(rpm))
		}
		Expect(securityUpdateLogs).To(ContainSubstring(fmt.Sprintf("installing red hat cdn configuration on %s", sampleVm)))
		Expect(securityUpdateLogs).To(ContainSubstring(fmt.Sprintf("installing ARO security updates [%s] on %s", strings.Join(securityUpdatePackages, ", "), sampleVm)))
		Expect(securityUpdateLogs).To(ContainSubstring(fmt.Sprintf("removing red hat cdn configuration on %s", sampleVm)))

		By("Validating the cluster")
		errs := sanity.Checker.ValidateCluster(context.Background())
		Expect(errs).To(BeEmpty())
	})
})
