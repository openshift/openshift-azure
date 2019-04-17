package plugin

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-02-01/storage"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/davecgh/go-spew/spew"
	"github.com/ghodss/yaml"
	securityapi "github.com/openshift/api/security/v1"
	fakesec "github.com/openshift/client-go/security/clientset/versioned/fake"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	restclient "k8s.io/client-go/rest"
	k8stesting "k8s.io/client-go/testing"

	"github.com/openshift/openshift-azure/pkg/api"
	pluginapi "github.com/openshift/openshift-azure/pkg/api/plugin"
	"github.com/openshift/openshift-azure/pkg/cluster"
	"github.com/openshift/openshift-azure/pkg/cluster/kubeclient"
	"github.com/openshift/openshift-azure/pkg/cluster/updateblob"
	"github.com/openshift/openshift-azure/pkg/config"
	fakecloud "github.com/openshift/openshift-azure/pkg/util/azureclient/fake"
	"github.com/openshift/openshift-azure/pkg/util/random"
	"github.com/openshift/openshift-azure/pkg/util/resourceid"
	"github.com/openshift/openshift-azure/pkg/util/tls"
	testtls "github.com/openshift/openshift-azure/test/util/tls"
)

const (
	vaultKeyNamePublicHostname = "PublicHostname"
	vaultKeyNameRouter         = "Router"
)

func getFakeDeployer(log *logrus.Entry, cs *api.OpenShiftManagedCluster, az *fakecloud.AzureCloud) api.DeployFn {
	return func(ctx context.Context, azuretemplate map[string]interface{}) error {
		log.Info("applying arm template deployment")

		future, err := az.DeploymentsClient.CreateOrUpdate(ctx, cs.Properties.AzProfile.ResourceGroup, "azuredeploy", resources.Deployment{
			Properties: &resources.DeploymentProperties{
				Template: azuretemplate,
				Mode:     resources.Incremental,
			},
		})
		if err != nil {
			return err
		}

		cli := az.DeploymentsClient.Client()

		log.Info("waiting for arm template deployment to complete")
		err = future.WaitForCompletionRef(ctx, cli)
		if err != nil {
			log.Warnf("deployment failed: %#v", err)
			return err
		}

		return nil
	}
}

func enrich(cs *api.OpenShiftManagedCluster) error {
	tenantID := uuid.NewV4().String()
	clientID := uuid.NewV4().String()
	secret := "suspicious"
	cs.Properties.AzProfile = api.AzProfile{
		TenantID:       tenantID,
		SubscriptionID: uuid.NewV4().String(),
		ResourceGroup:  "testRG",
	}
	cs.Properties.AuthProfile.IdentityProviders = make([]api.IdentityProvider, 1)
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
	cs.ID = resourceid.ResourceID(cs.Properties.AzProfile.SubscriptionID, cs.Properties.AzProfile.ResourceGroup, "Microsoft.ContainerService/openshiftmanagedClusters", cs.Name)

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

	cs.Properties.PublicHostname = "openshift." + os.Getenv("RESOURCEGROUP") + "." + os.Getenv("DNS_DOMAIN")
	cs.Properties.RouterProfiles[0].PublicSubdomain = "apps." + os.Getenv("RESOURCEGROUP") + "." + os.Getenv("DNS_DOMAIN")

	if cs.Properties.FQDN == "" {
		cs.Properties.FQDN, err = random.FQDN(cs.Location+".cloudapp.azure.com", 20)
		if err != nil {
			return err
		}
	}

	if cs.Properties.RouterProfiles[0].FQDN == "" {
		cs.Properties.RouterProfiles[0].FQDN, err = random.FQDN(cs.Location+".cloudapp.azure.com", 20)
		if err != nil {
			return err
		}
	}

	return nil
}

func (w fakeResponseWrapper) DoRaw() ([]byte, error) {
	return w.raw, nil
}

func (w fakeResponseWrapper) Stream() (io.ReadCloser, error) {
	return nil, nil
}

func newFakeResponseWrapper(raw []byte) fakeResponseWrapper {
	return fakeResponseWrapper{raw: raw}
}

type fakeResponseWrapper struct {
	raw []byte
}

func TestCreateThenUpdateCausesNoRotations(t *testing.T) {
	cs := &api.OpenShiftManagedCluster{
		Config: api.Config{
			ConfigStorageAccount:      "config",
			RegistryStorageAccount:    "registry",
			RegistryStorageAccountKey: "foo",

			SSHKey: testtls.DummyPrivateKey,
			Certificates: api.CertificateConfig{
				Ca:            api.CertKeyPair{Cert: testtls.DummyCertificate, Key: testtls.DummyPrivateKey},
				NodeBootstrap: api.CertKeyPair{Cert: testtls.DummyCertificate, Key: testtls.DummyPrivateKey},
			},
		},
		Properties: api.Properties{
			AgentPoolProfiles: []api.AgentPoolProfile{
				{Role: api.AgentPoolProfileRoleMaster, Count: 3, Name: "master", VMSize: "Standard_D2s_v3"},
				{Role: api.AgentPoolProfileRoleCompute, Count: 1, Name: "compute", VMSize: "Standard_D2s_v3"},
				{Role: api.AgentPoolProfileRoleInfra, Count: 2, Name: "infra", VMSize: "Standard_D2s_v3"},
			},
		},
	}
	log := logrus.NewEntry(logrus.StandardLogger())
	ctx := context.Background()

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

	data, err := ioutil.ReadFile("../../pluginconfig/pluginconfig-311.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var template *pluginapi.Config
	if err := yaml.Unmarshal(data, &template); err != nil {
		t.Fatal(err)
	}

	template.Certificates.GenevaLogging.Cert = testtls.DummyCertificate
	template.Certificates.GenevaLogging.Key = testtls.DummyPrivateKey
	template.Certificates.GenevaMetrics.Cert = testtls.DummyCertificate
	template.Certificates.GenevaMetrics.Key = testtls.DummyPrivateKey
	template.GenevaImagePullSecret = []byte("pullSecret")
	template.ImagePullSecret = []byte("imagePullSecret")

	az := fakecloud.NewFakeAzureCloud(log, []compute.VirtualMachineScaleSetVM{}, []compute.VirtualMachineScaleSet{}, secrets, []storage.Account{}, map[string]map[string][]byte{})
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
				Name:      get.GetName(),
				Namespace: get.GetNamespace(),
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
	kc := kubeclient.NewFakeKubeclient(log, cli, seccli)
	p := &plugin{
		pluginConfig: template,
		testConfig:   api.TestConfig{RunningUnderTest: true},
		upgraderFactory: func(ctx context.Context, log *logrus.Entry, cs *api.OpenShiftManagedCluster, initializeStorageClients, disableKeepAlives bool, testConfig api.TestConfig) (cluster.Upgrader, error) {
			return cluster.NewFakeUpgrader(ctx, log, cs, testConfig, kc, az)
		},
		configInterfaceFactory: config.New,
		log:                    log,
		now:                    time.Now,
	}
	err = enrich(cs)
	if err != nil {
		t.Fatal(err)
	}

	err = p.GenerateConfig(ctx, cs, false)
	if err != nil {
		t.Fatal(err)
	}

	if err := p.CreateOrUpdate(ctx, cs, false, getFakeDeployer(log, cs, az)); err != nil {
		t.Errorf("plugin.CreateOrUpdate [create] error = %v", err)
	}
	bsc := az.StorageClient.GetBlobService()
	updateBlobService := updateblob.NewBlobService(bsc)
	beforeBlob, err := updateBlobService.Read()
	// - update
	if err := p.CreateOrUpdate(ctx, cs, true, getFakeDeployer(log, cs, az)); err != nil {
		t.Errorf("plugin.CreateOrUpdate [update] error = %v", err)
	}
	// - assert that the hashes are the same
	afterBlob, err := updateBlobService.Read()
	if !reflect.DeepEqual(beforeBlob, afterBlob) {
		t.Errorf("unexpected result:\n%#v\nexpected:\n%#v", spew.Sprint(beforeBlob), spew.Sprint(afterBlob))
	}
}
