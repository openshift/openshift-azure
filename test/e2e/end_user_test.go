//+build e2e

package e2e

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	templatev1 "github.com/openshift/api/template/v1"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	policy "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
)

var _ = Describe("Openshift on Azure end user e2e tests [EndUser]", func() {
	defer GinkgoRecover()

	BeforeEach(func() {
		namespace := nameGen.generate("e2e-test-")
		c.createProject(namespace)
	})

	AfterEach(func() {
		if CurrentGinkgoTestDescription().Failed {
			if err := c.dumpInfo(); err != nil {
				logrus.Warn(err)
			}
		}
		c.cleanupProject(10 * time.Minute)
	})

	It("should disallow PDB mutations", func() {
		maxUnavailable := intstr.FromInt(1)
		selector, err := metav1.ParseToLabelSelector("key=value")
		Expect(err).NotTo(HaveOccurred())

		pdb := &policy.PodDisruptionBudget{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
			Spec: policy.PodDisruptionBudgetSpec{
				MaxUnavailable: &maxUnavailable,
				Selector:       selector,
			},
		}

		_, err = c.kc.Policy().PodDisruptionBudgets(c.namespace).Create(pdb)
		Expect(kerrors.IsForbidden(err)).To(Equal(true))
	})

	It("should deploy a template and check the visit counter increments", func() {
		const TPL = "cakephp-mysql-example"
		var regex = regexp.MustCompile(`id="count-value">(\d+)<`)

		// Create the template
		By(fmt.Sprintf("creating the template (%v)", time.Now()))
		template, err := c.tc.Templates("openshift").Get(
			TPL, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		// Instantiate the template
		_, err = c.tc.TemplateInstances(c.namespace).Create(
			&templatev1.TemplateInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name: c.namespace,
				},
				Spec: templatev1.TemplateInstanceSpec{
					Template: *template,
				},
			})
		Expect(err).NotTo(HaveOccurred())

		// Poll for the deployment status
		By(fmt.Sprintf("waiting for the template instance to turn ready (%v)", time.Now()))
		err = wait.PollImmediate(2*time.Second, 5*time.Minute, c.templateInstanceIsReady)
		Expect(err).NotTo(HaveOccurred())

		// Pull the route ingress from the namespace
		By(fmt.Sprintf("getting the route (%v)", time.Now()))
		route, err := c.rc.Routes(c.namespace).Get(TPL, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(len(route.Status.Ingress)).To(Equal(1))
		host := route.Status.Ingress[0].Host
		url := fmt.Sprintf("http://%s", host)

		prevCounter := 0
		currCounter := 0
		httpc := &http.Client{
			Timeout: 10 * time.Second,
		}
		By(fmt.Sprintf("hitting the route 3 times to check the counter (%v)", time.Now()))
		// Hit the ingress 3 times, each time the hit counter should increment
		for i := 0; i < 3; i++ {
			resp, err := httpc.Get(url)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()
			Expect(resp.StatusCode).Should(Equal(200))
			contents, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			matches := regex.FindStringSubmatch(string(contents))
			Expect(matches).NotTo(BeNil())
			currCounter, err = strconv.Atoi(matches[1])
			Expect(err).NotTo(HaveOccurred())
			Expect(currCounter).Should(BeNumerically(">", prevCounter))
			prevCounter = currCounter
			time.Sleep(time.Second)
		}
	})

	It("should not crud infra resources", func() {
		// attempt to read secrets
		_, err := c.kc.CoreV1().Secrets("default").List(metav1.ListOptions{})
		Expect(kerrors.IsForbidden(err)).To(Equal(true))

		// attempt to list pods
		_, err = c.kc.CoreV1().Pods("default").List(metav1.ListOptions{})
		Expect(kerrors.IsForbidden(err)).To(Equal(true))

		// attempt to fetch pod by name
		_, err = c.kc.CoreV1().Pods("kube-system").Get("api-master-000000", metav1.GetOptions{})
		Expect(kerrors.IsForbidden(err)).To(Equal(true))

		// attempt to escalate privileges
		_, err = c.kc.RbacV1().ClusterRoleBindings().Create(&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-escalate-cluster-admin",
			},
			Subjects: []rbacv1.Subject{
				{
					Kind: "User",
					Name: "enduser",
				},
			},
			RoleRef: rbacv1.RoleRef{
				Name: "cluster-admin",
				Kind: "ClusterRole",
			},
		})
		Expect(kerrors.IsForbidden(err)).To(Equal(true))

		// attempt to delete clusterrolebindings
		err = c.kc.RbacV1().ClusterRoleBindings().Delete("cluster-admin", &metav1.DeleteOptions{})
		Expect(kerrors.IsForbidden(err)).To(Equal(true))

		// attempt to delete clusterrole
		err = c.kc.RbacV1().ClusterRoles().Delete("cluster-admin", &metav1.DeleteOptions{})
		Expect(kerrors.IsForbidden(err)).To(Equal(true))

		// attempt to fetch pod logs
		req := c.kc.CoreV1().Pods("kube-system").GetLogs("sync-master-000000", &v1.PodLogOptions{})
		result := req.Do()
		fmt.Println(result.Error().Error())
		Expect(result.Error().Error()).To(ContainSubstring("pods \"sync-master-000000\" is forbidden: User \"enduser\" cannot get pods/log in the namespace \"kube-system\""))
	})
})
