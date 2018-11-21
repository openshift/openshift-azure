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
		timeout, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		err = waitutil.ForHTTPStatusOk(timeout, nil, url)
		Expect(err).NotTo(HaveOccurred())
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
		prevCounter := 0

		loopHTTPGet := func(url string, regex *regexp.Regexp, times int) error {
			httpc := &http.Client{
				Timeout: 10 * time.Second,
			}

			timeout, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			err := waitutil.ForHTTPStatusOk(timeout, nil, url)
			if err != nil {
				return err
			}

			for i := 0; i < times; i++ {
				resp, err := httpc.Get(url)
				if err != nil {
					return err
				}
				defer resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					return fmt.Errorf("unexpected http error returned: %d", resp.StatusCode)
				}

				contents, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					return err
				}
				matches := regex.FindStringSubmatch(string(contents))
				if matches == nil {
					return fmt.Errorf("no matches found for %s", regex)
				}

				currCounter, err := strconv.Atoi(matches[1])
				if err != nil {
					return err
				}
				if currCounter <= prevCounter {
					return fmt.Errorf("visit counter didn't increment: %d should be > than %d", currCounter, prevCounter)
				}
				prevCounter = currCounter
			}
			return nil
		}

		// instantiate the template
		tpl := "django-psql-persistent"
		By(fmt.Sprintf("instantiating the template and getting the route (%v)", time.Now()))
		err := c.instantiateTemplate(tpl)
		Expect(err).NotTo(HaveOccurred())

		// Pull the route ingress from the namespace and make sure only 1 ingress point is returned
		route, err := c.rc.Routes(c.namespace).Get(tpl, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(len(route.Status.Ingress)).To(Equal(1))

		// hit the ingress 3 times before killing the DB
		host := route.Status.Ingress[0].Host
		url := fmt.Sprintf("http://%s", host)
		regex := regexp.MustCompile(`Page views:\s*(\d+)`)
		By(fmt.Sprintf("hitting the route 3 times, expecting counter to increment (%v)", time.Now()))
		err = loopHTTPGet(url, regex, 3)
		Expect(err).NotTo(HaveOccurred())

		// Find the database deploymentconfig and scale down to 0, then back up to 1
		dcName := "postgresql"
		for _, i := range []int32{0, 1} {
			By(fmt.Sprintf("searching for the database deploymentconfig (%v)", time.Now()))
			dc, err := c.ac.DeploymentConfigs(c.namespace).Get(dcName, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			By(fmt.Sprintf("scaling the database deploymentconfig to %d (%v)", i, time.Now()))
			dc.Spec.Replicas = int32(i)
			_, err = c.ac.DeploymentConfigs(c.namespace).Update(dc)
			Expect(err).NotTo(HaveOccurred())

			By(fmt.Sprintf("waiting for database deploymentconfig to reflect %d replicas (%v)", i, time.Now()))
			waitErr := wait.PollImmediate(2*time.Second, 10*time.Minute, c.deploymentConfigIsReady(dcName, i))
			Expect(waitErr).NotTo(HaveOccurred())
		}

		// hit it again, will hit 3 times as specified initially
		By(fmt.Sprintf("hitting the route again, expecting counter to increment from last (%v)", time.Now()))
		err = loopHTTPGet(url, regex, 3)
		Expect(err).NotTo(HaveOccurred())
	})
})
