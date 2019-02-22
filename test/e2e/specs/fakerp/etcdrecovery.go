package fakerp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/openshift/openshift-azure/pkg/util/randomstring"
	"github.com/openshift/openshift-azure/pkg/util/ready"
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

		backup, err = randomstring.RandomString("abcdefghijklmnopqrstuvwxyz0123456789", 5)
		Expect(err).ToNot(HaveOccurred())
		backup = "e2e-test-" + backup
		namespace, err = randomstring.RandomString("abcdefghijklmnopqrstuvwxyz0123456789", 5)
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

		backupImage := "quay.io/openshift-on-azure/etcdbackup:latest"
		// Enable backup code PR testing with the ETCDBACKUP_IMAGE variable.
		if os.Getenv("ETCDBACKUP_IMAGE") != "" {
			backupImage = os.Getenv("ETCDBACKUP_IMAGE")
		}

		By(fmt.Sprintf("Run an etcd backup using %s", backupImage))
		bk, err := cli.Client.Admin.BatchV1.Jobs("openshift-etcd").Create(&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "e2e-test-etcdbackup",
				Namespace: "openshift-etcd",
			},
			Spec: batchv1.JobSpec{
				Template: v1.PodTemplateSpec{
					Spec: v1.PodSpec{
						NodeSelector:       map[string]string{"node-role.kubernetes.io/master": "true"},
						ServiceAccountName: "etcd-backup",
						RestartPolicy:      "Never",
						Containers: []v1.Container{
							{
								Name:            "etcdbackup",
								Image:           backupImage,
								ImagePullPolicy: "Always",
								Args:            []string{fmt.Sprintf("-blobname=%s", backup), "save"},
								VolumeMounts: []v1.VolumeMount{
									{
										Name:      "azureconfig",
										MountPath: "/_data/_out",
										ReadOnly:  true,
									},
									{
										Name:      "origin-master",
										MountPath: "/etc/origin/master",
										ReadOnly:  true,
									},
								},
							},
						},
						Volumes: []v1.Volume{
							{
								Name: "azureconfig",
								VolumeSource: v1.VolumeSource{
									HostPath: &v1.HostPathVolumeSource{
										Path: "/etc/origin/cloudprovider",
									},
								},
							},
							{
								Name: "origin-master",
								VolumeSource: v1.VolumeSource{
									HostPath: &v1.HostPathVolumeSource{
										Path: "/etc/origin/master",
									},
								},
							},
						},
					},
				},
			},
		})
		Expect(err).NotTo(HaveOccurred())
		err = wait.Poll(2*time.Second, 5*time.Minute, ready.BatchIsReady(cli.Client.Admin.BatchV1.Jobs(bk.Namespace), bk.Name))
		Expect(err).NotTo(HaveOccurred())

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
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
		defer cancel()
		resp, err := azurecli.OpenShiftManagedClustersAdmin.RestoreAndWait(ctx, resourceGroup, resourceGroup, backup)
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
