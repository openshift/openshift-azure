package fakerp

import (
	"fmt"
	"regexp"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift/openshift-azure/test/sanity"
)

var _ = Describe("sync pod tests [EveryPR]", func() {
	It("should not continuously update objects", func() {
		pods, err := sanity.Checker.Client.Admin.CoreV1.Pods("kube-system").List(metav1.ListOptions{LabelSelector: "app=sync"})
		Expect(err).ToNot(HaveOccurred())
		Expect(len(pods.Items)).To(BeNumerically(">", 0))

		rx := regexp.MustCompile(`(?ms)^[^\n]* msg="starting sync"[^\n]*$.*?^[^\n]* msg="sync done"[^\n]*$`)
		for {
			b, err := sanity.Checker.Client.Admin.CoreV1.Pods("kube-system").GetLogs(pods.Items[0].Name, &corev1.PodLogOptions{}).DoRaw()
			Expect(err).ToNot(HaveOccurred())

			runs := rx.FindAllString(string(b), -1)
			if len(runs) > 1 {
				for i, run := range runs {
					Expect(run).To(Not(ContainSubstring("level=error")))
					if i == 0 {
						By(fmt.Sprintf("ignoring the run %d/%d as it will have updates", i, len(runs)))
						continue
					}
					// check for constantly updated objects
					By(fmt.Sprintf("inspecting run %d/%d", i, len(runs)))
					Expect(run).To(Not(ContainSubstring("Update")))
				}
				break
			} else {
				By("waiting for another sync loop")
				time.Sleep(time.Minute)
			}
		}
	})
})
