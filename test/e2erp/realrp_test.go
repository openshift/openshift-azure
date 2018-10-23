//+build e2erp

package e2erp

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/log"
)

var _ = Describe("Resource provider e2e tests [Real]", func() {
	defer GinkgoRecover()

	It("should not be possible for customer to mutate an osa scale set", func() {
		ctx := context.Background()
		ctx = context.WithValue(ctx, api.ContextKeyClientID, azureConf.ClientID)
		ctx = context.WithValue(ctx, api.ContextKeyClientSecret, azureConf.ClientSecret)
		ctx = context.WithValue(ctx, api.ContextKeyTenantID, azureConf.TenantID)

		logrus.SetLevel(log.SanitizeLogLevel("Debug"))
		logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
		logger := logrus.WithFields(logrus.Fields{"location": c.location, "resourceGroup": c.resourceGroup})

		appRg := ApplicationResourceGroup(c.resourceGroup, c.resourceGroup, c.location)
		Expect(appRg).NotTo(And(BeNil(), BeEmpty()))
		logger.Infof("application resource group is %s", appRg)

		// update the logger to use the application resource group
		logger = logrus.WithFields(logrus.Fields{"location": c.location, "resourceGroup": appRg})

		managedRg, err := ManagedResourceGroup(ctx, c.appsc, appRg)
		Expect(err).NotTo(HaveOccurred())
		Expect(appRg).NotTo(And(BeNil(), BeEmpty()))
		logger.Infof("managed resource group is %s", managedRg)

		// update the logger to use the managed resource group
		logger = logrus.WithFields(logrus.Fields{"location": c.location, "resourceGroup": managedRg})

		scaleSets, err := ScaleSets(ctx, logger, c.ssc, managedRg)
		Expect(err).NotTo(HaveOccurred())
		Expect(scaleSets).NotTo(And(BeNil(), BeEmpty()))
		Expect(len(scaleSets)).Should(Equal(3))

		// TODO: get detailed error and match on them since we expect the customer to see errors with Code=ScopeLocked
		var errs []error

		By("Updating the scale set instance count")
		errs = UpdateScaleSetsCapacity(ctx, logger, c.ssc, c.ssvmc, managedRg)
		Expect(errs).NotTo(BeNil())
		Expect(len(errs)).To(BeEquivalentTo(len(scaleSets)))

		By("Updating the scale set instance type")
		errs = UpdateScaleSetsInstanceType(ctx, logger, c.ssc, managedRg)
		Expect(errs).NotTo(BeNil())
		Expect(len(errs)).To(BeEquivalentTo(len(scaleSets)))

		By("Updating the scale set SSH key")
		errs = UpdateScaleSetSSHKey(ctx, logger, c.ssc, managedRg)
		Expect(errs).NotTo(BeNil())
		Expect(len(errs)).To(BeEquivalentTo(len(scaleSets)))

		var vmCount int
		for _, s := range scaleSets {
			scaleSetVMs, err := ScaleSetVMs(ctx, logger, c.ssvmc, managedRg, *s.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(scaleSetVMs).NotTo(And(BeNil(), BeEmpty()))
			vmCount = vmCount + len(scaleSetVMs)
		}

		By("Rebooting all scale set instances")
		errs = RebootScaleSetVMs(ctx, logger, c.ssc, c.ssvmc, managedRg)
		Expect(errs).NotTo(BeNil())
		Expect(len(errs)).To(BeEquivalentTo(vmCount))

		By("Creating scale set script extensions")
		errs = UpdateScaleSetScriptExtension(ctx, logger, c.ssc, c.ssec, managedRg)
		Expect(errs).NotTo(BeNil())
		Expect(len(errs)).To(BeEquivalentTo(len(scaleSets)))
	})
})
