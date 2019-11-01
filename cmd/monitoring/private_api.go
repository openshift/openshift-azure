package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/fakerp"
	"github.com/openshift/openshift-azure/pkg/util/roundtrippers"
)

type privateAPI struct {
	log                     *logrus.Entry
	managementResourceGroup string
	location                string
}

func newPrivateAPI(ctx context.Context, log *logrus.Entry, oc *api.OpenShiftManagedCluster) (monitor, error) {
	proxyEnv := fmt.Sprintf("PROXYURL_%s", strings.ToUpper(oc.Location))
	proxyurl := os.Getenv(proxyEnv)
	if proxyurl == "" {
		return nil, fmt.Errorf("can not get env[%s]", proxyEnv)
	}
	m := privateAPI{
		log:                     log,
		location:                oc.Location,
		managementResourceGroup: fmt.Sprintf("management-%s", oc.Location),
	}
	fakerp.ConfigureProxyDialer()

	return &m, nil
}

func (m *privateAPI) name() string {
	return "Private API"
}

func (m *privateAPI) getDialContext() func(ctx context.Context, network, address string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		return roundtrippers.PrivateEndpointDialHook(m.location)(network, address)
	}
}

func (m *privateAPI) getHostnames(ctx context.Context, oc *api.OpenShiftManagedCluster) (hostnames []string, err error) {
	m.log.Debug("waiting for private link ip address")
	err = wait.PollImmediateInfinite(10*time.Second, func() (bool, error) {
		pIP, err := fakerp.GetPrivateEndpointIP(ctx, m.log, oc.Properties.AzProfile.SubscriptionID, m.managementResourceGroup, oc.Properties.AzProfile.ResourceGroup)
		if err != nil {
			m.log.Error(err)
			return false, nil
		}
		if pIP == nil {
			return false, nil
		}
		hostnames = append(hostnames, *pIP)
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return hostnames, nil
}
