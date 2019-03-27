package specs

import (
	"context"
	"io/ioutil"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/openshift/openshift-azure/pkg/util/ready"
	"github.com/openshift/openshift-azure/test/e2e/standard"
	"github.com/openshift/openshift-azure/test/sanity"
)

// ValidateOSBAApp verifies that the created application runs normally
func validateOSBAApp(ctx context.Context, cookie interface{}) []*standard.TestError {
	var errs []*standard.TestError

	sanity.Checker.Log.Debugf("validating that all osba components are healthy")
	err := wait.Poll(2*time.Second, 20*time.Minute, ready.CheckClusterServiceBrokerIsReady(sanity.Checker.Client.CustomerAdmin.ServicecatalogV1beta1.ClusterServiceBrokers(), "osba"))
	if err != nil {
		sanity.Checker.Log.Error(err)
		errs = append(errs, &standard.TestError{Err: err, Bucket: "ready.CheckClusterServiceBrokerIsReady"})
	}
	return errs
}

// DeleteOSBAApp deletes the OSBA Application after tests completed
func deleteOSBAApp(ctx context.Context, cookie interface{}) []*standard.TestError {
	var errs []*standard.TestError

	// the OSBA clusterservicebroker is not namespaced, remove it
	sanity.Checker.Log.Debugf("deleting azure cluster service broker")
	err := sanity.Checker.Client.CustomerAdmin.ServicecatalogV1beta1.ClusterServiceBrokers().Delete("osba", &metav1.DeleteOptions{})
	if err != nil {
		sanity.Checker.Log.Error(err)
		errs = append(errs, &standard.TestError{Err: err, Bucket: "sanity.Checker.Client.CustomerAdmin.ServicecatalogV1beta1.ClusterServiceBrokers().Delete"})
	}

	sanity.Checker.Log.Debugf("deleting openshift project for broker apps")
	err = sanity.Checker.Client.CustomerAdmin.CleanupProject(cookie.(string))
	if err != nil {
		sanity.Checker.Log.Error(err)
		errs = append(errs, &standard.TestError{Err: err, Bucket: "sanity.Checker.Client.CustomerAdmin.CleanupProject"})
	}

	return errs
}

// CreateOSBAApp creates the OSBA Application under test
func createOSBAApp(ctx context.Context) (interface{}, []*standard.TestError) {
	var errs []*standard.TestError
	sanity.Checker.Log.Debugf("creating openshift project for broker app")

	err := sanity.Checker.Client.CustomerAdmin.CreateProject("osba")
	if err != nil {
		sanity.Checker.Log.Error(err)
		errs = append(errs, &standard.TestError{Err: err, Bucket: "sanity.Checker.Client.CustomerAdmin.CreateProject"})
		return nil, errs
	}

	// Get IDs and Credentials for template substitution
	var parameters = map[string]string{}
	parameters["AZURE_SUBSCRIPTION_ID"] = os.Getenv("AZURE_SUBSCRIPTION_ID")
	parameters["AZURE_TENANT_ID"] = os.Getenv("AZURE_TENANT_ID")
	parameters["AZURE_CLIENT_ID"] = os.Getenv("AZURE_CLIENT_ID")
	parameters["AZURE_CLIENT_SECRET"] = os.Getenv("AZURE_CLIENT_SECRET")

	sanity.Checker.Log.Debugf("creating openshift broker app in %s", "osba")
	sanity.Checker.Log.Debugf("instantiating template for %s", "osba")

	templdata, err := ioutil.ReadFile("../manifests/osba/osba-os-template.yaml") // running via `go test`
	if os.IsNotExist(err) {
		templdata, err = ioutil.ReadFile("test/e2e/manifests/osba/osba-os-template.yaml") // running via compiled test binary
	}
	if err != nil {
		sanity.Checker.Log.Error(err)
		errs = append(errs, &standard.TestError{Err: err, Bucket: "sanity.Checker.Client.CustomerAdmin.CreateProject"})
		return nil, errs
	}

	err = sanity.Checker.Client.CustomerAdmin.InstantiateTemplateFromBytes(templdata, "osba", parameters)
	if err != nil {
		sanity.Checker.Log.Error(err)
		errs = append(errs, &standard.TestError{Err: err, Bucket: "sanity.Checker.Client.CustomerAdmin.InstantiateTemplateFromBytes"})
		return nil, errs
	}

	return "osba", errs
}

var _ = Describe("Openshift on Azure end user e2e tests [CostumerAdmin][Fake]", func() {
	It("should create and validate an OpenShift Broker App [CustomerAdmin][Fake][OSBA]", func() {
		ctx := context.Background()
		By("creating openshift broker app")
		namespace, errs := createOSBAApp(ctx)
		Expect(errs).To(BeEmpty())
		defer func() {
			By("deleting openshift broker app")
			_ = deleteOSBAApp(ctx, namespace)
		}()

		By("validating openshift broker app")
		errs = validateOSBAApp(ctx, namespace)
		Expect(errs).To(BeEmpty())
	})
})
