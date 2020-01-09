package sync

import (
	"context"
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	v10 "github.com/openshift/openshift-azure/pkg/sync/v10"
	v12 "github.com/openshift/openshift-azure/pkg/sync/v12"
	v14 "github.com/openshift/openshift-azure/pkg/sync/v14"
	v15 "github.com/openshift/openshift-azure/pkg/sync/v15"
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
	case "v7.1":
		return v71.New(log, cs, initClients)
	case "v10.0", "v10.1", "v10.2":
		return v10.New(log, cs, initClients)
	case "v12.0", "v12.1", "v12.2":
		return v12.New(log, cs, initClients)
	case "v14.0":
		return v14.New(log, cs, initClients)
	case "v15.0":
		return v15.New(log, cs, initClients)
	}

	return nil, fmt.Errorf("version %q not found", cs.Config.PluginVersion)
}

// hack: these additional functions are a bit of a violation (really they ought
// to belong in Interface), but they can be (are) called with an unenriched
// `cs`, which is not currently true of New().

func AssetNames(cs *api.OpenShiftManagedCluster) ([]string, error) {
	switch cs.Config.PluginVersion {
	case "v14.0":
		return v14.AssetNames(), nil
	case "v15.0":
		return v15.AssetNames(), nil
	}

	return nil, fmt.Errorf("version %q not found", cs.Config.PluginVersion)
}

func Asset(cs *api.OpenShiftManagedCluster, name string) ([]byte, error) {
	switch cs.Config.PluginVersion {
	case "v14.0":
		return v14.Asset(name)
	case "v15.0":
		return v15.Asset(name)
	}

	return nil, fmt.Errorf("version %q not found", cs.Config.PluginVersion)
}
