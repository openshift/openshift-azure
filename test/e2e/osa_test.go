//+build e2e

package e2e

import (
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type testClient struct {
	kc *kubernetes.Clientset
}

var c testClient

var _ = BeforeSuite(func() {

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	Expect(err).NotTo(HaveOccurred())

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	Expect(err).NotTo(HaveOccurred())
	c.kc = clientset
})

var _ = Describe("Openshift on Azure e2e tests", func() {
	It("should label nodes correctly", func() {
		labels := map[string]map[string]string{
			"master": {
				"node-role.kubernetes.io/master": "true",
				"openshift-infra":                "apiserver",
			},
			"compute": {
				"node-role.kubernetes.io/compute": "true",
				"region": "primary",
			},
			"infra": {
				"node-role.kubernetes.io/infra": "true",
				"region":                        "infra",
			},
		}
		list, err := c.kc.CoreV1().Nodes().List(metav1.ListOptions{})
		Expect(err).NotTo(HaveOccurred())

		for _, node := range list.Items {
			kind := strings.Split(node.Name, "-")[0]
			Expect(labels).To(HaveKey(kind))
			for k, v := range labels[kind] {
				Expect(node.Labels).To(HaveKeyWithValue(k, v))
			}
		}
	})

	It("should start prometheus correctly", func() {
		err := wait.Poll(2*time.Second, 20*time.Minute, func() (bool, error) {
			ss, err := c.kc.AppsV1().StatefulSets("openshift-metrics").Get("prometheus", metav1.GetOptions{})
			switch {
			case kerrors.IsNotFound(err):
				return false, nil
			case err == nil:
				specReplicas := int32(1)
				if ss.Spec.Replicas != nil {
					specReplicas = *ss.Spec.Replicas
				}
				return specReplicas == ss.Status.Replicas &&
					specReplicas == ss.Status.ReadyReplicas &&
					specReplicas == ss.Status.CurrentReplicas &&
					ss.Generation == ss.Status.ObservedGeneration, nil
			default:
				return false, err
			}
		})
		Expect(err).ToNot(HaveOccurred())
	})

	It("should run the correct image", func() {
		// image format is either:
		// docker.io/openshift || registry.access.redhat.com/openshift3
		format := strings.Split(c.cs.Config.Images.Format, "/")
		prefix := fmt.Sprintf("%v/%v", format[0], format[1])

		// these pods should have the same prefix
		podsToCheck := map[string]struct{}{
			"oauth-proxy":                    {},
			"origin-control-plane":           {},
			"origin-docker-registry":         {},
			"origin-haproxy-router":          {},
			"origin-node":                    {},
			"origin-service-catalog":         {},
			"origin-template-service-broker": {},
			"origin-web-console":             {},
			"prometheus-alert-buffer":        {},
			"prometheus-alertmanager":        {},
			"prometheus-node-exporter":       {},
			"prometheus":                     {},
		}

		pods, err := c.kc.CoreV1().Pods("").List(metav1.ListOptions{})
		Expect(err).ToNot(HaveOccurred())
		for _, pod := range pods.Items {
			// e2e check should ensure that no reg-aws images are running on box
			Expect(strings.Contains(pod.Status.ContainerStatuses[0].Image, "registry.reg-aws")).ToNot(BeTrue())
			parts := strings.Split(pod.Status.ContainerStatuses[0].Image, "/")

			if len(parts) < 3 {
				continue // skip unexpected image formats
			}
			if _, ok := podsToCheck[parts[2]]; ok {
				// also that centos boxes are running origin images, and rhel OCP images
				Expect(strings.HasPrefix(pod.Status.ContainerStatuses[0].Image, prefix)).To(BeTrue())
			}
		}
	})
})
