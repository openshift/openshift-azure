package standard

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	internalapi "github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/keyvault"
	"github.com/openshift/openshift-azure/pkg/util/enrich"
	"github.com/openshift/openshift-azure/pkg/util/random"
	"github.com/openshift/openshift-azure/test/clients/openshift"
)

type TestError struct {
	Bucket string
	Err    error
}

var _ error = &TestError{}

func (te *TestError) Error() string {
	return te.Bucket + ": " + te.Err.Error()
}

type DeepTestInterface interface {
	CreateTestApp(ctx context.Context) (interface{}, []*TestError)
	ValidateTestApp(ctx context.Context, cookie interface{}) []*TestError
	ValidateCluster(ctx context.Context) []*TestError
	DeleteTestApp(ctx context.Context, cookie interface{}) []*TestError
}

type SanityChecker struct {
	Log    *logrus.Entry
	cs     *internalapi.OpenShiftManagedCluster
	Client *openshift.ClientSet
}

var _ DeepTestInterface = &SanityChecker{}

// NewSanityChecker creates a new deep test sanity checker for OpenshiftManagedCluster resources.
func NewSanityChecker(ctx context.Context, log *logrus.Entry, cs *internalapi.OpenShiftManagedCluster) (*SanityChecker, error) {
	scc := &SanityChecker{
		Log: log,
		cs:  cs,
	}

	vaultauthorizer, err := azureclient.NewAuthorizer(cs.Properties.MasterServicePrincipalProfile.ClientID, cs.Properties.MasterServicePrincipalProfile.Secret, cs.Properties.AzProfile.TenantID, azureclient.KeyVaultEndpoint)
	if err != nil {
		return nil, err
	}

	kvc := keyvault.NewKeyVaultClient(ctx, log, vaultauthorizer)

	err = enrich.CertificatesFromVault(ctx, kvc, cs)
	if err != nil {
		return nil, err
	}
	scc.Client, err = openshift.NewClientSet(log, cs)
	if err != nil {
		return nil, err
	}
	return scc, nil
}

func (sc *SanityChecker) CreateTestApp(ctx context.Context) (interface{}, []*TestError) {
	var errs []*TestError
	sc.Log.Debugf("creating openshift project for test apps")
	namespace, err := sc.createProject(ctx)
	if err != nil {
		sc.Log.Error(err)
		errs = append(errs, &TestError{Err: err, Bucket: "createProject"})
		return nil, errs
	}
	sc.Log.Debugf("creating stateful test app in %s", namespace)
	err = sc.createStatefulApp(ctx, namespace)
	if err != nil {
		sc.Log.Error(err)
		errs = append(errs, &TestError{Err: err, Bucket: "createStatefulApp"})
	}
	if len(errs) > 0 {
		err := sc.Client.EndUser.DumpInfo(namespace, "createStatefulApp")
		if err != nil {
			sc.Log.Warn(err)
		}
	}
	return namespace, errs
}

func (sc *SanityChecker) debugValidateTestApp() error {
	nodes, err := sc.Client.Admin.CoreV1.Nodes().List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	var nodename string
	for _, node := range nodes.Items {
		if strings.HasPrefix(node.Name, "compute-") {
			nodename = node.Name
			break
		}
	}
	if nodename == "" {
		return fmt.Errorf("could not find compute node")
	}

	b, err := sc.Client.Admin.CoreV1.RESTClient().Get().
		Resource("nodes").
		Name(nodename).
		SubResource("proxy").
		Suffix("/debug/pprof/goroutine").
		Param("debug", "2").
		DoRaw()
	if err != nil {
		return err
	}

	return ioutil.WriteFile(os.Getenv("ARTIFACTS")+"/compute-testapp-goroutine-dump", b, 0666)
}

func (sc *SanityChecker) ValidateTestApp(ctx context.Context, cookie interface{}) (errs []*TestError) {
	namespace := cookie.(string)
	sc.Log.Debugf("validating stateful test app in %s", namespace)
	err := sc.validateStatefulApp(ctx, namespace)
	if err != nil {
		sc.Log.Error(err)
		errs = append(errs, &TestError{Err: err, Bucket: "validateStatefulApp"})
	}
	if os.Getenv("ARTIFACTS") != "" && err != nil {
		err = sc.debugValidateTestApp()
		if err != nil {
			errs = append(errs, &TestError{Err: err, Bucket: "validateStatefulApp"})
		}
	}
	if len(errs) > 0 {
		err := sc.Client.EndUser.DumpInfo(namespace, "validateStatefulApp")
		if err != nil {
			sc.Log.Warn(err)
		}
	}
	return
}

func (sc *SanityChecker) ValidateCluster(ctx context.Context) (errs []*TestError) {
	sc.Log.Debugf("validating that nodes are labelled correctly")
	err := sc.checkNodesLabelledCorrectly(ctx)
	if err != nil {
		sc.Log.Error(err)
		errs = append(errs, &TestError{Err: err, Bucket: "checkNodesLabelledCorrectly"})
	}
	sc.Log.Debugf("validating that all monitoring components are healthy")
	err = sc.checkMonitoringStackHealth(ctx)
	if err != nil {
		sc.Log.Error(err)
		sc.Client.EndUser.DumpInfo("", "checkMonitoringStackHealth")
		errs = append(errs, &TestError{Err: err, Bucket: "checkMonitoringStackHealth"})
	}
	sc.Log.Debugf("validating that pod disruption budgets are immutable")
	err = sc.checkDisallowsPdbMutations(ctx)
	if err != nil {
		sc.Log.Error(err)
		sc.Client.EndUser.DumpInfo("", "checkDisallowsPdbMutations")
		errs = append(errs, &TestError{Err: err, Bucket: "checkDisallowsPdbMutations"})
	}
	sc.Log.Debugf("validating that an end user cannot access infrastructure components")
	err = sc.checkCannotAccessInfraResources(ctx)
	if err != nil {
		sc.Log.Error(err)
		sc.Client.EndUser.DumpInfo("", "checkCannotAccessInfraResources")
		errs = append(errs, &TestError{Err: err, Bucket: "checkCannotAccessInfraResources"})
	}
	sc.Log.Debugf("validating that the cluster can pull redhat.io images")
	err = sc.checkCanDeployRedhatIoImages(ctx)
	if err != nil {
		sc.Log.Error(err)
		sc.Client.EndUser.DumpInfo("", "checkCanDeployRedhatIoImages")
		errs = append(errs, &TestError{Err: err, Bucket: "checkCanDeployRedhatIoImages"})
	}
	sc.Log.Debugf("validating that the cluster can create ELB and ILB")
	err = sc.checkCanCreateLB(ctx)
	if err != nil {
		sc.Log.Error(err)
		sc.Client.EndUser.DumpInfo("", "checkCanCreateLB")
		errs = append(errs, &TestError{Err: err, Bucket: "checkCanCreateLB"})
	}
	sc.Log.Debugf("validating that cluster services are available")
	err = sc.checkCanAccessServices(ctx)
	if err != nil {
		sc.Log.Error(err)
		sc.Client.EndUser.DumpInfo("", "checkCanAccessServices")
		errs = append(errs, &TestError{Err: err, Bucket: "checkCanAccessServices"})
	}
	sc.Log.Debugf("validating that the cluster can use azure-file storage")
	err = sc.checkCanUseAzureFileStorage(ctx)
	if err != nil {
		sc.Log.Error(err)
		sc.Client.EndUser.DumpInfo("", "checkCanUseAzureFile")
		errs = append(errs, &TestError{Err: err, Bucket: "checkCanUseAzureFile"})
	}
	sc.Log.Debugf("validating that the cluster enforces emptydir quotas")
	err = sc.checkEnforcesEmptyDirQuotas(ctx)
	if err != nil {
		sc.Log.Error(err)
		errs = append(errs, &TestError{Err: err, Bucket: "checkEnforcesEmptyDirQuotas"})
	}
	return
}

func (sc *SanityChecker) DeleteTestApp(ctx context.Context, cookie interface{}) []*TestError {
	var errs []*TestError
	sc.Log.Debugf("deleting openshift project for test apps")
	err := sc.deleteProject(ctx, cookie.(string))
	if err != nil {
		sc.Log.Error(err)
		errs = append(errs, &TestError{Err: err, Bucket: "deleteProject"})
	}
	return errs
}

func (sc *SanityChecker) createProject(ctx context.Context) (string, error) {
	template, err := random.LowerCaseAlphanumericString(5)
	if err != nil {
		return "", err
	}
	namespace := "e2e-test-" + template
	err = sc.Client.EndUser.CreateProject(namespace)
	if err != nil {
		return "", err
	}
	return namespace, nil
}

func (sc *SanityChecker) deleteProject(ctx context.Context, namespace string) error {
	err := sc.Client.EndUser.CleanupProject(namespace)
	if err != nil {
		return err
	}
	return nil
}

func (sc *SanityChecker) createService(name, namespace string, lbType corev1.ServiceType, annotations map[string]string) error {
	lb := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: annotations,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "port",
					Port: 8080,
				},
			},
			Type: lbType,
		},
	}
	_, err := sc.Client.EndUser.CoreV1.Services(namespace).Create(lb)
	if err != nil {
		return err
	}
	return nil
}
