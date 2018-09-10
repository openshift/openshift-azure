//+build e2e

package e2e

import (
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/managedcluster"
)

type testClient struct {
	kc *kubernetes.Clientset
	cs *api.OpenShiftManagedCluster
}

var c testClient

var _ = BeforeSuite(func() {
	cwd, _ := os.Getwd()
	// The current working dir of these tests is down a few levels from the root of the project.
	// We should traverse up that path so we can find the _data dir
	configPath := filepath.Join(cwd, "../../_data/containerservice.yaml")
	cs, err := managedcluster.ReadConfig(configPath)
	Expect(err).NotTo(HaveOccurred())
	c.cs = cs

	kc, err := managedcluster.ClientsetFromConfig(cs)
	Expect(err).NotTo(HaveOccurred())
	c.kc = kc
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
		list, err := c.kc.Core().Nodes().List(metav1.ListOptions{})
		Expect(err).NotTo(HaveOccurred())

		for _, node := range list.Items {
			kind := strings.Split(node.Name, "-")[0]
			Expect(labels).To(HaveKey(kind))
			for k, v := range labels[kind] {
				Expect(node.Labels).To(HaveKeyWithValue(k, v))
			}
		}
	})
})
