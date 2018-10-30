//+build e2e

package e2e

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	policy "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"

	waitutil "github.com/openshift/openshift-azure/pkg/util/wait"
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

	It("should deploy a template and ensure a given text is in the contents", func() {
		tpl := "nginx-example"
		By(fmt.Sprintf("instantiating the template and getting the route (%v)", time.Now()))
		// instantiate the template
		err := c.instantiateTemplate(tpl)
		Expect(err).NotTo(HaveOccurred())
		route, err := c.rc.Routes(c.namespace).Get(tpl, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		// make sure only 1 ingress point is returned
		Expect(len(route.Status.Ingress)).To(Equal(1))
		host := route.Status.Ingress[0].Host
		url := fmt.Sprintf("http://%s", host)

		// Curl the endpoint and search for a string
		httpc := &http.Client{
			Timeout: 10 * time.Second,
		}
		By(fmt.Sprintf("hitting the route and checking the contents (%v)", time.Now()))
		resp, err := httpc.Get(url)
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()
		Expect(resp.StatusCode).Should(Equal(http.StatusOK))
		contents, err := ioutil.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(contents)).Should(ContainSubstring("Welcome to your static nginx application on OpenShift"))
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

	It("should deploy a template with persistent storage and test failure modes", func() {
		tpl := "django-psql-persistent"
		By(fmt.Sprintf("instantiating the template and getting the route (%v)", time.Now()))
		// instantiate the template
		err := c.instantiateTemplate(tpl)
		Expect(err).NotTo(HaveOccurred())
		// Pull the route ingress from the namespace
		route, err := c.rc.Routes(c.namespace).Get(tpl, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		// make sure only 1 ingress point is returned
		Expect(len(route.Status.Ingress)).To(Equal(1))
		host := route.Status.Ingress[0].Host
		url := fmt.Sprintf("http://%s", host)

		// Curl the endpoint and search for a string
		httpc := &http.Client{
			Timeout: 10 * time.Second,
		}
		regex := regexp.MustCompile(`Page views:\s*(\d+)`)
		By(fmt.Sprintf("hitting the route 3 times, expecting counter to increment (%v)", time.Now()))
		var prevCounter, currCounter int
		for i := 0; i < 3; i++ {
			resp, err := httpc.Get(url)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()
			Expect(resp.StatusCode).Should(Equal(http.StatusOK))

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

		// Find the database deploymentconfig and scale to 0
		dcName := "postgresql"
		for i := range []int{0, 1} {
			By(fmt.Sprintf("searching for the database deploymentconfig (%v)", time.Now()))
			dbDeploymentConfig, err := c.ac.DeploymentConfigs(c.namespace).Get(dcName, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(dbDeploymentConfig).NotTo(BeNil())
			By(fmt.Sprintf("scaling the database deploymentconfig to %d (%v)", i, time.Now()))
			dbDeploymentConfig.Spec.Replicas = int32(i)
			updated, err := c.ac.DeploymentConfigs(c.namespace).Update(dbDeploymentConfig)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated).NotTo(BeNil())
			By(fmt.Sprintf("waiting for database deploymentconfig to reflect %d replicas (%v)", i, time.Now()))
			waitErr := wait.PollImmediate(2*time.Second, 10*time.Minute, func() (bool, error) {
				dc, err := c.ac.DeploymentConfigs(c.namespace).Get(dcName, metav1.GetOptions{})
				i32 := int32(i)
				switch {
				case err == nil:
					return i32 == dc.Status.Replicas &&
						i32 == dc.Status.ReadyReplicas &&
						i32 == dc.Status.AvailableReplicas &&
						i32 == dc.Status.UpdatedReplicas &&
						dc.Generation == dc.Status.ObservedGeneration, nil
				default:
					return false, err
				}
			})
			Expect(waitErr).NotTo(HaveOccurred())
		}

		// wait for the ingress to return 200 in case healthcheck failed when database got recreated
		By(fmt.Sprintf("making sure the ingress is healthy (%v)", time.Now()))
		var rt http.RoundTripper
		waitErr := waitutil.ForHTTPStatusOk(context.Background(), rt, url)
		Expect(waitErr).NotTo(HaveOccurred())

		By(fmt.Sprintf("hitting the route again, expecting counter to increment from last=%d (%v)", currCounter, time.Now()))
		for i := 0; i < 3; i++ {
			resp, err := httpc.Get(url)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()
			Expect(resp.StatusCode).Should(Equal(http.StatusOK))

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
})
