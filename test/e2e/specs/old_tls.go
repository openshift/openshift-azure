package specs

import (
	"crypto/tls"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift/openshift-azure/test/sanity"
)

var _ = Describe("Openshift on Azure TLS tests [TLS][EveryPR]", func() {
	It("should not support TLS 1.0 or 1.1 [TLS]", func() {
		// Known route: cluster Prometheus
		route, err := sanity.Checker.Client.Admin.RouteV1.Routes("openshift-monitoring").Get("prometheus-k8s", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		/* #nosec - connecting to internal API of a self created cluster */
		_, err10 := tls.Dial("tcp", route.Spec.Host+":443", &tls.Config{
			InsecureSkipVerify: true,
			MinVersion:         tls.VersionTLS10,
			MaxVersion:         tls.VersionTLS10,
		})

		Expect(err10).To(HaveOccurred())

		/* #nosec - connecting to internal API of a self created cluster */
		_, err11 := tls.Dial("tcp", route.Spec.Host+":443", &tls.Config{
			InsecureSkipVerify: true,
			MinVersion:         tls.VersionTLS11,
			MaxVersion:         tls.VersionTLS11,
		})

		Expect(err11).To(HaveOccurred())

		/* #nosec - connecting to internal API of a self created cluster */
		// Ensure we *can* connect with modern TLS
		_, err12 := tls.Dial("tcp", route.Spec.Host+":443", &tls.Config{
			InsecureSkipVerify: true,
			MinVersion:         tls.VersionTLS12,
			MaxVersion:         tls.VersionTLS12,
		})

		Expect(err12).NotTo(HaveOccurred())
	})
})
