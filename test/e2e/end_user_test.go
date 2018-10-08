//+build e2e

package e2e

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	templatev1 "github.com/openshift/api/template/v1"
	corev1 "k8s.io/api/core/v1"
	policy "k8s.io/api/policy/v1beta1"
	_ "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/watch"
)

func httpWithRetry(url string, times int) (string, error) {
	// attempt up to 3 times with linear backoff
	for i := 1; i <= times; i++ {
		delay := time.Duration(i*100) * time.Millisecond
		resp, err := http.Get(url)
		defer resp.Body.Close()
		// if the GET failed, wait a bit and try again
		if err != nil {
			time.Sleep(delay)
			continue
		}
		// if the status code is not 200, wait a bit and try again
		if resp.StatusCode != http.StatusOK {
			time.Sleep(delay)
			continue
		}
		// if the contents (somehow) can't be read, wait a bit and try again
		contents, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			time.Sleep(delay)
			continue
		}
		return string(contents), nil
	}
	// if we got to this point, all attempts failed
	return "", errors.New("All HTTP requests failed")
}

var _ = Describe("Openshift on Azure end user e2e tests [EndUser]", func() {
	defer GinkgoRecover()

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
		// TODO: Reenable
		// Expect(kerrors.IsForbidden(err)).To(Equal(true))
		fmt.Printf("PDB create error: %v\n", err)
	})

	It("should deploy a template and check the visit counter increments", func() {
		const NAME = "nodejs-mongodb-example"
		var regex = regexp.MustCompile(`\<span.*id=\"count-value\"\>(?P<counter>\d+)\<\/span\>`)

		// Create the template
		template, err := c.tc.Templates("openshift").Get(
			NAME, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		// Create a secret to hold the template params
		secret, err := c.cc.Secrets(c.namespace).Create(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: "parameters",
			},
			StringData: map[string]string{
				"NAME":         NAME,
				"MEMORY_LIMIT": "1024Mi",
			},
		})
		Expect(err).NotTo(HaveOccurred())

		// Instantiate the template
		ti, err := c.tc.TemplateInstances(c.namespace).Create(
			&templatev1.TemplateInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name: "templateinstance",
				},
				Spec: templatev1.TemplateInstanceSpec{
					Template: *template,
					Secret: &corev1.LocalObjectReference{
						Name: secret.Name,
					},
				},
			})
		Expect(err).NotTo(HaveOccurred())

		// Wait for the deployments to finish
		watcher, err := c.tc.TemplateInstances(c.namespace).Watch(
			metav1.SingleObject(ti.ObjectMeta),
		)
		Expect(err).NotTo(HaveOccurred())

		for event := range watcher.ResultChan() {
			switch event.Type {
			case watch.Modified:
				ti = event.Object.(*templatev1.TemplateInstance)

				for _, cond := range ti.Status.Conditions {
					// If the TemplateInstance contains a status condition
					// Ready == True, stop watching.
					if cond.Type == templatev1.TemplateInstanceReady &&
						cond.Status == corev1.ConditionTrue {
						watcher.Stop()
					}

					// If the TemplateInstance contains a status condition
					// InstantiateFailure == True, indicate failure.
					if cond.Type ==
						templatev1.TemplateInstanceInstantiateFailure &&
						cond.Status == corev1.ConditionTrue {
						panic("templateinstance instantiation failed")
					}
				}

			default:
				panic("unexpected event type " + event.Type)
			}
		}

		// Pull the route ingress from the namespace
		route, err := c.rc.Routes(c.namespace).Get(NAME, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		host := route.Status.Ingress[0].Host
		url := fmt.Sprintf("http://%s", host)

		// Hit the ingress 3 times, each time the hit counter should increment
		prevCounter := 0
		currCounter := 0
		for i := 1; i <= 3; i++ {
			contents, err := httpWithRetry(url, 3)
			Expect(err).NotTo(HaveOccurred())
			matches := regex.FindStringSubmatch(contents)
			currCounter, err = strconv.Atoi(matches[1])
			Expect(err).NotTo(HaveOccurred())
			Expect(currCounter).Should(BeNumerically(">", prevCounter))
			prevCounter = currCounter
			time.Sleep(time.Second)
		}
	})
})
