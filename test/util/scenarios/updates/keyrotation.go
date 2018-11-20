//+build e2e

package updates

import (
	"reflect"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openshift/openshift-azure/test/util/client/azure"
	"github.com/openshift/openshift-azure/test/util/client/cluster"
)

func RotateClusterCredentials(az *azure.Client, manifest, configBlob string) {
	By("Parsing the external manifest")
	external, err := cluster.ParseExternalConfig(manifest)
	Expect(err).NotTo(HaveOccurred())
	Expect(external).NotTo(BeNil())

	By("Parsing the internal manifest containing config blob")
	internal, err := cluster.ParseInternalConfig(configBlob)
	Expect(err).NotTo(HaveOccurred())
	Expect(internal).NotTo(BeNil())

	By("Deleting all non-ca cluster certificates and credentials from the config blob")
	mutated := cluster.DeleteSecrets(internal)
	Expect(err).NotTo(HaveOccurred())
	Expect(mutated).NotTo(BeNil())

	By("Running generate on the modified config blob")
	err = cluster.GenerateInternalConfig(mutated)
	Expect(err).NotTo(HaveOccurred())

	By("Persisting the config blob containing the new certificates and credentials")
	err = cluster.SaveConfig(mutated, configBlob)
	Expect(err).NotTo(HaveOccurred())

	By("Calling update on the fake rp with the updated config blob")
	updated, err := az.UpdateCluster(external, configBlob, cluster.NewPluginConfig())
	Expect(err).NotTo(HaveOccurred())
	Expect(updated).NotTo(BeNil())

	By("Parsing the config blob after the update")
	internalAfterUpdate, err := cluster.ParseInternalConfig(configBlob)
	Expect(err).NotTo(HaveOccurred())
	Expect(internalAfterUpdate).NotTo(BeNil())

	By("Verifying that the initial config blob does not match the one created after the update")
	configMatch := reflect.DeepEqual(internal.Config.Certificates, internalAfterUpdate.Config.Certificates)
	Expect(configMatch).To(BeFalse())

	By("Verifying that the mutated config blob matches the one created after the update")
	configMatch = reflect.DeepEqual(mutated.Config.Certificates, internalAfterUpdate.Config.Certificates)
	Expect(configMatch).To(BeTrue())
}
