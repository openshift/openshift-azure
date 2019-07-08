package fakerp

import (
	"regexp"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift/openshift-azure/test/sanity"
)

var _ = Describe("sync pod tests [Fake][EveryPR]", func() {
	It("should not continuously update objects", func() {
		pods, err := sanity.Checker.Client.Admin.CoreV1.Pods("kube-system").List(metav1.ListOptions{LabelSelector: "app=sync"})
		Expect(err).ToNot(HaveOccurred())
		Expect(len(pods.Items)).To(BeNumerically(">", 0))

		b, err := sanity.Checker.Client.Admin.CoreV1.Pods("kube-system").GetLogs(pods.Items[0].Name, &corev1.PodLogOptions{}).DoRaw()
		Expect(err).ToNot(HaveOccurred())

		rx := regexp.MustCompile(`(?ms)^[^\n]* msg="starting sync"[^\n]*$.*?^[^\n]* msg="sync done"[^\n]*$`)
		runs := rx.FindAllString(string(b), -1)
		// check for constantly updated objects
		Expect(runs).To(ContainElement(Not(ContainSubstring("Update"))))
		// check for update once objects
		Expect(runs).To(ContainElement(Not(ContainSubstring("Deploy once resource project-request detected"))))
	})
})
