package fakerp

import (
	"context"
	"crypto/sha1"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-02-01/storage"
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/ghodss/yaml"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	azureclientstorage "github.com/openshift/openshift-azure/pkg/util/azureclient/storage"
	"github.com/openshift/openshift-azure/pkg/util/randomstring"
	"github.com/openshift/openshift-azure/pkg/util/ready"

	"github.com/openshift/openshift-azure/pkg/cluster/updateblob"
	"github.com/openshift/openshift-azure/pkg/jsonpath"
	"github.com/openshift/openshift-azure/test/clients/azure"
	"github.com/openshift/openshift-azure/test/e2e/standard"
)

var _ = Describe("Openshift on Azure admin e2e tests [AzureClusterReader][Fake]", func() {
	var (
		cli       *standard.SanityChecker
		azCli     *azure.Client
		ctx       context.Context
		namespace string
		err       error
	)

	BeforeEach(func() {
		cli, err = standard.NewDefaultSanityChecker()
		Expect(err).NotTo(HaveOccurred())
		Expect(cli).ToNot(BeNil())
		azCli, err = azure.NewClientFromEnvironment(true)
		Expect(err).NotTo(HaveOccurred())
		ctx = context.Background()

		// Create a temp project for specs in this suite
		suffix, err := randomstring.RandomString("abcdefghijklmnopqrstuvwxyz0123456789", 5)
		Expect(err).NotTo(HaveOccurred())
		namespace = fmt.Sprintf("admin-test-%s", suffix)
		err = cli.Client.Admin.CreateProject(namespace)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		By("Cleaning up...")
		err = cli.Client.Admin.CleanupProject(namespace)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should run the correct image", func() {
		// e2e check should ensure that no reg-aws images are running on box
		pods, err := cli.Client.AzureClusterReader.CoreV1.Pods("").List(metav1.ListOptions{})
		Expect(err).ToNot(HaveOccurred())

		for _, pod := range pods.Items {
			for _, container := range pod.Spec.Containers {
				Expect(strings.HasPrefix(container.Image, "registry.reg-aws.openshift.com/")).ToNot(BeTrue())
			}
		}

		// fetch master-000000 and determine the OS type
		master0, _ := cli.Client.AzureClusterReader.CoreV1.Nodes().Get("master-000000", metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		// set registryPrefix to appropriate string based upon master's OS type
		var registryPrefix string
		if strings.HasPrefix(master0.Status.NodeInfo.OSImage, "Red Hat Enterprise") {
			registryPrefix = "registry.access.redhat.com/openshift3/ose-"
		} else {
			registryPrefix = "quay.io/openshift/origin-"
		}

		// Check all Configmaps for image format matches master's OS type
		// format: registry.access.redhat.com/openshift3/ose-${component}:${version}
		configmaps, err := cli.Client.AzureClusterReader.CoreV1.ConfigMaps("openshift-node").List(metav1.ListOptions{})
		Expect(err).ToNot(HaveOccurred())
		var nodeConfig map[string]interface{}
		for _, configmap := range configmaps.Items {
			err = yaml.Unmarshal([]byte(configmap.Data["node-config.yaml"]), &nodeConfig)
			format := jsonpath.MustCompile("$.imageConfig.format").MustGetString(nodeConfig)
			Expect(strings.HasPrefix(format, registryPrefix)).To(BeTrue())
		}
	})

	It("should ensure no unnecessary VM rotations occured", func() {
		Expect(os.Getenv("RESOURCEGROUP")).ToNot(BeEmpty())
		azurecli, err := azure.NewClientFromEnvironment(true)
		Expect(err).ToNot(HaveOccurred())

		ubs, err := updateblob.NewBlobService(azurecli.BlobStorage)
		Expect(err).ToNot(HaveOccurred())

		By("reading the update blob before running an update")
		before, err := ubs.Read()
		Expect(err).ToNot(HaveOccurred())

		By("ensuring the update blob has the right amount of entries")
		Expect(len(before.InstanceHashes)).To(BeEquivalentTo(3)) // one per master instance
		Expect(len(before.ScalesetHashes)).To(BeEquivalentTo(2)) // one per worker scaleset

		By("running an update")
		external, err := azurecli.OpenShiftManagedClusters.Get(context.Background(), os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"))
		Expect(err).NotTo(HaveOccurred())
		Expect(external).NotTo(BeNil())
		external.Properties.ProvisioningState = nil

		updated, err := azurecli.OpenShiftManagedClusters.CreateOrUpdateAndWait(context.Background(), os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"), external)
		Expect(err).NotTo(HaveOccurred())
		Expect(updated.StatusCode).To(Equal(http.StatusOK))
		Expect(updated).NotTo(BeNil())

		By("reading the update blob after running an update")
		after, err := ubs.Read()
		Expect(err).ToNot(HaveOccurred())

		By("comparing the update blob before and after an update")
		Expect(reflect.DeepEqual(before, after)).To(Equal(true))
	})

	It("should be able to configure azure file persistent volumes", func() {
		const (
			prefix = "azure-file"
		)

		var (
			pvName string
		)

		pvName = fmt.Sprintf("%s-pv", prefix)
		// need a way to lessen the chances of name collision in storage account names
		// and also they must lowercase letters and numbers
		clusterSHA := sha1.Sum([]byte(azCli.Config.ResourceGroup))
		// storage account names are capped at 24 chars
		accountName := fmt.Sprintf("azf%x", clusterSHA)[:24]

		// create storage accounts and get their keys
		By(fmt.Sprintf("Creating %s storage account", accountName))
		_, err = azCli.Accounts.GetOrCreate(
			ctx,
			azCli.Config.ResourceGroup,
			accountName,
			storage.AccountCreateParameters{
				Sku: &storage.Sku{
					Name: storage.StandardLRS},
				Kind:                              storage.Storage,
				Location:                          to.StringPtr(azCli.Config.Location),
				AccountPropertiesCreateParameters: &storage.AccountPropertiesCreateParameters{},
				Tags: map[string]*string{
					"type": to.StringPtr("test"),
				},
			})
		Expect(err).NotTo(HaveOccurred())
		By(fmt.Sprintf("Created %s storage account", accountName))
		By(fmt.Sprintf("Fetching %s account keys", accountName))
		accountKeys, err := azCli.Accounts.ListKeys(ctx, azCli.Config.ResourceGroup, accountName)
		Expect(err).NotTo(HaveOccurred())
		accountKey := *(*accountKeys.Keys)[0].Value
		By(fmt.Sprintf("Fetched %s account keys", accountName))

		// create a file share to hold the PV
		shareName := fmt.Sprintf("%s-share", prefix)
		By(fmt.Sprintf("Creating %s file share", shareName))
		storageCli, err := azureclientstorage.NewClient(accountName, accountKey, azureclientstorage.DefaultBaseURL, azureclientstorage.DefaultAPIVersion, true)
		Expect(err).NotTo(HaveOccurred())
		fss := storageCli.GetFileService()
		share := fss.GetShareReference(shareName)
		_, err = share.CreateIfNotExists(nil)
		Expect(err).NotTo(HaveOccurred())
		By(fmt.Sprintf("Created %s file share", shareName))

		// Create a secret to hold the account data
		secretName := fmt.Sprintf("%s-secret", prefix)
		By(fmt.Sprintf("Creating %s in %s to hold the account credentials", secretName, namespace))
		_, err = cli.Client.Admin.CoreV1.Secrets(namespace).Create(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: secretName,
			},
			StringData: map[string]string{
				"azurestorageaccountname": accountName,
				"azurestorageaccountkey":  accountKey,
			},
		})
		Expect(err).NotTo(HaveOccurred())
		By(fmt.Sprintf("Created %s", secretName))

		// Create a PersistentVolume
		storageClass := fmt.Sprintf("%s-sc", prefix)
		pvQuota, err := resource.ParseQuantity("10Gi")
		Expect(err).NotTo(HaveOccurred())
		By(fmt.Sprintf("Creating PV %s of size %v", pvName, pvQuota))
		_, err = cli.Client.Admin.CoreV1.PersistentVolumes().Create(&corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: pvName,
			},
			Spec: corev1.PersistentVolumeSpec{
				Capacity: corev1.ResourceList{
					corev1.ResourceStorage: pvQuota,
				},
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.PersistentVolumeAccessMode("ReadWriteMany"),
				},
				StorageClassName: storageClass,
				PersistentVolumeSource: corev1.PersistentVolumeSource{
					AzureFile: &corev1.AzureFilePersistentVolumeSource{
						SecretName:      secretName,
						SecretNamespace: to.StringPtr(namespace),
						ShareName:       shareName,
						ReadOnly:        false,
					},
				},
				MountOptions: []string{
					"dir_mode=0777",
					"file_mode=0777",
				},
			},
		})
		defer cli.Client.Admin.CoreV1.PersistentVolumes().Delete(pvName, nil)
		Expect(err).NotTo(HaveOccurred())
		By(fmt.Sprintf("Created PV %s", pvName))

		// Create a PersistentVolumeClaim
		pvcName := fmt.Sprintf("%s-pvc", prefix)
		pvcStorage, err := resource.ParseQuantity("2Gi")
		Expect(err).NotTo(HaveOccurred())
		By(fmt.Sprintf("Creating PVC %s in namespace %s", pvcName, namespace))
		_, err = cli.Client.Admin.CoreV1.PersistentVolumeClaims(namespace).Create(&corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: pvcName,
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.PersistentVolumeAccessMode("ReadWriteMany"),
				},
				StorageClassName: to.StringPtr(storageClass),
				VolumeName:       pvName,
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: pvcStorage,
					},
				},
			},
		})
		Expect(err).NotTo(HaveOccurred())
		By(fmt.Sprintf("Created PVC %s", pvcName))

		// Create a pod to run a simple program to test azure-file
		By("Creating a simple pod to run dd")
		podName := "busybox-1"
		_, err = cli.Client.Admin.CoreV1.Pods(namespace).Create(&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: podName,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  podName,
						Image: "busybox",
						Command: []string{
							"/bin/dd",
							"if=/dev/urandom",
							fmt.Sprintf("of=/data/%s.bin", namespace),
							"bs=1M",
							"count=100",
						},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      fmt.Sprintf("%s-vol", prefix),
								MountPath: "/data",
								ReadOnly:  false,
							},
						},
					},
				},
				Volumes: []corev1.Volume{
					{
						Name: fmt.Sprintf("%s-vol", prefix),
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: pvcName,
							},
						},
					},
				},
				RestartPolicy: corev1.RestartPolicyNever,
			},
		})
		Expect(err).NotTo(HaveOccurred())
		By("Created a simple pod to run dd")
		By("Waiting for pod to succeed")
		err = wait.PollImmediate(2*time.Second, 10*time.Minute, ready.PodReachesPhase(cli.Client.Admin.CoreV1.Pods(namespace), podName, corev1.PodSucceeded))
		Expect(err).NotTo(HaveOccurred())
		By("Pod succeeded")

		// TODO: cleanup azure resources? Storage accounts act up funny if you
		// remove and re-create in quick succession
	})
})
