package sync

import (
	"context"
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	v20 "github.com/openshift/openshift-azure/pkg/sync/v20"
	v21 "github.com/openshift/openshift-azure/pkg/sync/v21"
	v22 "github.com/openshift/openshift-azure/pkg/sync/v22"
	v23 "github.com/openshift/openshift-azure/pkg/sync/v23"
)

type Interface interface {
	Sync(ctx context.Context) error
	ReadyHandler(w http.ResponseWriter, r *http.Request)
	PrintDB() error
	Hash() ([]byte, error)
}

func New(log *logrus.Entry, cs *api.OpenShiftManagedCluster, initClients bool) (Interface, error) {
	switch cs.Config.PluginVersion {
	case "v20.0":
		return v20.New(log, cs, initClients)
	case "v21.0":
		return v21.New(log, cs, initClients)
	case "v22.0":
		return v22.New(log, cs, initClients)
	case "v23.0":
		return v23.New(log, cs, initClients)
	}

	return nil, fmt.Errorf("version %q not found", cs.Config.PluginVersion)
}

// hack: these additional functions are a bit of a violation (really they ought
// to belong in Interface), but they can be (are) called with an unenriched
// `cs`, which is not currently true of New().

func AssetNames(cs *api.OpenShiftManagedCluster) ([]string, error) {
	switch cs.Config.PluginVersion {
	case "v20.0":
		return v20.AssetNames(), nil
	case "v21.0":
		return v21.AssetNames(), nil
	case "v22.0":
		return v22.AssetNames(), nil
	case "v23.0":
		return v23.AssetNames(), nil
	}

	return nil, fmt.Errorf("version %q not found", cs.Config.PluginVersion)
}

func Asset(cs *api.OpenShiftManagedCluster, name string) ([]byte, error) {
	switch cs.Config.PluginVersion {
	case "v20.0":
		return v20.Asset(name)
	case "v21.0":
		return v21.Asset(name)
	case "v22.0":
		return v22.Asset(name)
	case "v23.0":
		return v23.Asset(name)
	}

	return nil, fmt.Errorf("version %q not found", cs.Config.PluginVersion)
}
