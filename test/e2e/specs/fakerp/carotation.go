package fakerp

import (
	"context"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openshift/openshift-azure/test/clients/azure"
	"github.com/openshift/openshift-azure/test/sanity"
)

var _ = Describe("Ca Rotation E2E tests [CaRotation][Fake][LongRunning]", func() {
	It("should be possible to maintain a healthy cluster after rotating all certificates", func() {
		By("Reading the cluster state")
		before, err := azure.RPClient.OpenShiftManagedClustersAdmin.Get(context.Background(), os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"))
		Expect(err).NotTo(HaveOccurred())
		Expect(before).NotTo(BeNil())

		By("Executing Certificate rotation on the cluster.")
		err = azure.RPClient.OpenShiftManagedClustersAdmin.RotateCertificates(context.Background(), os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"))
		Expect(err).NotTo(HaveOccurred())

		By("Reading the cluster state after the update")
		after, err := azure.RPClient.OpenShiftManagedClustersAdmin.Get(context.Background(), os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"))
		Expect(err).NotTo(HaveOccurred())
		Expect(after).NotTo(BeNil())

		By("Verifying that the correct ca certificates have been updated")
		Expect(before.Config.Certificates.Ca.Cert).To(Equal(after.Config.Certificates.Ca.Cert))
		Expect(before.Config.Certificates.ServiceSigningCa.Cert).To(Equal(after.Config.Certificates.ServiceSigningCa.Cert))
		Expect(before.Config.Certificates.ServiceCatalogCa.Cert).To(Equal(after.Config.Certificates.ServiceCatalogCa.Cert))
		Expect(before.Config.Certificates.FrontProxyCa.Cert).To(Equal(after.Config.Certificates.FrontProxyCa.Cert))

		Expect(before.Config.Certificates.EtcdCa.Cert).NotTo(Equal(after.Config.Certificates.EtcdCa.Cert))
		Expect(before.Config.Certificates.EtcdClient.Cert).NotTo(Equal(after.Config.Certificates.EtcdClient.Cert))
		Expect(before.Config.Certificates.EtcdPeer.Cert).NotTo(Equal(after.Config.Certificates.EtcdPeer.Cert))
		Expect(before.Config.Certificates.EtcdServer.Cert).NotTo(Equal(after.Config.Certificates.EtcdServer.Cert))
		Expect(before.Config.Certificates.MasterProxyClient.Cert).NotTo(Equal(after.Config.Certificates.MasterProxyClient.Cert))
		Expect(before.Config.Certificates.MasterServer.Cert).NotTo(Equal(after.Config.Certificates.MasterServer.Cert))
		Expect(before.Config.Certificates.AggregatorFrontProxy.Cert).NotTo(Equal(after.Config.Certificates.AggregatorFrontProxy.Cert))

		By("Validating the cluster")
		errs := sanity.Checker.ValidateCluster(context.Background())
		Expect(errs).To(BeEmpty())
	})
})
