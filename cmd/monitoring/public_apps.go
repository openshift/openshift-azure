package main

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/openshift/openshift-azure/pkg/api"
)

type publicApps struct {
	log *logrus.Entry
}

func newPublicApps(ctx context.Context, log *logrus.Entry) (monitor, error) {
	m := publicApps{
		log: log,
	}
	return &m, nil
}

func (m *publicApps) name() string {
	return "Public Apps"
}

func (m *publicApps) getDialContext() func(ctx context.Context, network, address string) (net.Conn, error) {
	return (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
		LocalAddr: &net.TCPAddr{},
	}).DialContext
}

func (m *publicApps) getHostnames(ctx context.Context, oc *api.OpenShiftManagedCluster) (hostnames []string, err error) {
	err = wait.PollImmediateInfinite(time.Second, func() (bool, error) {
		hostnames = append(hostnames, fmt.Sprintf("canary-openshift-azure-monitoring.%s", oc.Properties.RouterProfiles[0].PublicSubdomain))
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return hostnames, nil
}
