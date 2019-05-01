package sync

import (
	"context"
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	v3 "github.com/openshift/openshift-azure/pkg/sync/v3"
	v4 "github.com/openshift/openshift-azure/pkg/sync/v4"
	v5 "github.com/openshift/openshift-azure/pkg/sync/v5"
)

type Interface interface {
	Sync(ctx context.Context) error
	ReadyHandler(w http.ResponseWriter, r *http.Request)
	PrintDB() error
	Hash() ([]byte, error)
}

func New(log *logrus.Entry, cs *api.OpenShiftManagedCluster, initClients bool) (Interface, error) {
	switch cs.Config.PluginVersion {
	case "v3.2":
		return v3.New(log, cs, initClients)
	case "v4.2", "v4.3":
		return v4.New(log, cs, initClients)
	case "v5.0":
		return v5.New(log, cs, initClients)
	}

	return nil, fmt.Errorf("version %q not found", cs.Config.PluginVersion)
}
