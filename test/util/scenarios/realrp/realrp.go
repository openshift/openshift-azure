//+build e2e

package realrp

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openshift/openshift-azure/test/util/client/azure"
)

func TestCustomerCannotModifyScaleSet(az *azure.Client) {
	appRg := az.ApplicationResourceGroup()
	Expect(appRg).NotTo(And(BeNil(), BeEmpty()))

	managedRg, err := az.ManagedResourceGroup(appRg)
	Expect(err).NotTo(HaveOccurred())
	Expect(appRg).NotTo(And(BeNil(), BeEmpty()))

	scaleSets, err := az.ScaleSets(managedRg)
	Expect(err).NotTo(HaveOccurred())
	Expect(scaleSets).NotTo(And(BeNil(), BeEmpty()))
	Expect(len(scaleSets)).Should(Equal(3))

	// TODO: get detailed error and match on them since we expect the customer to see errors with Code=ScopeLocked
	var errs []error

	By("Updating the scale set instance count")
	errs = az.UpdateScaleSetsCapacity(managedRg)
	Expect(errs).NotTo(BeNil())
	Expect(len(errs)).To(BeEquivalentTo(len(scaleSets)))

	By("Updating the scale set instance type")
	errs = az.UpdateScaleSetsInstanceType(managedRg)
	Expect(errs).NotTo(BeNil())
	Expect(len(errs)).To(BeEquivalentTo(len(scaleSets)))

	By("Updating the scale set SSH key")
	errs = az.UpdateScaleSetSSHKey(managedRg)
	Expect(errs).NotTo(BeNil())
	Expect(len(errs)).To(BeEquivalentTo(len(scaleSets)))

	var vmCount int
	for _, s := range scaleSets {
		scaleSetVMs, err := az.ScaleSetVMs(managedRg, *s.Name)
		Expect(err).NotTo(HaveOccurred())
		Expect(scaleSetVMs).NotTo(And(BeNil(), BeEmpty()))
		vmCount = vmCount + len(scaleSetVMs)
	}

	By("Rebooting all scale set instances")
	errs = az.RebootScaleSetVMs(managedRg)
	Expect(errs).NotTo(BeNil())
	Expect(len(errs)).To(BeEquivalentTo(vmCount))

	By("Creating scale set script extensions")
	errs = az.UpdateScaleSetScriptExtension(managedRg)
	Expect(errs).NotTo(BeNil())
	Expect(len(errs)).To(BeEquivalentTo(len(scaleSets)))
}
