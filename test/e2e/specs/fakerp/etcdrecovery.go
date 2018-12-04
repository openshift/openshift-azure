package fakerp

import (
	"context"
	"fmt"
	"os"
	"path"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/openshift/openshift-azure/pkg/api"
	realfakerp "github.com/openshift/openshift-azure/pkg/fakerp"
	"github.com/openshift/openshift-azure/pkg/plugin"
	"github.com/openshift/openshift-azure/pkg/util/log"
	"github.com/openshift/openshift-azure/pkg/util/managedcluster"
	"github.com/openshift/openshift-azure/pkg/util/randomstring"
	"github.com/openshift/openshift-azure/pkg/util/ready"
	"github.com/openshift/openshift-azure/test/clients/openshift"
)

var _ = Describe("Etcd Recovery E2E tests [EtcdRecovery][Fake][LongRunning]", func() {
	const (
		configMapName = "recovery-test-data"
	)
	var (
		cli       *openshift.Client
		admincli  *openshift.Client
		backup    string
		namespace string
	)

	BeforeEach(func() {
		var err error
		admincli, err = openshift.NewAdminClient()
		Expect(err).ToNot(HaveOccurred())
		cli, err = openshift.NewEndUserClient()
		Expect(err).ToNot(HaveOccurred())

		backup, err = randomstring.RandomString("abcdefghijklmnopqrstuvwxyz0123456789", 5)
		Expect(err).ToNot(HaveOccurred())
		backup = "e2e-test-" + backup
		namespace, err = randomstring.RandomString("abcdefghijklmnopqrstuvwxyz0123456789", 5)
		Expect(err).ToNot(HaveOccurred())
		namespace = "e2e-test-" + namespace
		fmt.Fprintln(GinkgoWriter, "Using namespace", namespace)
		err = cli.CreateProject(namespace)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		cli.CoreV1.Namespaces().Delete(namespace, nil)
		admincli.BatchV1.Jobs("openshift-etcd").Delete("e2e-test-etcdbackup", nil)
	})

	It("should be possible to recover etcd from a backup", func() {
		dataDir, err := realfakerp.FindDirectory(realfakerp.DataDirectory)
		Expect(err).NotTo(HaveOccurred())
		cs, err := managedcluster.ReadConfig(path.Join(dataDir, "containerservice.yaml"))
		Expect(cs).NotTo(BeNil())
		cs.Properties.ProvisioningState = ""

		By("Create a test configmap with value=first")
		cm1, err := cli.CoreV1.ConfigMaps(namespace).Create(&v1.ConfigMap{
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

		By("Run an etcd backup")
		bk, err := admincli.BatchV1.Jobs("openshift-etcd").Create(&batchv1.Job{
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
								Image:           cs.Config.Images.EtcdBackup,
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
		err = wait.Poll(2*time.Second, 5*time.Minute, ready.BatchIsReady(admincli.BatchV1.Jobs(bk.Namespace), bk.Name))
		Expect(err).NotTo(HaveOccurred())

		// wait for it to exist
		By("Overwrite the test configmap with value=second")
		cm2, err := cli.CoreV1.ConfigMaps(namespace).Update(&v1.ConfigMap{
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

		By("Run the Recovery")
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
		defer cancel()
		ctx = context.WithValue(ctx, api.ContextKeyClientID, os.Getenv("AZURE_CLIENT_ID"))
		ctx = context.WithValue(ctx, api.ContextKeyClientSecret, os.Getenv("AZURE_CLIENT_SECRET"))
		ctx = context.WithValue(ctx, api.ContextKeyTenantID, os.Getenv("AZURE_TENANT_ID"))
		logrus.SetLevel(log.SanitizeLogLevel("Debug"))
		logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
		logrus.SetOutput(GinkgoWriter)
		log := logrus.NewEntry(logrus.StandardLogger())

		err = recover(ctx, log, backup, cs)
		Expect(err).NotTo(HaveOccurred())

		By("confirm the state of the backup")
		final, err := cli.CoreV1.ConfigMaps(namespace).Get(configMapName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(final.Data).To(HaveKeyWithValue("value", "before-backup"))
	})
})

func recover(ctx context.Context, log *logrus.Entry, blobName string, cs *api.OpenShiftManagedCluster) error {
	config, err := realfakerp.GetPluginConfig()
	if err != nil {
		return err
	}

	p, errs := plugin.NewPlugin(log, config)
	if len(errs) > 0 {
		return kerrors.NewAggregate(errs)
	}
	deployer := realfakerp.GetDeployer(cs, log, config)
	if err := p.RecoverEtcdCluster(ctx, cs, deployer, blobName); err != nil {
		fmt.Fprintf(GinkgoWriter, "RecoverEtcdCluster error: %v", err)
		return err
	}

	return nil
}
