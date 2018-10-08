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
	routev1client "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	templatev1client "github.com/openshift/client-go/template/clientset/versioned/typed/template/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
)

const GOOD_TEMPLATE = "nodejs-mongodb-example"
const BAD_TEMPLATE = "non-existent"
const REGEX = `\<span.*id=\"count-value\"\>(?P<counter>\d+)\<\/span\>`

var _ = Describe("Openshift on Azure end user e2e tests [SmokeTest]", func() {
	defer GinkgoRecover()

	Context("When the template exists", func() {
		It("should deploy a template and check the visit counter increments", func() {
			// Create the template
			templateclient, err := templatev1client.NewForConfig(c.config)
			Expect(err).NotTo(HaveOccurred())
			template, err := templateclient.Templates("openshift").Get(
				GOOD_TEMPLATE, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Create a secret to hold the template params
			coreclient, err := corev1client.NewForConfig(c.config)
			Expect(err).NotTo(HaveOccurred())
			secret, err := coreclient.Secrets(c.namespace).Create(&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "parameters",
				},
				StringData: map[string]string{
					"NAME":         GOOD_TEMPLATE,
					"MEMORY_LIMIT": "1024Mi",
				},
			})
			Expect(err).NotTo(HaveOccurred())

			// Instantiate the template
			ti, err := templateclient.TemplateInstances(c.namespace).Create(
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
			watcher, err := templateclient.TemplateInstances(c.namespace).Watch(
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

			r := regexp.MustCompile(REGEX)

			// Pull the route ingress from the namespace
			routeclient, err := routev1client.NewForConfig(c.config)
			Expect(err).NotTo(HaveOccurred())
			route, err := routeclient.Routes(c.namespace).Get(GOOD_TEMPLATE, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			host := route.Status.Ingress[0].Host
			url := fmt.Sprintf("http://%s", host)

			// Hit the ingress 3 times, each time the hit counter should increment
			prevCounter := 0
			currCounter := 0
			for i := 1; i <= 3; i++ {
				resp, err := http.Get(url)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()
				contents, err := ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				matches := r.FindStringSubmatch(string(contents))
				currCounter, err = strconv.Atoi(matches[1])
				Expect(err).NotTo(HaveOccurred())
				Expect(currCounter).Should(BeNumerically(">", prevCounter))
				prevCounter = currCounter
				time.Sleep(1)
			}
		})
	})

	Context("When the template doesn't exist", func() {
		It("should error when fetching the template from the openshift namespace", func() {
			templateclient, err := templatev1client.NewForConfig(c.config)
			Expect(err).NotTo(HaveOccurred())
			template, err := templateclient.Templates("openshift").Get(
				BAD_TEMPLATE, metav1.GetOptions{})
			Expect(err).To(HaveOccurred())
			var _ = template
		})
	})
})
