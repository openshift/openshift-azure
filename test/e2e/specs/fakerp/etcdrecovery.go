package fakerp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift/openshift-azure/pkg/util/random"
	"github.com/openshift/openshift-azure/test/clients/azure"
	"github.com/openshift/openshift-azure/test/e2e/standard"
)

var _ = Describe("Etcd Recovery E2E tests [EtcdRecovery][Fake][LongRunning]", func() {
	const (
		configMapName = "recovery-test-data"
	)
	var (
		cli       *standard.SanityChecker
		azurecli  *azure.Client
		backup    string
		namespace string
	)

	BeforeEach(func() {
		var err error
		cli, err = standard.NewDefaultSanityChecker()
		Expect(cli).ToNot(BeNil())
		azurecli, err = azure.NewClientFromEnvironment(false)
		Expect(err).ToNot(HaveOccurred())

		backup, err = random.LowerCaseAlphanumericString(5)
		Expect(err).ToNot(HaveOccurred())
		backup = "e2e-test-" + backup
		namespace, err = random.LowerCaseAlphanumericString(5)
		Expect(err).ToNot(HaveOccurred())
		namespace = "e2e-test-" + namespace
		fmt.Fprintln(GinkgoWriter, "Using namespace", namespace)
		err = cli.Client.EndUser.CreateProject(namespace)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		cli.Client.EndUser.CleanupProject(namespace)
		cli.Client.Admin.BatchV1.Jobs("openshift-etcd").Delete("e2e-test-etcdbackup", nil)
	})

	It("should be possible to recover etcd from a backup", func() {
		resourceGroup := os.Getenv("RESOURCEGROUP")
		if resourceGroup == "" {
			Expect(errors.New("RESOURCEGROUP is not set")).NotTo(BeNil())
		}

		By("Create a test configmap with value=first")
		cm1, err := cli.Client.EndUser.CoreV1.ConfigMaps(namespace).Create(&v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      configMapName,
				Namespace: namespace,
			},
			Data: map[string]string{
				"value": "before-backup",
			},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(cm1.Data).To(HaveKeyWithValue("value", "before-backup"))

		By(fmt.Sprintf("Running an etcd backup"))
		resp, err := azurecli.OpenShiftManagedClustersAdmin.BackupAndWait(context.Background(), resourceGroup, resourceGroup, backup)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		// wait for it to exist
		By("Overwrite the test configmap with value=second")
		cm2, err := cli.Client.EndUser.CoreV1.ConfigMaps(namespace).Update(&v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      configMapName,
				Namespace: namespace,
			},
			Data: map[string]string{
				"value": "after-backup",
			},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(cm2.Data).To(HaveKeyWithValue("value", "after-backup"))

		By("Restore from the backup")
		resp, err = azurecli.OpenShiftManagedClustersAdmin.RestoreAndWait(context.Background(), resourceGroup, resourceGroup, backup)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		By("Confirm the state of the backup")
		final, err := cli.Client.EndUser.CoreV1.ConfigMaps(namespace).Get(configMapName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(final.Data).To(HaveKeyWithValue("value", "before-backup"))

		By("Validating the cluster")
		errs := cli.ValidateCluster(context.Background())
		Expect(errs).To(BeEmpty())
	})
})
