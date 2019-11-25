package specs

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift/openshift-azure/test/sanity"
)

var _ = Describe("Openshift on Azure metrics server tests [MetricsServer][EveryPR]", func() {
	It("should return metrics for every node in the cluster [CustomerAdmin][MetricsServer]", func() {
		By("retrieving node metrics")
		res, err1 := sanity.Checker.Client.CustomerAdmin.MetricsServerV1beta1.NodeMetricses().List(metav1.ListOptions{})
		nodes, err2 := sanity.Checker.Client.CustomerAdmin.CoreV1.Nodes().List(metav1.ListOptions{})
		Expect(err1).ToNot(HaveOccurred())
		Expect(err2).ToNot(HaveOccurred())
		Expect(res.Items).To(HaveLen(len(nodes.Items)))
	})

	It("should return valid metrics for a given node [CustomerAdmin][MetricsServer]", func() {
		By("retrieving master node metrics")
		res, err := sanity.Checker.Client.CustomerAdmin.MetricsServerV1beta1.NodeMetricses().Get("master-000000", metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Usage.Cpu().MilliValue() > 0)
		Expect(res.Usage.Memory().Value() > 0)
	})

	It("should return valid metrics for a given pod [CustomerAdmin][MetricsServer]", func() {
		By("retrieving metrics for pod master-api-master-000000")
		res, err := sanity.Checker.Client.CustomerAdmin.MetricsServerV1beta1.PodMetricses("kube-system").Get("master-api-master-000000", metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Containers[0].Usage.Cpu().MilliValue() > 0)
		Expect(res.Containers[0].Usage.Memory().Value() > 0)
	})
})
