package specs

import (
	"context"
	"io/ioutil"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/openshift/openshift-azure/pkg/util/jsonpath"
	"github.com/openshift/openshift-azure/test/sanity"
)

const (
	kafkaClusterName = "test-kafka-cluster"
	templateName     = "strimzi-ephemeral"
)

func validateKafkaCluster(ctx context.Context, namespace string) (err error) {
	sanity.Checker.Log.Debugf("validating that kafka CR is healthy")

	return wait.PollImmediate(5*time.Second, 5*time.Minute,
		func() (bool, error) {
			ko, err := sanity.Checker.Client.EndUser.GetKafka(kafkaClusterName, namespace)
			if kerrors.IsNotFound(err) {
				sanity.Checker.Log.Debugf("kafka CR not found")
				return false, nil
			}
			if err != nil {
				sanity.Checker.Log.Debugf("kafka CR %v", err)
				return false, err
			}

			status := jsonpath.MustCompile("$.status").MustGetObject(ko.Object)
			if status == nil {
				return false, nil
			}
			for _, cond := range status["conditions"].([]interface{}) {
				condition := cond.(map[string]interface{})
				if condition["type"] == "Ready" && condition["status"] == "True" {
					return true, nil
				}
			}

			return false, nil
		})
}

func createKafkaCluster(ctx context.Context, namespace string) (err error) {
	templdata, err := ioutil.ReadFile("../manifests/kafka/ephemeral-template.yaml") // running via `go test`
	if os.IsNotExist(err) {
		templdata, err = ioutil.ReadFile("test/manifests/kafka/ephemeral-template.yaml") // running via compiled test binary
	}
	if err != nil {
		sanity.Checker.Log.Error(err)
		return
	}
	var parameters = map[string]string{}
	parameters["CLUSTER_NAME"] = kafkaClusterName
	parameters["ZOOKEEPER_NODE_COUNT"] = "3"
	parameters["KAFKA_NODE_COUNT"] = "3"
	parameters["KAFKA_VERSION"] = "2.2.1"

	sanity.Checker.Log.Debugf("creating kafka cluster in %s", namespace)
	err = sanity.Checker.Client.EndUser.InstantiateTemplateFromBytes(templdata, namespace, parameters)
	if err != nil {
		sanity.Checker.Log.Error(err)
	}
	return
}

var _ = Describe("Openshift on Azure end user e2e tests [EndUser][EveryPR]", func() {
	It("should create and validate a strimzi kafka cluster [EndUser][Strimzi]", func() {
		sanity.Checker.Log.Debugf("creating openshift project for kafka cluster")
		ctx := context.Background()
		namespace, err := sanity.Checker.CreateProject(ctx)
		Expect(err).ToNot(HaveOccurred())

		By("creating strimzi kafka cluster")
		err = createKafkaCluster(ctx, namespace)
		Expect(err).ToNot(HaveOccurred())
		defer func() {
			By("deleting strimzi kafka cluster")
			_ = sanity.Checker.Client.EndUser.DeleteKafka(kafkaClusterName, namespace)
		}()

		By("validating strimzi kafka cluster")
		var errs []error
		if err := validateKafkaCluster(ctx, namespace); err != nil {
			errs = append(errs, err)
			if err = sanity.Checker.Client.EndUser.DumpInfo("strimzi", "validateKafkaCluster"); err != nil {
				errs = append(errs, err)
			}
		}
		Expect(errs).To(BeEmpty())
	})
})
