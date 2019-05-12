package realrp

import (
	"context"
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"
	"github.com/Azure/go-autorest/autorest/to"

	v20190430 "github.com/openshift/openshift-azure/pkg/api/2019-04-30"
	"github.com/openshift/openshift-azure/pkg/fakerp/client"
	"github.com/openshift/openshift-azure/test/clients/azure"
	"github.com/openshift/openshift-azure/test/util/log"
)

var _ = Describe("Resource provider e2e tests [Default][Real]", func() {
	var (
		ctx = context.Background()
		cli *azure.Client
		cfg *client.Config
	)

	// NOTE: Ensure this is always the first test in the [Default][Real] spec!
	It("should deploy a cluster using the production RP", func() {
		var err error
		cli, err = azure.NewClientFromEnvironment(ctx, log.GetTestLogger(), false)
		Expect(err).ToNot(HaveOccurred())

		cfg, err = client.NewConfig(log.GetTestLogger())
		Expect(err).NotTo(HaveOccurred())

		// create a new resource group
		err = client.EnsureResourceGroup(cfg)
		Expect(err).ToNot(HaveOccurred())

		By("creating an OSA cluster")
		var config v20190430.OpenShiftManagedCluster
		err = client.GenerateManifest(cfg, "../../test/manifests/realrp/create.yaml", &config)
		Expect(err).ToNot(HaveOccurred())
		deployCtx, cancelFn := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancelFn()
		_, err = cli.OpenShiftManagedClusters.CreateOrUpdateAndWait(deployCtx, cfg.ResourceGroup, cfg.ResourceGroup, config)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should keep the end user from deleting any azure resources", func() {
		resourcegroup, err := cli.OSAResourceGroup(ctx, cfg.ResourceGroup, cfg.ResourceGroup, cfg.Region)
		Expect(err).NotTo(HaveOccurred())
		By(fmt.Sprintf("OSA resource group is %s", resourcegroup))

		pages, err := cli.Resources.ListByResourceGroup(ctx, resourcegroup, "", "", nil)
		Expect(err).ToNot(HaveOccurred())
		// attempt to delete all resources in the resourcegroup
		for pages.NotDone() {
			for _, v := range pages.Values() {
				By(fmt.Sprintf("trying to delete %s/%s", *v.Type, *v.Name))
				_, err := cli.Resources.DeleteByID(ctx, *v.ID)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring(`StatusCode=403`))
			}
			err = pages.Next()
			Expect(err).NotTo(HaveOccurred())
		}
	})

	It("should keep the end user from reading the config blob", func() {
		resourcegroup, err := cli.OSAResourceGroup(ctx, cfg.ResourceGroup, cfg.ResourceGroup, cfg.Region)
		Expect(err).NotTo(HaveOccurred())
		By(fmt.Sprintf("OSA resource group is %s", resourcegroup))

		accts, err := cli.Accounts.ListByResourceGroup(ctx, resourcegroup)
		Expect(err).NotTo(HaveOccurred())

		for _, acct := range *accts.Value {
			By(fmt.Sprintf("trying to read account %s", *acct.Name))
			_, err := cli.Accounts.ListKeys(ctx, resourcegroup, *acct.Name)
			Expect(err).To(HaveOccurred())
		}
	})

	It("should not be possible for customer to mutate an osa scale set", func() {
		const (
			fakepubkey = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC7laRyN4B3YZmVrDEZLZoIuUA72pQ0DpGuZBZWykCofIfCPrFZAJgFvonKGgKJl6FGKIunkZL9Us/mV4ZPkZhBlE7uX83AAf5i9Q8FmKpotzmaxN10/1mcnEE7pFvLoSkwqrQSkrrgSm8zaJ3g91giXSbtqvSIj/vk2f05stYmLfhAwNo3Oh27ugCakCoVeuCrZkvHMaJgcYrIGCuFo6q0Pfk9rsZyriIqEa9AtiUOtViInVYdby7y71wcbl0AbbCZsTSqnSoVxm2tRkOsXV6+8X4SnwcmZbao3H+zfO1GBhQOLxJ4NQbzAa8IJh810rYARNLptgmsd4cYXVOSosTX azureuser"
		)

		resourcegroup, err := cli.OSAResourceGroup(ctx, cfg.ResourceGroup, cfg.ResourceGroup, cfg.Region)
		Expect(err).NotTo(HaveOccurred())
		By(fmt.Sprintf("OSA resource group is %s", resourcegroup))

		scaleSets, err := cli.VirtualMachineScaleSets.List(ctx, resourcegroup)
		Expect(err).NotTo(HaveOccurred())
		Expect(scaleSets).Should(HaveLen(3))

		for _, scaleSet := range scaleSets {
			vms, err := cli.VirtualMachineScaleSetVMs.List(ctx, resourcegroup, *scaleSet.Name, "", "", "")
			Expect(err).NotTo(HaveOccurred())

			By("trying to update the scale set capacity")
			err = cli.VirtualMachineScaleSets.Update(ctx, resourcegroup, *scaleSet.Name, compute.VirtualMachineScaleSetUpdate{
				Sku: &compute.Sku{
					Capacity: to.Int64Ptr(int64(len(vms) + 1)),
				},
			})
			Expect(err).To(HaveOccurred())

			By("trying to update the scale set type")
			err = cli.VirtualMachineScaleSets.Update(ctx, resourcegroup, *scaleSet.Name, compute.VirtualMachineScaleSetUpdate{
				Sku: &compute.Sku{
					Name: to.StringPtr("Standard_DS1_v2"),
				},
			})
			Expect(err).To(HaveOccurred())

			By("trying to update the scale set SSH key")
			err = cli.VirtualMachineScaleSets.Update(ctx, resourcegroup, *scaleSet.Name, compute.VirtualMachineScaleSetUpdate{
				VirtualMachineScaleSetUpdateProperties: &compute.VirtualMachineScaleSetUpdateProperties{
					VirtualMachineProfile: &compute.VirtualMachineScaleSetUpdateVMProfile{
						OsProfile: &compute.VirtualMachineScaleSetUpdateOSProfile{
							LinuxConfiguration: &compute.LinuxConfiguration{
								SSH: &compute.SSHConfiguration{
									PublicKeys: &[]compute.SSHPublicKey{
										{
											Path:    to.StringPtr("/home/cloud-user/.ssh/authorized_keys"),
											KeyData: to.StringPtr(fakepubkey),
										},
									},
								},
							},
						},
					},
				},
			})

			Expect(err).To(HaveOccurred())

			By("trying to create scale set script extension")
			_, err = cli.VirtualMachineScaleSetExtensions.CreateOrUpdate(ctx, resourcegroup, *scaleSet.Name, "test", compute.VirtualMachineScaleSetExtension{
				VirtualMachineScaleSetExtensionProperties: &compute.VirtualMachineScaleSetExtensionProperties{
					Type:     to.StringPtr("CustomScript"),
					Settings: `{"fileUris":["https://raw.githubusercontent.com/Azure-Samples/compute-automation-configurations/master/automate_nginx.sh"],"commandToExecute":"./automate_nginx.sh"}`,
				},
			})
			Expect(err).To(HaveOccurred())

			for _, vm := range vms {
				By("trying to restart scale set instance vm")
				err = cli.VirtualMachineScaleSetVMs.Restart(ctx, resourcegroup, *scaleSet.Name, *vm.InstanceID)
				Expect(err).To(HaveOccurred())
			}
		}
	})

	// NOTE: Ensure this is always the last test in the [Default][Real] spec!
	It("should delete a cluster using the production RP", func() {
		deleteCtx, cancelFn := context.WithTimeout(context.Background(), time.Hour)
		defer cancelFn()
		By("deleting OSA resource")
		future, err := cli.OpenShiftManagedClusters.Delete(deleteCtx, cfg.ResourceGroup, cfg.ResourceGroup)
		Expect(err).NotTo(HaveOccurred())
		// Avoid failing while waiting for the OSA resource to get cleaned up
		// since the delete code is not super-stable atm.
		err = future.WaitForCompletionRef(deleteCtx, cli.OpenShiftManagedClusters.Client)
		if err != nil {
			fmt.Fprintf(GinkgoWriter, "error while waiting for OSA resource to get cleaned up: %v", err)
		} else {
			resp, err := future.Result(cli.OpenShiftManagedClusters)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		}

		By(fmt.Sprintf("deleting resource group %s", cfg.ResourceGroup))
		err = cli.Groups.Delete(deleteCtx, cfg.ResourceGroup)
		Expect(err).NotTo(HaveOccurred())
	})
})
