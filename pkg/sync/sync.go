package sync

import (
	"context"
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	v15 "github.com/openshift/openshift-azure/pkg/sync/v15"
	v16 "github.com/openshift/openshift-azure/pkg/sync/v16"
	v17 "github.com/openshift/openshift-azure/pkg/sync/v17"
	v19 "github.com/openshift/openshift-azure/pkg/sync/v19"
	v20 "github.com/openshift/openshift-azure/pkg/sync/v20"
)

type Interface interface {
	Sync(ctx context.Context) error
	ReadyHandler(w http.ResponseWriter, r *http.Request)
	PrintDB() error
	Hash() ([]byte, error)
}

func New(log *logrus.Entry, cs *api.OpenShiftManagedCluster, initClients bool) (Interface, error) {
	switch cs.Config.PluginVersion {
	case "v15.0":
		return v15.New(log, cs, initClients)
	case "v16.0", "v16.1":
		return v16.New(log, cs, initClients)
	case "v17.0":
		return v17.New(log, cs, initClients)
	case "v19.0":
		return v19.New(log, cs, initClients)
	case "v20.0":
		return v20.New(log, cs, initClients)
	}

	return nil, fmt.Errorf("version %q not found", cs.Config.PluginVersion)
}

// hack: these additional functions are a bit of a violation (really they ought
// to belong in Interface), but they can be (are) called with an unenriched
// `cs`, which is not currently true of New().

func AssetNames(cs *api.OpenShiftManagedCluster) ([]string, error) {
	switch cs.Config.PluginVersion {
	case "v15.0":
		return v15.AssetNames(), nil
	case "v16.0", "v16.1":
		return v16.AssetNames(), nil
	case "v17.0":
		return v17.AssetNames(), nil
	case "v19.0":
		return v19.AssetNames(), nil
	case "v20.0":
		return v20.AssetNames(), nil
	}

	return nil, fmt.Errorf("version %q not found", cs.Config.PluginVersion)
}

func Asset(cs *api.OpenShiftManagedCluster, name string) ([]byte, error) {
	switch cs.Config.PluginVersion {
	case "v15.0":
		return v15.Asset(name)
	case "v16.0", "v16.1":
		return v16.Asset(name)
	case "v17.0":
		return v17.Asset(name)
	case "v19.0":
		return v19.Asset(name)
	case "v20.0":
		return v20.Asset(name)
	}

	return nil, fmt.Errorf("version %q not found", cs.Config.PluginVersion)
}
