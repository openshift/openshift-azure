package checks

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"syscall"
	"time"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/log"
	"github.com/openshift/openshift-azure/pkg/upgrade"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

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

func WaitForNodes(ctx context.Context, cs *api.OpenShiftManagedCluster, kc *kubernetes.Clientset) error {
	// TODO: this function should not be here.  It is a layering violation.
	// Remove after PP day 1, see issue #390.  There should be a plugin
	// CreateOrUpdate() method and it should ensure readiness on all nodes
	// before returning.  Currently plugin Update() does this and there is no
	// plugin Create().  This code should not run in the healthcheck.  With this
	// code here, on create we wait for readiness, and on upgrade we wait for
	// readiness twice.

	config := auth.NewClientCredentialsConfig(ctx.Value(api.ContextKeyClientID).(string), ctx.Value(api.ContextKeyClientSecret).(string), ctx.Value(api.ContextKeyTenantID).(string))
	authorizer, err := config.Authorizer()
	if err != nil {
		return err
	}
	vmc := compute.NewVirtualMachineScaleSetVMsClient(cs.Properties.AzProfile.SubscriptionID)
	vmc.Authorizer = authorizer

	for _, role := range []api.AgentPoolProfileRole{api.AgentPoolProfileRoleMaster, api.AgentPoolProfileRoleInfra, api.AgentPoolProfileRoleCompute} {
		vms, err := upgrade.ListVMs(ctx, cs, vmc, role)
		if err != nil {
			return err
		}
		for _, vm := range vms {
			log.Infof("waiting for %s to be ready", *vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
			err = upgrade.WaitForReady(ctx, cs, role, *vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// WaitForInfraServices verifies daemonsets, statefulsets
func WaitForInfraServices(ctx context.Context, kc *kubernetes.Clientset) error {
	for _, app := range daemonsetWhitelist {
		log.Infof("checking daemonset %s/%s", app.Namespace, app.Name)

		wait.PollUntil(2*time.Second, func() (bool, error) {
			ds, err := kc.AppsV1().DaemonSets(app.Namespace).Get(app.Name, metav1.GetOptions{})
			switch {
			case kerrors.IsNotFound(err):
				return false, nil
			case err == nil:
				return ds.Status.DesiredNumberScheduled == ds.Status.CurrentNumberScheduled &&
					ds.Status.DesiredNumberScheduled == ds.Status.NumberReady &&
					ds.Status.DesiredNumberScheduled == ds.Status.UpdatedNumberScheduled &&
					ds.Generation == ds.Status.ObservedGeneration, nil
			default:
				return false, err
			}
		}, ctx.Done())
	}

	for _, app := range statefulsetWhitelist {
		log.Infof("checking statefulset %s/%s", app.Namespace, app.Name)

		wait.PollUntil(2*time.Second, func() (bool, error) {
			ss, err := kc.AppsV1().StatefulSets(app.Namespace).Get(app.Name, metav1.GetOptions{})
			switch {
			case kerrors.IsNotFound(err):
				return false, nil
			case err == nil:
				specReplicas := int32(1)
				if ss.Spec.Replicas != nil {
					specReplicas = *ss.Spec.Replicas
				}
				return specReplicas == ss.Status.Replicas &&
					specReplicas == ss.Status.ReadyReplicas &&
					specReplicas == ss.Status.CurrentReplicas &&
					ss.Generation == ss.Status.ObservedGeneration, nil
			default:
				return false, err
			}
		}, ctx.Done())
	}

	for _, app := range deploymentWhitelist {
		log.Infof("checking deployment %s/%s", app.Namespace, app.Name)

		wait.PollUntil(2*time.Second, func() (bool, error) {
			d, err := kc.AppsV1().Deployments(app.Namespace).Get(app.Name, metav1.GetOptions{})
			switch {
			case kerrors.IsNotFound(err):
				return false, nil
			case err == nil:
				specReplicas := int32(1)
				if d.Spec.Replicas != nil {
					specReplicas = *d.Spec.Replicas
				}

				return specReplicas == d.Status.Replicas &&
					specReplicas == d.Status.ReadyReplicas &&
					specReplicas == d.Status.AvailableReplicas &&
					specReplicas == d.Status.UpdatedReplicas &&
					d.Generation == d.Status.ObservedGeneration, nil
			default:
				return false, err
			}
		}, ctx.Done())
	}

	return nil
}
