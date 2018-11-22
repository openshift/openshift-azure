//+build e2erp

package e2erp

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

var _ = Describe("Resource provider e2e tests [Real]", func() {
	defer GinkgoRecover()

	It("should keep the end user from reading the config blob", func() {
		By(fmt.Sprintf("application resource group is %s", c.appResourceGroup))

		managedRg, err := ManagedResourceGroup(c.ctx, c.appsc, c.appResourceGroup)
		Expect(err).NotTo(HaveOccurred())
		Expect(managedRg).NotTo(And(BeNil(), BeEmpty()))
		By(fmt.Sprintf("managed resource group is %s", managedRg))
		accts, err := c.accsc.ListByResourceGroup(c.ctx, managedRg)
		Expect(err).NotTo(HaveOccurred())
		Expect(accts).NotTo(BeNil())
		for _, acct := range *accts.Value {
			By(fmt.Sprintf("trying to read account %s", *acct.Name))
			if acct.Tags["type"] != nil && *acct.Tags["type"] == "config" {
				// should throw an error when trying to list the keys with the given name
				_, err := c.accsc.ListKeys(c.ctx, managedRg, *acct.Name)
				Expect(err).To(HaveOccurred())
				if err != nil {
					By(fmt.Sprintf("can't read %s, OK", *acct.Name))
				}
			} else {
				By(fmt.Sprintf("account %s is not a config account", *acct.Name))
			}
		}
	})

	It("should not be possible for customer to mutate an osa scale set", func() {
		managedRg, err := ManagedResourceGroup(c.ctx, c.appsc, c.appResourceGroup)
		Expect(err).NotTo(HaveOccurred())
		logrus.Infof("managed resource group is %s", managedRg)

		scaleSets, err := ScaleSets(c.ctx, c.ssc, managedRg)
		Expect(err).NotTo(HaveOccurred())
		Expect(scaleSets).NotTo(And(BeNil(), BeEmpty()))
		Expect(len(scaleSets)).Should(Equal(3))

		// TODO: get detailed error and match on them since we expect the customer to see errors with Code=ScopeLocked
		var errs []error

		By("Updating the scale set instance count")
		errs = UpdateScaleSetsCapacity(c.ctx, c.ssc, c.ssvmc, managedRg)
		Expect(errs).NotTo(BeNil())
		Expect(len(errs)).To(BeEquivalentTo(len(scaleSets)))

		By("Updating the scale set instance type")
		errs = UpdateScaleSetsInstanceType(c.ctx, c.ssc, managedRg)
		Expect(errs).NotTo(BeNil())
		Expect(len(errs)).To(BeEquivalentTo(len(scaleSets)))

		By("Updating the scale set SSH key")
		errs = UpdateScaleSetSSHKey(c.ctx, c.ssc, managedRg)
		Expect(errs).NotTo(BeNil())
		Expect(len(errs)).To(BeEquivalentTo(len(scaleSets)))

		var vmCount int
		for _, s := range scaleSets {
			scaleSetVMs, err := ScaleSetVMs(c.ctx, c.ssvmc, managedRg, *s.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(scaleSetVMs).NotTo(And(BeNil(), BeEmpty()))
			vmCount = vmCount + len(scaleSetVMs)
		}

		By("Rebooting all scale set instances")
		errs = RebootScaleSetVMs(c.ctx, c.ssc, c.ssvmc, managedRg)
		Expect(errs).NotTo(BeNil())
		Expect(len(errs)).To(BeEquivalentTo(vmCount))

		By("Creating scale set script extensions")
		errs = UpdateScaleSetScriptExtension(c.ctx, c.ssc, c.ssec, managedRg)
		Expect(errs).NotTo(BeNil())
		Expect(len(errs)).To(BeEquivalentTo(len(scaleSets)))
	})
})
