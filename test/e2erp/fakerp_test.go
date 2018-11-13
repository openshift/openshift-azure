//+build e2erp

package e2erp

import (
	"reflect"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Resource provider e2e tests [Fake]", func() {
	defer GinkgoRecover()

	It("should be possible to maintain a healthy cluster after rotating all credentials", func() {
		By("Parsing the external manifest")
		external, err := parseExternalConfig(*manifest)
		Expect(err).NotTo(HaveOccurred())
		Expect(external).NotTo(BeNil())

		By("Parsing the internal manifest containing config blob")
		internal, err := parseInternalConfig(*configBlob)
		Expect(err).NotTo(HaveOccurred())
		Expect(internal).NotTo(BeNil())

		By("Deleting all non-ca cluster certificates and credentials from the config blob")
		mutated := deleteCertificates(internal)
		Expect(err).NotTo(HaveOccurred())
		Expect(mutated).NotTo(BeNil())

		By("Running generate on the modified config blob")
		err = generateInternalConfig(mutated, pluginConfig)
		Expect(err).NotTo(HaveOccurred())

		By("Persisting the config blob containing the new certificates and credentials")
		err = saveConfig(mutated, *configBlob)
		Expect(err).NotTo(HaveOccurred())

		By("Calling update on the fake rp with the updated config blob")
		updated, err := updateCluster(ctx, external, *configBlob, logger, pluginConfig)
		Expect(err).NotTo(HaveOccurred())
		Expect(updated).NotTo(BeNil())

		By("Parsing the config blob after the update")
		internalAfterUpdate, err := parseInternalConfig(*configBlob)
		Expect(err).NotTo(HaveOccurred())
		Expect(internalAfterUpdate).NotTo(BeNil())

		By("Verifying that the initial config blob does not match the one created after the update")
		configMatch := reflect.DeepEqual(internal.Config.Certificates, internalAfterUpdate.Config.Certificates)
		Expect(configMatch).To(BeFalse())

		By("Verifying that the mutated config blob matches the one created after the update")
		configMatch = reflect.DeepEqual(mutated.Config.Certificates, internalAfterUpdate.Config.Certificates)
		Expect(configMatch).To(BeTrue())
	})
})
