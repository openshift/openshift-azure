package main

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/network"
)

type publicAPI struct {
	log    *logrus.Entry
	pipcli network.PublicIPAddressesClient
}

func newPublicAPI(ctx context.Context, log *logrus.Entry, oc *api.OpenShiftManagedCluster) (monitor, error) {
	authorizer, err := azureclient.GetAuthorizerFromContext(ctx, api.ContextKeyClientAuthorizer)
	if err != nil {
		return nil, err
	}
	m := publicAPI{
		log: log,
	}
	m.pipcli = network.NewPublicIPAddressesClient(ctx, log, oc.Properties.AzProfile.SubscriptionID, authorizer)
	return &m, nil
}

func (m *publicAPI) name() string {
	return "Public API"
}

func (m *publicAPI) getDialContext() func(ctx context.Context, network, address string) (net.Conn, error) {
	return (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
		LocalAddr: &net.TCPAddr{},
	}).DialContext
}

func (m *publicAPI) getHostnames(ctx context.Context, oc *api.OpenShiftManagedCluster) (hostnames []string, err error) {
	// get dedicated routes we want to monitor
	m.log.Debug("waiting for OpenShiftManagedCluster config to be persisted")
	// TODO: once we can read provisioningState from disk file, remove network
	// polls

	// get all external IP's used by VMSS
	m.log.Debug("waiting for ss-masters ip addresses")
	err = wait.PollImmediateInfinite(10*time.Second, func() (bool, error) {
		var ips []string
		for iter, err := m.pipcli.ListVirtualMachineScaleSetPublicIPAddressesComplete(ctx, oc.Properties.AzProfile.ResourceGroup, "ss-master"); iter.NotDone(); err = iter.Next() {
			if err != nil {
				m.log.Error(err)
				return false, nil
			}

			if iter.Value().IPAddress != nil {
				ips = append(ips, *iter.Value().IPAddress)
			}
		}
		if len(ips) == 3 {
			hostnames = append(hostnames, ips...)
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return nil, err
	}

	// get api server hostname
	m.log.Debug("waiting for ip-apiserver server hostname")
	ip, err := m.pipcli.Get(ctx, oc.Properties.AzProfile.ResourceGroup, "ip-apiserver", "")
	if err != nil {
		return nil, err
	}
	hostnames = append(hostnames, fmt.Sprintf("%s.%s.cloudapp.azure.com", *ip.DNSSettings.DomainNameLabel, *ip.Location))

	return hostnames, nil
}
