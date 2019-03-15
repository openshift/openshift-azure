package standard

import (
	"context"
	"fmt"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/openshift/openshift-azure/pkg/util/ready"
	waitutil "github.com/openshift/openshift-azure/pkg/util/wait"
)

const (
	statelessApp = "nginx-example"
	statefulApp  = "django-psql-persistent"
)

func (sc *SanityChecker) createStatefulApp(ctx context.Context, namespace string) error {
	sc.log.Debugf("instantiating %s template", statefulApp)
	err := sc.Client.EndUser.InstantiateTemplate(statefulApp, namespace)
	if err != nil {
		return err
	}
	return nil
}

func (sc *SanityChecker) createStatelessApp(ctx context.Context, namespace string) error {
	sc.log.Debugf("instantiating %s template", statelessApp)
	err := sc.Client.EndUser.InstantiateTemplate(statelessApp, namespace)
	if err != nil {
		return err
	}
	return nil
}

func (sc *SanityChecker) validateStatefulApp(ctx context.Context, namespace string) error {
	prevCounter := 0
	loopHTTPGet := func(url string, regex *regexp.Regexp, times int) error {
		timeout, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		for i := 0; i < times; i++ {
			resp, err := waitutil.ForHTTPStatusOk(timeout, sc.log, nil, url)
			if err != nil {
				return err
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
	// Pull the route ingress from the namespace
	route, err := sc.Client.EndUser.RouteV1.Routes(namespace).Get(statefulApp, metav1.GetOptions{})
	if err != nil {
		return err
	}
	// make sure only 1 ingress point is returned
	length := len(route.Status.Ingress)
	if length != 1 {
		return fmt.Errorf("expected only 1 ingress point, got %d", length)
	}

	// hit the ingress 3 times before killing the DB
	host := route.Status.Ingress[0].Host
	url := fmt.Sprintf("http://%s", host)
	regex := regexp.MustCompile(`Page views:\s*(\d+)`)
	sc.log.Debugf("hitting the route 3 times, expecting counter to increment")
	err = loopHTTPGet(url, regex, 3)
	if err != nil {
		return err
	}

	// Find the database deploymentconfig and scale down to 0, then back up to 1
	dcName := "postgresql"
	for _, i := range []int32{0, 1} {
		sc.log.Debugf("searching for the database deploymentconfig")
		dc, err := sc.Client.EndUser.OAppsV1.DeploymentConfigs(namespace).Get(dcName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		sc.log.Debugf("scaling the database deploymentconfig to %d", i)
		dc.Spec.Replicas = int32(i)
		_, err = sc.Client.EndUser.OAppsV1.DeploymentConfigs(namespace).Update(dc)
		if err != nil {
			return err
		}
		sc.log.Debugf("waiting for database deploymentconfig to reflect %d replicas", i)
		waitErr := wait.PollImmediate(2*time.Second, 10*time.Minute, ready.CheckDeploymentConfigIsReady(sc.Client.EndUser.OAppsV1.DeploymentConfigs(namespace), dcName))
		if waitErr != nil {
			return waitErr
		}
	}
	// hit it again, will hit 3 times as specified initially
	sc.log.Debugf("hitting the route again, expecting counter to increment from last")
	err = loopHTTPGet(url, regex, 3)
	if err != nil {
		return err
	}
	return nil
}

func (sc *SanityChecker) validateStatelessApp(ctx context.Context, namespace string) error {
	route, err := sc.Client.EndUser.RouteV1.Routes(namespace).Get(statelessApp, metav1.GetOptions{})
	if err != nil {
		return err
	}
	// make sure only 1 ingress point is returned
	length := len(route.Status.Ingress)
	if length != 1 {
		return fmt.Errorf("expected only 1 ingress point, got %d", length)
	}
	host := route.Status.Ingress[0].Host
	url := fmt.Sprintf("http://%s", host)

	// Curl the endpoint and search for a string
	sc.log.Debugf("hitting the route %s and verifying the response", url)
	timeout, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	resp, err := waitutil.ForHTTPStatusOk(timeout, sc.log, nil, url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	expected := "Welcome to your static nginx application on OpenShift"
	if !strings.Contains(string(contents), expected) {
		return fmt.Errorf("did not find expected string in response")
	}
	return nil
}
