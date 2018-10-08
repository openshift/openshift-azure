//+build e2e

package e2e

import (
	"strings"
	"time"

	"github.com/ghodss/yaml"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/openshift/openshift-azure/pkg/jsonpath"
)

var _ = Describe("Openshift on Azure admin e2e tests [AzureClusterReader]", func() {
	defer GinkgoRecover()

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
		pods, err := c.kc.CoreV1().Pods("").List(metav1.ListOptions{})
		Expect(err).ToNot(HaveOccurred())
		// e2e check should ensure that no reg-aws images are running on box
		for _, pod := range pods.Items {
			for _, cntr := range pod.Spec.Containers {
				Expect(strings.HasPrefix(cntr.Image, "registry.reg-aws.openshift.com/")).ToNot(BeTrue())
			}
		}
		// fetch master-000000 and determine the OS type
		master0, _ := c.kc.CoreV1().Nodes().Get("master-000000", metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		// set registryPrefix to appropriate string based upon master's OS type
		var registryPrefix string
		if strings.HasPrefix(master0.Status.NodeInfo.OSImage, "Red Hat Enterprise") {
			registryPrefix = "registry.access.redhat.com/openshift3/ose-"
		} else {
			registryPrefix = "docker.io/openshift/origin-"
		}

		// Check all Configmaps for image format matches master's OS type
		// format: registry.access.redhat.com/openshift3/ose-${component}:${version}
		maps, err := c.kc.CoreV1().ConfigMaps("openshift-node").List(metav1.ListOptions{})
		Expect(err).ToNot(HaveOccurred())
		var nodeConfig map[string]interface{}
		for _, cm := range maps.Items {
			err = yaml.Unmarshal([]byte(cm.Data["node-config.yaml"]), &nodeConfig)
			format := jsonpath.MustCompile("$.imageConfig.format").MustGetString(nodeConfig)
			Expect(strings.HasPrefix(format, registryPrefix)).To(BeTrue())
		}
	})
})
