package checks

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"syscall"
	"time"

	acsapi "github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/log"

	"k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

// WaitForHTTPStatusOk poll until URL returns 200
func WaitForHTTPStatusOk(ctx context.Context, transport http.RoundTripper, urltocheck string) error {
	cli := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	req, err := http.NewRequest("GET", urltocheck, nil)
	if err != nil {
		return err
	}
	return wait.PollUntil(time.Second, func() (bool, error) {
		resp, err := cli.Do(req)
		if err, ok := err.(*url.Error); ok {
			if err, ok := err.Err.(*net.OpError); ok {
				if err, ok := err.Err.(*os.SyscallError); ok {
					if err.Err == syscall.ENETUNREACH {
						return false, nil
					}
				}
			}
			if err.Timeout() || err.Err == io.EOF || err.Err == io.ErrUnexpectedEOF {
				return false, nil
			}
		}
		if err == io.EOF {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		return resp != nil && resp.StatusCode == http.StatusOK, nil
	}, ctx.Done())
}

type Node struct {
	Ready bool
	Name  string
}

func GenerateNodeMap(cs *acsapi.OpenShiftManagedCluster) map[string][]Node {
	nodes := make(map[string][]Node)
	for _, app := range cs.Properties.AgentPoolProfiles {
		nodes[string(app.Role)] = []Node{}
		for i := 0; i < app.Count; i++ {
			n := Node{Name: fmt.Sprintf("%s-%06d", app.Role, i)}
			nodes[string(app.Role)] = append(nodes[string(app.Role)], n)
		}
	}
	return nodes
}

func isNodeReady(nodeName string, kc *kubernetes.Clientset) (bool, error) {
	n, err := kc.CoreV1().Nodes().Get(nodeName, metav1.GetOptions{})
	if err == nil {
		for _, cond := range n.Status.Conditions {
			if cond.Type == v1.NodeReady {
				return true, nil
			}
		}
	}
	if err != nil && !kerrors.IsNotFound(err) && !kerrors.IsTimeout(err) {
		return false, err
	}

	return false, nil
}

func WaitForNodeReady(ctx context.Context, nodes map[string][]Node, kc *kubernetes.Clientset) error {
	for nodeType, ntNodes := range nodes { // walk node types, e.g. master, compute, infra
		for idx, node := range ntNodes { // walk each node in type, e.g. master00000[1,2,3]
			for {
				ready, err := isNodeReady(node.Name, kc)
				if err != nil {
					return err
				}
				if ready {
					nodes[nodeType][idx].Ready = true
					break
				}
				// check NodeReady for each node to ensure readiness
				select {
				case <-time.After(2 * time.Second):
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}
	}
	return nil
}

// map to hold the namespace and the deployments, statefulsets, daemonset
var deploymentWhitelist = []struct {
	Name      string
	Namespace string
}{
	{
		Name:      "docker-registry",
		Namespace: "default",
	},
	{
		Name:      "router",
		Namespace: "default",
	},
	{
		Name:      "registry-console",
		Namespace: "default",
	},
	{
		Name:      "apiserver",
		Namespace: "kube-service-catalog",
	},
	{
		Name:      "controller-manager",
		Namespace: "kube-service-catalog",
	},
	{
		Name:      "bootstrap-autoapprover",
		Namespace: "openshift-infra",
	},
	{
		Name:      "asb",
		Namespace: "openshift-ansible-service-broker",
	},
	{
		Name:      "apiserver",
		Namespace: "openshift-template-service-broker",
	},
	{
		Name:      "bootstrap-autoapprover",
		Namespace: "openshift-infra",
	},
	{
		Name:      "webconsole",
		Namespace: "openshift-web-console",
	},
}
var staticWhitelist = []struct {
	Name      string
	Namespace string
	NodeType  string
}{
	{
		Name:      "api",
		Namespace: "kube-system",
		NodeType:  "master",
	},
	{
		Name:      "controllers",
		Namespace: "kube-system",
		NodeType:  "master",
	},
	{
		Name:      "etcd",
		Namespace: "kube-system",
		NodeType:  "master",
	},
	{
		Name:      "logbridge",
		Namespace: "kube-system",
		NodeType:  "all",
	},
	{
		Name:      "sync-master-000000",
		Namespace: "kube-system",
	},
}
var daemonsetWhitelist = []struct {
	Name      string
	Namespace string
}{
	{
		Name:      "prometheus-node-exporter",
		Namespace: "openshift-metrics",
	},
	{
		Name:      "sync",
		Namespace: "openshift-node",
	},
	{
		Name:      "ovs",
		Namespace: "openshift-sdn",
	},
	{
		Name:      "sdn",
		Namespace: "openshift-sdn",
	},
}
var statefulsetWhitelist = []struct {
	Name      string
	Namespace string
}{
	{
		Name:      "prometheus",
		Namespace: "openshift-metrics",
	},
}

// WaitForInfraServices verify daemonsets, statefulsets
func WaitForInfraServices(ctx context.Context, kc *kubernetes.Clientset, nodes map[string][]Node) error {
	log.Info("checking infrastructure service health")

	for idx, app := range daemonsetWhitelist {
		log.Info(fmt.Sprintf("checking   daemonset[%d]: [%s] %s", idx, app.Namespace, app.Name))
		for {
			ds, err := kc.AppsV1().DaemonSets(app.Namespace).Get(app.Name, metav1.GetOptions{})
			// wait for the ds to come online
			if err == nil {
				if ds.Status.NumberMisscheduled > 0 {
					return fmt.Errorf("Daemonset[%v] in Namespace[%v] has missscheduled[%v]", ds.Name, ds.Namespace, ds.Status.NumberMisscheduled)
				}

				if ds.Status.DesiredNumberScheduled == ds.Status.CurrentNumberScheduled &&
					ds.Status.DesiredNumberScheduled == ds.Status.NumberReady &&
					ds.Status.DesiredNumberScheduled == ds.Status.UpdatedNumberScheduled &&
					ds.Generation == ds.Status.ObservedGeneration {
					break
				}
			}
			if err != nil && !kerrors.IsNotFound(err) {
				return err
			}
			select {
			case <-time.After(2 * time.Second):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	// Statefulsets
	for idx, app := range statefulsetWhitelist {
		log.Infof("checking statefulset[%d]: [%s] %s", idx, app.Namespace, app.Name)
		for {
			ss, err := kc.AppsV1().StatefulSets(app.Namespace).Get(app.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			if ss.Status.Replicas == ss.Status.ReadyReplicas &&
				ss.Status.ReadyReplicas == ss.Status.CurrentReplicas &&
				ss.Spec.Replicas != nil &&
				*ss.Spec.Replicas == ss.Status.Replicas &&
				ss.Generation == ss.Status.ObservedGeneration {
				break
			}
			select {
			case <-time.After(2 * time.Second):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	// Deployments
	for idx, app := range deploymentWhitelist {
		log.Infof("checking  deployment[%d]: [%s] %s", idx, app.Namespace, app.Name)
		for {
			d, err := kc.AppsV1().Deployments(app.Namespace).Get(app.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			if d.Status.Replicas == d.Status.ReadyReplicas &&
				d.Status.ReadyReplicas == d.Status.AvailableReplicas &&
				d.Spec.Replicas != nil &&
				*d.Spec.Replicas == d.Status.Replicas &&
				d.Status.Replicas == d.Status.UpdatedReplicas &&
				d.Generation == d.Status.ObservedGeneration {
				break
			}
			select {
			case <-time.After(2 * time.Second):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	// generate a list of the expected static pods
	pods := make(map[string]struct{ Namespace string })
	for _, app := range staticWhitelist {
		switch app.NodeType {
		case "all":
			for _, nodes := range nodes {
				for _, node := range nodes {
					pods[fmt.Sprintf("%s-%s", app.Name, node.Name)] = struct{ Namespace string }{Namespace: app.Namespace}
				}
			}
		default:
			for _, node := range nodes[app.NodeType] {
				pods[fmt.Sprintf("%s-%s", app.Name, node.Name)] = struct{ Namespace string }{Namespace: app.Namespace}
			}
		}
	}

	// static pods
	for podname, podinfo := range pods {
		log.Infof("checking         static: [%s] %s", podname, podinfo.Namespace)
		for {
			// get a list of all pods, in the namespace in which we want the static pod to show up.
			pod, err := kc.CoreV1().Pods(podinfo.Namespace).Get(podname, metav1.GetOptions{})
			if err != nil {
				return err
			}
			if len(pod.Status.ContainerStatuses) > 0 && pod.Status.ContainerStatuses[0].Ready {
				break
			}

			select {
			case <-time.After(2 * time.Second):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	log.Info("done checking infrastructure service health")
	return nil
}
