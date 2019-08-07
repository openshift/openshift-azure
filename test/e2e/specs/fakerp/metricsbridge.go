package fakerp

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/ghodss/yaml"
	"github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/prometheus/promql"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift/openshift-azure/pkg/entrypoint/metricsbridge"
	"github.com/openshift/openshift-azure/pkg/util/roundtrippers"
	"github.com/openshift/openshift-azure/test/sanity"
)

func getMetrics(ast promql.Expr) []string {
	metrics := []string{}
	switch t := ast.(type) {
	case *promql.AggregateExpr:
		metrics = append(metrics, getMetrics(t.Expr)...)
		metrics = append(metrics, getMetrics(t.Param)...)
	case *promql.BinaryExpr:
		metrics = append(metrics, getMetrics(t.LHS)...)
		metrics = append(metrics, getMetrics(t.RHS)...)
	case *promql.Call:
		for _, expr := range t.Args {
			metrics = append(metrics, getMetrics(expr)...)
		}
	case *promql.MatrixSelector:
		metrics = append(metrics, t.Name)
	case *promql.VectorSelector:
		metrics = append(metrics, t.Name)
	case *promql.SubqueryExpr:
		metrics = append(metrics, getMetrics(t.Expr)...)
	case *promql.ParenExpr:
		metrics = append(metrics, getMetrics(t.Expr)...)
	case *promql.UnaryExpr:
		metrics = append(metrics, getMetrics(t.Expr)...)
	case *promql.NumberLiteral, *promql.StringLiteral, nil:
		//literals will always work, no metrics to verify
	default:
		panic("Promql AST type unknown")
	}
	return metrics
}

func parseMetrics(query string) ([]string, error) {
	ast, err := promql.ParseExpr(query)
	l := getMetrics(ast)
	if err != nil {
		return nil, err
	}
	return l, nil
}

// Test checks if all configured queries are valid prometheus queries, and if the metric exists
// Existence of metric is checked by either:
//  - query returning data
//  - query being on a list of manually verified correct queries (that normally don't return any data)
//  - query having ' != 0' at the end, that returns data after this condition is removed
//  - query calculating rate( x [1m]) of a metric x, when querying for metric x returns data
var _ = Describe("Metricsbridge E2E check configured queries ", func() {
	// queries which usually return no datapoints, but have been verified as valid manually
	manuallyVerifiedQueries := map[string]bool{
		"kubelet_runtime_operations_errors != 0": true,
		"kubelet_docker_operations_errors != 0":  true,
	}
	fmt.Print(manuallyVerifiedQueries)
	It("should be possible to get data for each metric in defined queries", func() {
		By("getting the configMap")
		mbConfigMap, err := sanity.Checker.Client.Admin.CoreV1.ConfigMaps("openshift-azure-monitoring").Get("metrics-bridge", metav1.GetOptions{})
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

		route, err := sanity.Checker.Client.Admin.RouteV1.Routes("openshift-monitoring").Get("prometheus-k8s", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(route).NotTo(BeNil())

		By("dialing Prometheus")
		cli, err := api.NewClient(api.Config{
			Address: "https://" + route.Spec.Host,
			RoundTripper: &roundtrippers.AuthorizingRoundTripper{
				RoundTripper: &http.Transport{
					/* #nosec - connecting to internal API of a self created cluster */
					TLSClientConfig: &tls.Config{
						RootCAs:            nil,
						InsecureSkipVerify: true,
					},
				},
				Token: string(token),
			},
		})
		Expect(err).NotTo(HaveOccurred())

		prometheusApi := promv1.NewAPI(cli)

		for _, q := range mbConfig.Queries {
			By(fmt.Sprintf("checking query %s", q.Query))
			// checking if the whole query works
			value, err := prometheusApi.Query(context.Background(), q.Query, time.Time{})
			Expect(err).NotTo(HaveOccurred())
			if len(value.String()) == 0 {
				//parse the expression to get to metic names
				m, _ := parseMetrics(q.Query)
				//check if all metrics in the query return values
				for _, metric := range m {
					v, err := prometheusApi.Query(context.Background(), metric, time.Time{})
					Expect(err).NotTo(HaveOccurred())
					Expect(v.String()).NotTo(BeEmpty())
				}
			}
		}

	})
})
