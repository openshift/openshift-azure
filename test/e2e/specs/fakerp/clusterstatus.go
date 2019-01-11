package fakerp

import (
	"context"
	"encoding/json"
	"os"
	"sort"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/api/core/v1"

	"github.com/openshift/openshift-azure/test/clients/azure"
)

var _ = Describe("Control Plane Pods Status E2E tests [AzureClusterReader][Fake][EveryPR]", func() {
	var (
		cli *azure.Client
	)

	BeforeEach(func() {
		var err error
		cli, err = azure.NewClientFromEnvironment(false)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should allow an SRE to fetch the status of control plane pods", func() {
		By("Using the OSA admin client to fetch the raw cluster status")
		status, err := cli.OpenShiftManagedClustersAdmin.GetControlPlanePods(context.Background(), os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"))
		Expect(err).NotTo(HaveOccurred())
		Expect(status).NotTo(BeNil())

		By("Unmarshalling the returned raw cluster status to a Pod slice")
		var pods []v1.Pod
		err = json.Unmarshal(status.Items, &pods)
		Expect(err).NotTo(HaveOccurred())

		By("Constructing a mapping of namespaces to pods")
		podnames := make(map[string][]string)
		for _, pod := range pods {
			list := append(podnames[pod.Namespace], pod.Name)
			sort.Strings(list)
			podnames[pod.Namespace] = list
		}

		By("Verifying that certain namespaces contain certain expected pods")
		namespace := "kube-system"
		Expect(sort.SearchStrings(podnames[namespace], "controllers-master-000000")).NotTo(Equal(len(podnames[namespace])))
		Expect(sort.SearchStrings(podnames[namespace], "controllers-master-000001")).NotTo(Equal(len(podnames[namespace])))
		Expect(sort.SearchStrings(podnames[namespace], "controllers-master-000002")).NotTo(Equal(len(podnames[namespace])))
		Expect(sort.SearchStrings(podnames[namespace], "master-api-master-000000")).NotTo(Equal(len(podnames[namespace])))
		Expect(sort.SearchStrings(podnames[namespace], "master-api-master-000001")).NotTo(Equal(len(podnames[namespace])))
		Expect(sort.SearchStrings(podnames[namespace], "master-api-master-000002")).NotTo(Equal(len(podnames[namespace])))
		Expect(sort.SearchStrings(podnames[namespace], "master-etcd-master-000000")).NotTo(Equal(len(podnames[namespace])))
		Expect(sort.SearchStrings(podnames[namespace], "master-etcd-master-000001")).NotTo(Equal(len(podnames[namespace])))
		Expect(sort.SearchStrings(podnames[namespace], "master-etcd-master-000002")).NotTo(Equal(len(podnames[namespace])))
		Expect(sort.SearchStrings(podnames[namespace], "sync-master-000000")).NotTo(Equal(len(podnames[namespace])))
	})
})
