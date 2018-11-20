// +build e2e

package enduser

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
	policy "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"

	waitutil "github.com/openshift/openshift-azure/pkg/util/wait"
	"github.com/openshift/openshift-azure/test/util/client/kubernetes"
)

func CheckPdbMutationsDisallowed(c *kubernetes.Client) {
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

	err = c.CreatePodDisruptionBudget(pdb)
	Expect(kerrors.IsForbidden(err)).To(Equal(true))
}

func CheckCanDeployTemplate(c *kubernetes.Client) {
	tpl := "nginx-example"
	By(fmt.Sprintf("instantiating the template and getting the route (%v)", time.Now()))
	// instantiate the template
	err := c.InstantiateTemplate(tpl)
	Expect(err).NotTo(HaveOccurred())
	route, err := c.GetRoute(tpl, nil)
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
}

func CheckCrudOnInfraDisallowed(c *kubernetes.Client) {
	// attempt to read secrets
	_, err := c.ListSecrets("default", nil)
	Expect(kerrors.IsForbidden(err)).To(Equal(true))

	// attempt to list pods
	_, err = c.ListPods("default", nil)
	Expect(kerrors.IsForbidden(err)).To(Equal(true))

	// attempt to fetch pod by name
	_, err = c.GetPodByName("kube-system", "api-master-000000", nil)
	Expect(kerrors.IsForbidden(err)).To(Equal(true))

	// attempt to escalate privileges
	newBinding := &rbacv1.ClusterRoleBinding{
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
	}
	_, err = c.CreateClusterRoleBinding(newBinding)
	Expect(kerrors.IsForbidden(err)).To(Equal(true))

	// attempt to delete clusterrolebindings
	err = c.DeleteClusterRoleBinding("cluster-admin", nil)
	Expect(kerrors.IsForbidden(err)).To(Equal(true))

	// attempt to delete clusterrole
	err = c.DeleteClusterRole("cluster-admin", nil)
	Expect(kerrors.IsForbidden(err)).To(Equal(true))

	// attempt to fetch pod logs
	req := c.GetPodLogs("kube-system", "sync-master-000000", nil)
	result := req.Do()
	fmt.Println(result.Error().Error())
	Expect(result.Error().Error()).To(ContainSubstring("pods \"sync-master-000000\" is forbidden: User \"enduser\" cannot get pods/log in the namespace \"kube-system\""))
}

func CheckCanDeployTemplateWithPV(c *kubernetes.Client) {
	prevCounter := 0

	loopHTTPGet := func(url string, regex *regexp.Regexp, times int) error {
		httpc := &http.Client{
			Timeout: 10 * time.Second,
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

			currcounter, err := strconv.Atoi(matches[1])
			if err != nil {
				return err
			}
			if currcounter <= prevCounter {
				return fmt.Errorf("visit counter didn't increment: %d should be > than %d", currcounter, prevCounter)
			}
			prevCounter = currcounter
		}
		return nil
	}

	// instantiate the template
	tpl := "django-psql-persistent"
	By(fmt.Sprintf("instantiating the template and getting the route (%v)", time.Now()))
	err := c.InstantiateTemplate(tpl)
	Expect(err).NotTo(HaveOccurred())

	// Pull the route ingress from the namespace and make sure only 1 ingress point is returned
	route, err := c.GetRoute(tpl, nil)
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
		dc, err := c.GetDeploymentConfig(dcName, nil)
		Expect(err).NotTo(HaveOccurred())

		By(fmt.Sprintf("scaling the database deploymentconfig to %d (%v)", i, time.Now()))
		dc.Spec.Replicas = int32(i)
		_, err = c.UpdateDeploymentConfig(dc)
		Expect(err).NotTo(HaveOccurred())

		By(fmt.Sprintf("waiting for database deploymentconfig to reflect %d replicas (%v)", i, time.Now()))
		waitErr := wait.PollImmediate(2*time.Second, 10*time.Minute, c.DeploymentConfigIsReady(dcName, i))
		Expect(waitErr).NotTo(HaveOccurred())
	}

	// wait for the ingress to return 200 in case healthcheck failed when database got recreated
	By(fmt.Sprintf("making sure the ingress is healthy (%v)", time.Now()))
	waitErr := waitutil.ForHTTPStatusOk(context.Background(), nil, url)
	Expect(waitErr).NotTo(HaveOccurred())

	// hit it again, will hit 3 times as specified initially
	By(fmt.Sprintf("hitting the route again, expecting counter to increment from last (%v)", time.Now()))
	err = loopHTTPGet(url, regex, 3)
	Expect(err).NotTo(HaveOccurred())
}
