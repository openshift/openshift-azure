package fakerp

import (
	"context"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openshift/openshift-azure/test/clients/azure"
	"github.com/openshift/openshift-azure/test/sanity"
)

var _ = Describe("Key Rotation E2E tests [KeyRotation][Fake][LongRunning]", func() {
	It("should be possible to maintain a healthy cluster after rotating all credentials", func() {
		By("Reading the cluster state")
		before, err := azure.FakeRPClient.OpenShiftManagedClustersAdmin.Get(context.Background(), os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"))
		Expect(err).NotTo(HaveOccurred())
		Expect(before).NotTo(BeNil())

		By("Executing key rotation on the cluster.")
		err = azure.FakeRPClient.OpenShiftManagedClustersAdmin.RotateSecrets(context.Background(), os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"))
		Expect(err).NotTo(HaveOccurred())

		By("Reading the cluster state after the update")
		after, err := azure.FakeRPClient.OpenShiftManagedClustersAdmin.Get(context.Background(), os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"))
		Expect(err).NotTo(HaveOccurred())
		Expect(after).NotTo(BeNil())

		By("Verifying that ca certificates have not been updated")
		Expect(before.Config.Certificates.ServiceSigningCa.Cert).To(Equal(after.Config.Certificates.ServiceSigningCa.Cert))
		Expect(before.Config.Certificates.ServiceCatalogCa.Cert).To(Equal(after.Config.Certificates.ServiceCatalogCa.Cert))
		Expect(before.Config.Certificates.FrontProxyCa.Cert).To(Equal(after.Config.Certificates.FrontProxyCa.Cert))
		Expect(before.Config.Certificates.EtcdCa.Cert).To(Equal(after.Config.Certificates.EtcdCa.Cert))
		Expect(before.Config.Certificates.Ca.Cert).To(Equal(after.Config.Certificates.Ca.Cert))

		By("Validating the cluster")
		errs := sanity.Checker.ValidateCluster(context.Background())
		Expect(errs).To(BeEmpty())
	})
})
