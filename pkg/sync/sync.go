package sync

import (
	"context"
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	v5 "github.com/openshift/openshift-azure/pkg/sync/v5"
	v6 "github.com/openshift/openshift-azure/pkg/sync/v6"
	v7 "github.com/openshift/openshift-azure/pkg/sync/v7"
	v71 "github.com/openshift/openshift-azure/pkg/sync/v71"
)

type Interface interface {
	Sync(ctx context.Context) error
	ReadyHandler(w http.ResponseWriter, r *http.Request)
	PrintDB() error
	Hash() ([]byte, error)
}

func New(log *logrus.Entry, cs *api.OpenShiftManagedCluster, initClients bool) (Interface, error) {
	switch cs.Config.PluginVersion {
	case "v5.1", "v5.2":
		return v5.New(log, cs, initClients)
	case "v6.0":
		return v6.New(log, cs, initClients)
	case "v7.0":
		return v7.New(log, cs, initClients)
	case "v7.1":
		return v71.New(log, cs, initClients)
	}

	return nil, fmt.Errorf("version %q not found", cs.Config.PluginVersion)
}
