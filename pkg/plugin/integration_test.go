package plugin

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/ghodss/yaml"
	securityapi "github.com/openshift/api/security/v1"
	fakesec "github.com/openshift/client-go/security/clientset/versioned/fake"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1b1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes/fake"
	restclient "k8s.io/client-go/rest"
	k8stesting "k8s.io/client-go/testing"

	"github.com/openshift/openshift-azure/pkg/api"
	pluginapi "github.com/openshift/openshift-azure/pkg/api/plugin"
	"github.com/openshift/openshift-azure/pkg/arm"
	"github.com/openshift/openshift-azure/pkg/cluster"
	"github.com/openshift/openshift-azure/pkg/cluster/kubeclient"
	"github.com/openshift/openshift-azure/pkg/cluster/scaler"
	"github.com/openshift/openshift-azure/pkg/cluster/updateblob"
	"github.com/openshift/openshift-azure/pkg/config"
	"github.com/openshift/openshift-azure/pkg/startup"
	"github.com/openshift/openshift-azure/pkg/sync"
	fakecloud "github.com/openshift/openshift-azure/pkg/util/azureclient/fake"
	"github.com/openshift/openshift-azure/pkg/util/random"
	"github.com/openshift/openshift-azure/pkg/util/resourceid"
	"github.com/openshift/openshift-azure/pkg/util/tls"
	"github.com/openshift/openshift-azure/pkg/util/wait"
	"github.com/openshift/openshift-azure/test/util/populate"
	testtls "github.com/openshift/openshift-azure/test/util/tls"
)

const (
	vaultKeyNamePublicHostname = "PublicHostname"
	vaultKeyNameRouter         = "Router"
)

func getFakeDeployer(log *logrus.Entry, cs *api.OpenShiftManagedCluster, az *fakecloud.AzureCloud) api.DeployFn {
	return func(ctx context.Context, azuretemplate map[string]interface{}) (*string, error) {
		log.Info("applying arm template deployment")

		err := az.DeploymentsClient.CreateOrUpdateAndWait(ctx, cs.Properties.AzProfile.ResourceGroup, "azuredeploy", resources.Deployment{
			Properties: &resources.DeploymentProperties{
				Template: azuretemplate,
				Mode:     resources.Incremental,
			},
		})
		if err != nil {
			log.Warnf("deployment failed: %#v", err)
			return nil, err
		}

		return nil, nil
	}
}

func enrichCs(cs *api.OpenShiftManagedCluster) error {
	rg := "testRG"
	dnsDomain := "cloudapp.azure.com"
	tenantID := uuid.NewV4().String()
	clientID := uuid.NewV4().String()
	secret := "suspicious"
	cs.Properties.AzProfile = api.AzProfile{
		TenantID:       tenantID,
		SubscriptionID: uuid.NewV4().String(),
		ResourceGroup:  rg,
	}

	cs.Properties.AuthProfile.IdentityProviders = make([]api.IdentityProvider, 1)
	cs.Properties.AuthProfile.IdentityProviders[0].Name = "Azure AD"
	cs.Properties.AuthProfile.IdentityProviders[0].Provider = &api.AADIdentityProvider{
		Kind:     "AADIdentityProvider",
		ClientID: clientID,
		Secret:   secret,
		TenantID: tenantID,
	}

	cs.Properties.MasterServicePrincipalProfile = api.ServicePrincipalProfile{
		ClientID: uuid.NewV4().String(),
		Secret:   uuid.NewV4().String(),
	}
	cs.Properties.WorkerServicePrincipalProfile = api.ServicePrincipalProfile{
		ClientID: uuid.NewV4().String(),
		Secret:   uuid.NewV4().String(),
	}

	// /subscriptions/{subscription}/resourcegroups/{resource_group}/providers/Microsoft.ContainerService/openshiftmanagedClusters/{cluster_name}
	cs.ID = resourceid.ResourceID(cs.Properties.AzProfile.SubscriptionID, rg, "Microsoft.ContainerService/openshiftmanagedClusters", cs.Name)

	if len(cs.Properties.RouterProfiles) == 0 {
		cs.Properties.RouterProfiles = []api.RouterProfile{
			{
				Name: "default",
			},
		}
	}

	var vaultURL string
	var err error
	vaultURL, err = random.VaultURL("kv-")
	if err != nil {
		return err
	}

	cs.Properties.APICertProfile.KeyVaultSecretURL = vaultURL + "/secrets/" + vaultKeyNamePublicHostname
	cs.Properties.RouterProfiles[0].RouterCertProfile.KeyVaultSecretURL = vaultURL + "/secrets/" + vaultKeyNameRouter

	cs.Properties.PublicHostname = "openshift." + rg + "." + dnsDomain
	cs.Properties.RouterProfiles[0].PublicSubdomain = "apps." + rg + "." + dnsDomain

	if cs.Properties.FQDN == "" {
		cs.Properties.FQDN, err = random.FQDN(cs.Location+"."+dnsDomain, 20)
		if err != nil {
			return err
		}
	}

	if cs.Properties.RouterProfiles[0].FQDN == "" {
		cs.Properties.RouterProfiles[0].FQDN, err = random.FQDN(cs.Location+"."+dnsDomain, 20)
		if err != nil {
			return err
		}
	}

	return nil
}

func (w fakeResponseWrapper) DoRaw() ([]byte, error) {
	return w, nil
}

func (w fakeResponseWrapper) Stream() (io.ReadCloser, error) {
	return nil, nil
}

func newFakeResponseWrapper(raw []byte) restclient.ResponseWrapper {
	var fr fakeResponseWrapper = raw
	return fr
}

type fakeResponseWrapper []byte

func getFakeHTTPClient(cs *api.OpenShiftManagedCluster) wait.SimpleHTTPClient {
	return wait.NewFakeHTTPClient()
}

func newFakeUpgrader(ctx context.Context, log *logrus.Entry, cs *api.OpenShiftManagedCluster, testConfig api.TestConfig, kubeclient kubeclient.Interface, azs *fakecloud.AzureCloud) (cluster.Upgrader, error) {
	arm, err := arm.New(ctx, log, cs, testConfig)
	if err != nil {
		return nil, err
	}

	u := &cluster.Upgrade{
		Interface: kubeclient,

		TestConfig:     testConfig,
		AccountsClient: azs.AccountsClient,
		StorageClient:  azs.StorageClient,
		Vmc:            azs.VirtualMachineScaleSetVMsClient,
		Ssc:            azs.VirtualMachineScaleSetsClient,
		Kvc:            azs.KeyVaultClient,
		Vnc:            azs.VirtualNetworksClient,
		Log:            log,
		ScalerFactory:  scaler.NewFactory(),
		Hasher: &cluster.Hash{
			Log:            log,
			TestConfig:     testConfig,
			StartupFactory: startup.New,
			Arm:            arm,
		},
		Arm:                arm,
		GetConsoleClient:   getFakeHTTPClient,
		GetAPIServerClient: getFakeHTTPClient,
		Cs:                 cs,
	}

	u.Cs.Config.ConfigStorageAccountKey = "config"
	u.Cs.Config.ConfigStorageAccountKey = uuid.NewV4().String()
	bsc := u.StorageClient.GetBlobService()
	u.UpdateBlobService = updateblob.NewBlobService(bsc)

	return u, nil
}

func newFakeKubeclient(log *logrus.Entry, cli *fake.Clientset, seccli *fakesec.Clientset) kubeclient.Interface {
	return &kubeclient.Kubeclientset{
		Log:    log,
		Client: cli,
		Seccli: seccli,
	}
}

func setupNewCluster(ctx context.Context, log *logrus.Entry, cs *api.OpenShiftManagedCluster, az *fakecloud.AzureCloud) (*plugin, *fake.Clientset, error) {
	data, err := ioutil.ReadFile("../../pluginconfig/pluginconfig-311.yaml")
	if err != nil {
		return nil, nil, err
	}
	var template *pluginapi.Config
	if err := yaml.Unmarshal(data, &template); err != nil {
		return nil, nil, err
	}

	template.Certificates.GenevaLogging.Cert = testtls.DummyCertificate
	template.Certificates.GenevaLogging.Key = testtls.DummyPrivateKey
	template.Certificates.GenevaMetrics.Cert = testtls.DummyCertificate
	template.Certificates.GenevaMetrics.Key = testtls.DummyPrivateKey
	template.Certificates.PackageRepository.Cert = testtls.DummyCertificate
	template.Certificates.PackageRepository.Key = testtls.DummyPrivateKey
	template.ImagePullSecret = populate.DummyImagePullSecret("registry.redhat.io")
	template.GenevaImagePullSecret = populate.DummyImagePullSecret("osarpint.azurecr.io")

	cli := fake.NewSimpleClientset()
	cli.PrependReactor("get", "deployments", func(action k8stesting.Action) (bool, runtime.Object, error) {
		get := action.(k8stesting.GetAction)
		d := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      get.GetName(),
				Namespace: get.GetNamespace(),
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				UpdatedReplicas:   1,
			},
		}
		return true, d, nil
	})
	cli.PrependReactor("get", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		get := action.(k8stesting.GetAction)
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      get.GetName(),
				Namespace: get.GetNamespace(),
			},
			Status: corev1.PodStatus{
				Conditions: []corev1.PodCondition{
					{
						Type:   corev1.PodReady,
						Status: corev1.ConditionTrue,
					},
				},
			},
		}
		return true, pod, nil
	})
	cli.PrependReactor("get", "nodes", func(action k8stesting.Action) (bool, runtime.Object, error) {
		get := action.(k8stesting.GetAction)
		node := &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: get.GetName(),
			},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{
						Type:   corev1.NodeReady,
						Status: corev1.ConditionTrue,
					},
				},
			},
		}

		return true, node, nil
	})
	cli.PrependReactor("get", "cronjobs", func(action k8stesting.Action) (bool, runtime.Object, error) {
		get := action.(k8stesting.GetAction)
		cronjob := &batchv1b1.CronJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:      get.GetName(),
				Namespace: get.GetNamespace(),
			},
			Spec: batchv1b1.CronJobSpec{
				JobTemplate: batchv1b1.JobTemplateSpec{
					Spec: batchv1.JobSpec{
						BackoffLimit: to.Int32Ptr(4),
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{{}},
							},
						},
					},
				},
			},
		}
		return true, cronjob, nil
	})
	cli.PrependReactor("get", "jobs", func(action k8stesting.Action) (bool, runtime.Object, error) {
		get := action.(k8stesting.GetAction)
		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      get.GetName(),
				Namespace: get.GetNamespace(),
			},
			Status: batchv1.JobStatus{
				Conditions: []batchv1.JobCondition{
					{
						Type:   batchv1.JobComplete,
						Status: corev1.ConditionTrue,
					},
				},
			},
		}
		return true, job, nil
	})

	cli.AddProxyReactor("services", func(action k8stesting.Action) (handled bool, ret restclient.ResponseWrapper, err error) {
		return true, newFakeResponseWrapper(nil), nil
	})

	priv := securityapi.SecurityContextConstraints{
		TypeMeta: metav1.TypeMeta{
			Kind: "securitycontextconstraints",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "privileged",
		},
		AllowPrivilegedContainer: true,
		Users:                    []string{"system:admin"},
	}
	seccli := fakesec.NewSimpleClientset()
	// securitycontextconstraints gets mistakenly converted to securitycontextconstraints"es" by the generic
	// plural converter and it's not in the list of exceptions as it's an openshift type. So
	// we are using a reactor to return value.
	seccli.PrependReactor("get", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		get := action.(k8stesting.GetAction)
		if get.GetResource().Resource == "securitycontextconstraints" {
			return true, &priv, nil
		}
		return false, nil, fmt.Errorf("does not exist")
	})
	seccli.PrependReactor("update", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		get := action.(k8stesting.UpdateAction)
		if get.GetResource().Resource == "securitycontextconstraints" {
			return true, &priv, nil
		}
		return false, nil, fmt.Errorf("does not exist")
	})
	kc := newFakeKubeclient(log, cli, seccli)
	p := &plugin{
		pluginConfig: template,
		testConfig:   api.TestConfig{RunningUnderTest: true, DebugHashFunctions: os.Getenv("DEBUG_HASH_FUNCTIONS") == "true"},
		upgraderFactory: func(ctx context.Context, log *logrus.Entry, cs *api.OpenShiftManagedCluster, initializeStorageClients, disableKeepAlives bool, testConfig api.TestConfig) (cluster.Upgrader, error) {
			return newFakeUpgrader(ctx, log, cs, testConfig, kc, az)
		},
		configInterfaceFactory: config.New,
		log:                    log,
		now:                    time.Now,
	}
	err = enrichCs(cs)
	if err != nil {
		return nil, nil, err
	}

	err = p.GenerateConfig(ctx, cs, false)
	if err != nil {
		return nil, nil, err
	}

	if err := p.CreateOrUpdate(ctx, cs, false, getFakeDeployer(log, cs, az)); err != nil {
		return nil, nil, err
	}
	return p, cli, nil
}

func newTestCs() *api.OpenShiftManagedCluster {
	return &api.OpenShiftManagedCluster{
		Name:     "integrationTest",
		Location: "eastus",
		Config: api.Config{
			ConfigStorageAccount:      "config",
			RegistryStorageAccount:    "registry",
			RegistryStorageAccountKey: "foo",
			ServiceAccountKey:         testtls.DummyPrivateKey,
			SSHKey:                    testtls.DummyPrivateKey,
			Certificates: api.CertificateConfig{
				Ca:                     api.CertKeyPair{Cert: testtls.DummyCertificate, Key: testtls.DummyPrivateKey},
				NodeBootstrap:          api.CertKeyPair{Cert: testtls.DummyCertificate, Key: testtls.DummyPrivateKey},
				EtcdCa:                 api.CertKeyPair{Cert: testtls.DummyCertificate, Key: testtls.DummyPrivateKey},
				FrontProxyCa:           api.CertKeyPair{Cert: testtls.DummyCertificate, Key: testtls.DummyPrivateKey},
				ServiceSigningCa:       api.CertKeyPair{Cert: testtls.DummyCertificate, Key: testtls.DummyPrivateKey},
				ServiceCatalogCa:       api.CertKeyPair{Cert: testtls.DummyCertificate, Key: testtls.DummyPrivateKey},
				EtcdServer:             api.CertKeyPair{Cert: testtls.DummyCertificate, Key: testtls.DummyPrivateKey},
				EtcdPeer:               api.CertKeyPair{Cert: testtls.DummyCertificate, Key: testtls.DummyPrivateKey},
				EtcdClient:             api.CertKeyPair{Cert: testtls.DummyCertificate, Key: testtls.DummyPrivateKey},
				MasterServer:           api.CertKeyPair{Cert: testtls.DummyCertificate, Key: testtls.DummyPrivateKey},
				Admin:                  api.CertKeyPair{Cert: testtls.DummyCertificate, Key: testtls.DummyPrivateKey},
				AggregatorFrontProxy:   api.CertKeyPair{Cert: testtls.DummyCertificate, Key: testtls.DummyPrivateKey},
				MasterKubeletClient:    api.CertKeyPair{Cert: testtls.DummyCertificate, Key: testtls.DummyPrivateKey},
				MasterProxyClient:      api.CertKeyPair{Cert: testtls.DummyCertificate, Key: testtls.DummyPrivateKey},
				OpenShiftMaster:        api.CertKeyPair{Cert: testtls.DummyCertificate, Key: testtls.DummyPrivateKey},
				SDN:                    api.CertKeyPair{Cert: testtls.DummyCertificate, Key: testtls.DummyPrivateKey},
				Registry:               api.CertKeyPair{Cert: testtls.DummyCertificate, Key: testtls.DummyPrivateKey},
				RegistryConsole:        api.CertKeyPair{Cert: testtls.DummyCertificate, Key: testtls.DummyPrivateKey},
				ServiceCatalogServer:   api.CertKeyPair{Cert: testtls.DummyCertificate, Key: testtls.DummyPrivateKey},
				AroAdmissionController: api.CertKeyPair{Cert: testtls.DummyCertificate, Key: testtls.DummyPrivateKey},
				BlackBoxMonitor:        api.CertKeyPair{Cert: testtls.DummyCertificate, Key: testtls.DummyPrivateKey},
				GenevaLogging:          api.CertKeyPair{Cert: testtls.DummyCertificate, Key: testtls.DummyPrivateKey},
				GenevaMetrics:          api.CertKeyPair{Cert: testtls.DummyCertificate, Key: testtls.DummyPrivateKey},
				PackageRepository:      api.CertKeyPair{Cert: testtls.DummyCertificate, Key: testtls.DummyPrivateKey},
			},
		},
		Properties: api.Properties{
			OpenShiftVersion: "v3.11",
			AgentPoolProfiles: []api.AgentPoolProfile{
				{Role: api.AgentPoolProfileRoleMaster, Count: 3, Name: "master", VMSize: "Standard_D2s_v3", SubnetCIDR: "10.0.0.0/24", OSType: "Linux"},
				{Role: api.AgentPoolProfileRoleCompute, Count: 1, Name: "compute", VMSize: "Standard_D2s_v3", SubnetCIDR: "10.0.0.0/24", OSType: "Linux"},
				{Role: api.AgentPoolProfileRoleInfra, Count: 3, Name: "infra", VMSize: "Standard_D2s_v3", SubnetCIDR: "10.0.0.0/24", OSType: "Linux"},
			},
			NetworkProfile: api.NetworkProfile{
				VnetCIDR:             "10.0.0.0/8",
				ManagementSubnetCIDR: to.StringPtr("10.0.1.0/24"),
			},
		},
	}
}

func newFakeAzureCloud(log *logrus.Entry) *fakecloud.AzureCloud {
	bKey, _ := tls.PrivateKeyAsBytes(testtls.DummyPrivateKey)
	bCert, _ := tls.CertAsBytes(testtls.DummyCertificate)
	secret := "PRIVATE KEY\n" + string(bKey) + "CERTIFICATE\n" + string(bCert)
	secrets := []keyvault.SecretBundle{
		{
			ID:    to.StringPtr("PublicHostname"),
			Value: to.StringPtr(secret),
		},
		{
			ID:    to.StringPtr("Router"),
			Value: to.StringPtr(secret),
		},
	}

	return fakecloud.NewFakeAzureCloud(log, secrets)
}

func getHashes(az *fakecloud.AzureCloud, cs *api.OpenShiftManagedCluster) (*updateblob.UpdateBlob, string, error) {
	bsc := az.StorageClient.GetBlobService()
	updateBlobService := updateblob.NewBlobService(bsc)
	blob, err := updateBlobService.Read()
	if err != nil {
		return nil, "", err
	}
	// get sync deployment checksum annotation
	// FIXME: only doing it this way as fake kube Get on the deployment
	// always returns an empty string on the annotation
	syncer, err := sync.New(az.ComputeRP.Log, cs, false)
	if err != nil {
		return nil, "", err
	}
	syncChecksum, err := syncer.Hash()
	if err != nil {
		return nil, "", err
	}
	return blob, hex.EncodeToString(syncChecksum), nil
}

type rotationType string

const (
	rotationCompute rotationType = "compute"
	rotationInfra   rotationType = "infra"
	rotationMaster  rotationType = "master"
	rotationSync    rotationType = "sync"
)

func getRotations(beforeBlob, afterBlob *updateblob.UpdateBlob, beforeSyncChecksum, afterSyncChecksum string) map[rotationType]bool {
	rotations := map[rotationType]bool{rotationCompute: false, rotationInfra: false, rotationMaster: false, rotationSync: false}
	for host := range beforeBlob.HostnameHashes {
		rotated := !reflect.DeepEqual(beforeBlob.HostnameHashes[host], afterBlob.HostnameHashes[host])
		if rotated {
			rotations[rotationMaster] = true
		}
	}
	for scaleset := range beforeBlob.ScalesetHashes {
		rotated := !reflect.DeepEqual(beforeBlob.ScalesetHashes[scaleset], afterBlob.ScalesetHashes[scaleset])
		if strings.Contains(scaleset, "compute") && rotated {
			rotations[rotationCompute] = true
		} else if strings.Contains(scaleset, "infra") && rotated {
			rotations[rotationInfra] = true
		}
	}
	if beforeSyncChecksum != afterSyncChecksum {
		rotations[rotationSync] = true
	}
	return rotations
}

func getNodeCountFromAz(az *fakecloud.AzureCloud) map[rotationType]int {
	nodeCount := map[rotationType]int{rotationCompute: 0, rotationInfra: 0, rotationMaster: 0}
	for scaleset, vms := range az.ComputeRP.Vms {
		for _, role := range []rotationType{rotationCompute, rotationInfra, rotationMaster} {
			if strings.Contains(scaleset, string(role)) {
				nodeCount[role] += len(vms)
				break
			}
		}
	}

	return nodeCount
}

func TestHowAdminConfigChangesCausesRotations(t *testing.T) {
	tests := []struct {
		name           string
		change         func(cs *api.OpenShiftManagedCluster)
		expectRotation map[rotationType]bool
	}{
		{
			name:           "no changes",
			expectRotation: map[rotationType]bool{rotationMaster: false, rotationInfra: false, rotationSync: false, rotationCompute: false},
			change:         func(cs *api.OpenShiftManagedCluster) {},
		},
		{
			name:           "change vm image",
			expectRotation: map[rotationType]bool{rotationMaster: true, rotationInfra: true, rotationSync: true, rotationCompute: true},
			change:         func(cs *api.OpenShiftManagedCluster) { cs.Config.ImageVersion = "311.12.12345678" },
		},
		{
			name:           "change controller loglevel",
			expectRotation: map[rotationType]bool{rotationMaster: true, rotationInfra: false, rotationSync: false, rotationCompute: false},
			change:         func(cs *api.OpenShiftManagedCluster) { cs.Config.ComponentLogLevel.ControllerManager = to.IntPtr(5) },
		},
		{
			name:           "change container image",
			expectRotation: map[rotationType]bool{rotationMaster: false, rotationInfra: false, rotationSync: true, rotationCompute: false},
			change:         func(cs *api.OpenShiftManagedCluster) { cs.Config.Images.WebConsole = "newImage" },
		},
		{
			name:           "change security patch packages",
			expectRotation: map[rotationType]bool{rotationMaster: true, rotationInfra: true, rotationSync: false, rotationCompute: true},
			change:         func(cs *api.OpenShiftManagedCluster) { cs.Config.SecurityPatchPackages = []string{"patch-rpm"} },
		},
		{
			name:           "change SSHSourceAddressPrefixes",
			expectRotation: map[rotationType]bool{rotationMaster: false, rotationInfra: false, rotationSync: false, rotationCompute: false},
			change: func(cs *api.OpenShiftManagedCluster) {
				cs.Config.SSHSourceAddressPrefixes = []string{"101.165.48.112/24"}
			},
		},
	}

	log := logrus.NewEntry(logrus.StandardLogger())
	ctx := context.Background()
	cs := newTestCs()
	az := newFakeAzureCloud(log)
	p, _, err := setupNewCluster(ctx, log, cs, az)
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log.Infof("--- Test: %s", tt.name)
			beforeBlob, beforeSyncChecksum, err := getHashes(az, cs)
			if err != nil {
				t.Fatal(err)
			}
			oldCs := cs.DeepCopy()
			tt.change(cs)

			errs := p.ValidateAdmin(ctx, cs, oldCs)
			if errs != nil {
				t.Fatal(errs)
			}
			perr := p.CreateOrUpdate(ctx, cs, true, getFakeDeployer(log, cs, az))
			if perr != nil {
				t.Fatal(perr)
			}

			afterBlob, afterSyncChecksum, err := getHashes(az, cs)
			if err != nil {
				t.Fatal(err)
			}
			rotations := getRotations(beforeBlob, afterBlob, beforeSyncChecksum, afterSyncChecksum)
			if !reflect.DeepEqual(tt.expectRotation, rotations) {
				t.Fatalf("rotation mismatch: expected %v, got %v", tt.expectRotation, rotations)
			}
		})
	}
}

func TestHowUserConfigChangesCausesRotations(t *testing.T) {
	tests := []struct {
		name           string
		change         func(cs *api.OpenShiftManagedCluster)
		expectRotation map[rotationType]bool
	}{
		{
			name:           "no changes",
			expectRotation: map[rotationType]bool{rotationMaster: false, rotationInfra: false, rotationSync: false, rotationCompute: false},
			change:         func(cs *api.OpenShiftManagedCluster) {},
		},
		{
			// Note: master and infra must be changed together.
			name:           "change master and infra vm size",
			expectRotation: map[rotationType]bool{rotationMaster: true, rotationInfra: true, rotationSync: false, rotationCompute: false},
			change: func(cs *api.OpenShiftManagedCluster) {
				for i := range cs.Properties.AgentPoolProfiles {
					if cs.Properties.AgentPoolProfiles[i].Role != api.AgentPoolProfileRoleCompute {
						cs.Properties.AgentPoolProfiles[i].VMSize = "Standard_D16s_v3"
					}
				}
			},
		},
		{
			// Note: master is rotating here, is this expected?
			name:           "change compute vm size",
			expectRotation: map[rotationType]bool{rotationMaster: true, rotationInfra: false, rotationSync: false, rotationCompute: true},
			change: func(cs *api.OpenShiftManagedCluster) {
				for i := range cs.Properties.AgentPoolProfiles {
					if cs.Properties.AgentPoolProfiles[i].Role == api.AgentPoolProfileRoleCompute {
						cs.Properties.AgentPoolProfiles[i].VMSize = "Standard_F16s_v2"
					}
				}
			},
		},
		{
			name:           "change AADIdentityProvider",
			expectRotation: map[rotationType]bool{rotationMaster: true, rotationInfra: false, rotationSync: true, rotationCompute: false},
			change: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.AuthProfile.IdentityProviders[0].Provider.(*api.AADIdentityProvider).Secret = "new"
			},
		},
		{
			name:           "change MonitorProfile",
			expectRotation: map[rotationType]bool{rotationMaster: false, rotationInfra: false, rotationSync: true, rotationCompute: false},
			change: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.MonitorProfile.Enabled = true
				oc.Properties.MonitorProfile.WorkspaceResourceID = "/subscriptions/foo/resourceGroups/bar/providers/Microsoft.OperationalInsights/workspaces/baz"
				oc.Properties.MonitorProfile.WorkspaceID = "00000000-0000-0000-0000-000000000000"
				oc.Properties.MonitorProfile.WorkspaceKey = "a2V5Cg=="
			},
		},
	}

	log := logrus.NewEntry(logrus.StandardLogger())
	ctx := context.Background()
	cs := newTestCs()
	az := newFakeAzureCloud(log)
	p, _, err := setupNewCluster(ctx, log, cs, az)
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			beforeBlob, beforeSyncChecksum, err := getHashes(az, cs)
			if err != nil {
				t.Fatal(err)
			}
			oldCs := cs.DeepCopy()
			tt.change(cs)

			errs := p.Validate(ctx, cs, oldCs, false)
			if errs != nil {
				t.Fatal(errs)
			}
			perr := p.CreateOrUpdate(ctx, cs, true, getFakeDeployer(log, cs, az))
			if perr != nil {
				t.Fatal(perr)
			}

			afterBlob, afterSyncChecksum, err := getHashes(az, cs)
			if err != nil {
				t.Fatal(err)
			}
			rotations := getRotations(beforeBlob, afterBlob, beforeSyncChecksum, afterSyncChecksum)
			if !reflect.DeepEqual(tt.expectRotation, rotations) {
				t.Fatalf("rotation mismatch: expected %v, got %v", tt.expectRotation, rotations)
			}
		})
	}
}

func TestHowActionsCauseRotations(t *testing.T) {
	log := logrus.NewEntry(logrus.StandardLogger())
	ctx := context.Background()
	tests := []struct {
		name           string
		change         func(p *plugin, cs *api.OpenShiftManagedCluster, az *fakecloud.AzureCloud) error
		expectRotation map[rotationType]bool
		expectNodes    map[rotationType]int
		expectCalls    []string
	}{
		{
			name:           "scale up",
			expectRotation: map[rotationType]bool{rotationMaster: false, rotationInfra: false, rotationSync: false, rotationCompute: false},
			expectNodes:    map[rotationType]int{rotationCompute: 6, rotationInfra: 3, rotationMaster: 3},
			change: func(p *plugin, cs *api.OpenShiftManagedCluster, az *fakecloud.AzureCloud) error {
				oldCs := cs.DeepCopy()
				for i, p := range cs.Properties.AgentPoolProfiles {
					if p.Role == api.AgentPoolProfileRoleCompute {
						cs.Properties.AgentPoolProfiles[i].Count = 6
					}
				}
				errs := p.Validate(ctx, cs, oldCs, false)
				if errs != nil {
					return kerrors.NewAggregate(errs)
				}
				perr := p.CreateOrUpdate(ctx, cs, true, getFakeDeployer(log, cs, az))
				if perr != nil {
					return perr
				}
				return nil
			},
		},
		{
			name:           "rotate cluster secrets",
			expectRotation: map[rotationType]bool{rotationMaster: true, rotationInfra: true, rotationSync: true, rotationCompute: true},
			expectNodes:    map[rotationType]int{rotationCompute: 1, rotationInfra: 3, rotationMaster: 3},
			change: func(p *plugin, cs *api.OpenShiftManagedCluster, az *fakecloud.AzureCloud) error {
				perr := p.RotateClusterSecrets(ctx, cs, getFakeDeployer(log, cs, az))
				if perr != nil {
					return perr
				}
				return nil
			},
		},
		{
			name:           "GetPluginVersion() cause no rotations",
			expectRotation: map[rotationType]bool{rotationMaster: false, rotationInfra: false, rotationSync: false, rotationCompute: false},
			expectNodes:    map[rotationType]int{rotationCompute: 1, rotationInfra: 3, rotationMaster: 3},
			change: func(p *plugin, cs *api.OpenShiftManagedCluster, az *fakecloud.AzureCloud) error {
				p.GetPluginVersion(ctx)
				return nil
			},
		},
		{
			name:           "ListClusterVMs() cause no rotations",
			expectRotation: map[rotationType]bool{rotationMaster: false, rotationInfra: false, rotationSync: false, rotationCompute: false},
			expectNodes:    map[rotationType]int{rotationCompute: 1, rotationInfra: 3, rotationMaster: 3},
			change: func(p *plugin, cs *api.OpenShiftManagedCluster, az *fakecloud.AzureCloud) error {
				_, perr := p.ListClusterVMs(ctx, cs)
				if perr != nil {
					return perr
				}
				return nil
			},
		},
		{
			name:           "ListEtcdBackups() cause no rotations",
			expectRotation: map[rotationType]bool{rotationMaster: false, rotationInfra: false, rotationSync: false, rotationCompute: false},
			expectNodes:    map[rotationType]int{rotationCompute: 1, rotationInfra: 3, rotationMaster: 3},
			change: func(p *plugin, cs *api.OpenShiftManagedCluster, az *fakecloud.AzureCloud) error {
				_, perr := p.ListEtcdBackups(ctx, cs)
				if perr != nil {
					return perr
				}
				return nil
			},
		},
		{
			name:           "GetControlPlanePods() cause no rotations",
			expectRotation: map[rotationType]bool{rotationMaster: false, rotationInfra: false, rotationSync: false, rotationCompute: false},
			expectNodes:    map[rotationType]int{rotationCompute: 1, rotationInfra: 3, rotationMaster: 3},
			change: func(p *plugin, cs *api.OpenShiftManagedCluster, az *fakecloud.AzureCloud) error {
				_, perr := p.GetControlPlanePods(ctx, cs)
				if perr != nil {
					return perr
				}
				return nil
			},
		},
		{
			name:           "runcommand - no rotations and correct call",
			expectRotation: map[rotationType]bool{rotationMaster: false, rotationInfra: false, rotationSync: false, rotationCompute: false},
			expectNodes:    map[rotationType]int{rotationCompute: 1, rotationInfra: 3, rotationMaster: 3},
			expectCalls:    []string{"VirtualMachineScaleSetVMsClient:RunCommand:ss-master:1"},
			change: func(p *plugin, cs *api.OpenShiftManagedCluster, az *fakecloud.AzureCloud) error {
				perr := p.RunCommand(ctx, cs, "master-000001", "RestartDocker")
				if perr != nil {
					return perr
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// for this test we always start with a new cluster (unlike the config change test above)
			cs := newTestCs()
			az := newFakeAzureCloud(log)
			p, _, err := setupNewCluster(ctx, log, cs, az)

			beforeBlob, beforeSyncChecksum, err := getHashes(az, cs)
			if err != nil {
				t.Fatal(err)
			}

			// clear the calls
			az.ComputeRP.Calls = []string{}

			err = tt.change(p, cs, az)
			if err != nil {
				t.Fatal(err)
			}

			for _, ec := range tt.expectCalls {
				found := false
				for _, call := range az.ComputeRP.Calls {
					if call == ec {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("call %s not found in %v", ec, az.ComputeRP.Calls)
				}
			}
			nodeCount := getNodeCountFromAz(az)
			if !reflect.DeepEqual(tt.expectNodes, nodeCount) {
				t.Fatalf("node mismatch: expected %v, got %v", tt.expectNodes, nodeCount)
			}

			afterBlob, afterSyncChecksum, err := getHashes(az, cs)
			if err != nil {
				t.Fatal(err)
			}
			rotations := getRotations(beforeBlob, afterBlob, beforeSyncChecksum, afterSyncChecksum)
			if !reflect.DeepEqual(tt.expectRotation, rotations) {
				t.Fatalf("rotation mismatch: expected %v, got %v", tt.expectRotation, rotations)
			}
		})
	}
}
