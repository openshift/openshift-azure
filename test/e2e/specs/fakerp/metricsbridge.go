package fakerp

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"regexp"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/ghodss/yaml"
	"github.com/prometheus/client_golang/api"
	"github.com/prometheus/client_golang/api/prometheus/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift/openshift-azure/pkg/entrypoint/metricsbridge"
	"github.com/openshift/openshift-azure/pkg/util/roundtrippers"
	"github.com/openshift/openshift-azure/test/sanity"
)

// Test checks if all configured queries are valid prometheus queries, and if the metric exists
// Existence of metric is checked by either:
//  - query returning data
//  - query being on a list of manually verified correct queries (that normally don't return any data)
//  - query having ' != 0' at the end, that returns data after this condition is removed
//  - query calculating rate( x [1m]) of a metric x, when querying for metric x returns data
var _ = Describe("Metricsbridge E2E check configured queries [Fake][EveryPR]", func() {
	// queries which usually return no datapoints, but have been verified as valid manually
	manuallyVerifiedQueries := map[string]bool{
		"kubelet_runtime_operations_errors != 0":                                                  true,
		"kubelet_docker_operations_errors != 0":                                                   true,
		"kube_job_complete{namespace=~\"default|openshift|openshift-.+|kube-.+|\"} != 0":          true,
		"kube_job_status_completion_time{namespace=~\"default|openshift|openshift-.+|kube-.+|\"}": true,
	}
	It("should be possible to get data for each metric in defined queries", func() {
		By("getting the configMap")
		mbConfigMap, err := sanity.Checker.Client.Admin.CoreV1.ConfigMaps("openshift-azure-monitoring").Get("metrics-bridge", meta_v1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(mbConfigMap).ToNot(BeNil())

		var mbConfig metricsbridge.MetricsConfig

		By("unmarshalling the ConfigMap")
		err = yaml.Unmarshal([]byte(mbConfigMap.Data["config.yaml"]), &mbConfig)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(mbConfig.Queries)).NotTo(BeZero())

		By("getting the Prometheus token and URL")
		token, err := sanity.Checker.Client.Admin.GetServiceAccountToken("openshift-monitoring", "prometheus-k8s")
		Expect(err).NotTo(HaveOccurred())
		Expect(token).NotTo(BeEmpty())

		route, err := sanity.Checker.Client.Admin.RouteV1.Routes("openshift-monitoring").Get("prometheus-k8s", meta_v1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(route).NotTo(BeNil())

		By("dialing Prometheus")
		cli, err := api.NewClient(api.Config{
			Address: "https://" + route.Spec.Host,
			RoundTripper: &roundtrippers.AuthorizingRoundTripper{
				RoundTripper: &http.Transport{
					TLSClientConfig: &tls.Config{
						RootCAs:            nil,
						InsecureSkipVerify: true,
					},
				},
				Token: string(token),
			},
		})
		Expect(err).NotTo(HaveOccurred())

		prometheusApi := v1.NewAPI(cli)

		for _, q := range mbConfig.Queries {
			By(fmt.Sprintf("checking query %s", q.Query))
			// checking if the whole query works
			value, err := prometheusApi.Query(context.Background(), q.Query, time.Time{})
			Expect(err).NotTo(HaveOccurred())
			if len(value.String()) == 0 {
				//check if query has been verified manually despite not returning datapoints
				_, present := manuallyVerifiedQueries[q.Query]
				if !present {
					qStr := q.Query

					//in case of rate( x [1m]), unwrap x
					re := regexp.MustCompile(`rate\((.+)\[1m\]\)`)
					qStr = re.ReplaceAllString(qStr, "$1")

					// find any expressions in parenthesis, and replace qStr with the first one found
					// e.g. sum(foo{bar}) without (baz) != 0 -> foo{bar}
					re = regexp.MustCompile(`\((.+?)\)`)
					m := re.FindAllStringSubmatch(qStr, -1)
					if m != nil {
						//only if there is at least one expression in parenthesis
						qStr = m[0][1]
					}

					//remove anything after the first non-alphanumeric character
					re = regexp.MustCompile(`(?i)[^_a-z0-9].*`)
					qStr = re.ReplaceAllString(qStr, "")

					//check if it returns datapoints
					value, err := prometheusApi.Query(context.Background(), qStr, time.Time{})
					Expect(err).NotTo(HaveOccurred())
					Expect(value.String()).NotTo(BeEmpty())
				}
			}
		}

	})
})
