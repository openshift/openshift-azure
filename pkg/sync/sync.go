package sync

import (
	"context"
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	v10 "github.com/openshift/openshift-azure/pkg/sync/v10"
	v11 "github.com/openshift/openshift-azure/pkg/sync/v11"
	v7 "github.com/openshift/openshift-azure/pkg/sync/v7"
	v71 "github.com/openshift/openshift-azure/pkg/sync/v71"
	v9 "github.com/openshift/openshift-azure/pkg/sync/v9"
)

type Interface interface {
	Sync(ctx context.Context) error
	ReadyHandler(w http.ResponseWriter, r *http.Request)
	PrintDB() error
	Hash() ([]byte, error)
}

func New(log *logrus.Entry, cs *api.OpenShiftManagedCluster, initClients bool) (Interface, error) {
	switch cs.Config.PluginVersion {
	case "v7.0":
		return v7.New(log, cs, initClients)
	case "v7.1":
		return v71.New(log, cs, initClients)
	case "v9.0":
		return v9.New(log, cs, initClients)
	case "v10.0":
		return v10.New(log, cs, initClients)
	case "v11.0":
		return v11.New(log, cs, initClients)
	}

	return nil, fmt.Errorf("version %q not found", cs.Config.PluginVersion)
}
