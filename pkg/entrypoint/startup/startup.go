package startup

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"reflect"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/startup"
	"github.com/openshift/openshift-azure/pkg/util/cmp"
	"github.com/openshift/openshift-azure/pkg/util/log"
	"github.com/openshift/openshift-azure/pkg/util/wait"
)

func readConfig(ctx context.Context, log *logrus.Entry) (*api.OpenShiftManagedCluster, error) {
	log.Infof("reading config")
	var cs *api.OpenShiftManagedCluster
	err := wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		resp, err := http.Get(os.Getenv("SASURI"))
		if err != nil {
			log.Info(err)
			return false, nil
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			log.Infof("unexpected status code %d", resp.StatusCode)
			return false, nil
		}
		err = json.NewDecoder(resp.Body).Decode(&cs)
		if err != nil {
			log.Info(err)
			return false, nil
		}
		return true, nil
	}, ctx.Done())
	if err != nil {
		return nil, err
	}

	return cs, err
}

func runInitNetwork(ctx context.Context, log *logrus.Entry) error {
	cs, err := readConfig(ctx, log)
	if err != nil {
		return err
	}

	s, err := startup.New(log, cs, api.TestConfig{})
	if err != nil {
		return err
	}

	err = s.WriteSearchDomain(ctx, log)
	if err != nil {
		return err
	}

	return nil
}

func runStartup(ctx context.Context, log *logrus.Entry) error {
	cs, err := readConfig(ctx, log)
	if err != nil {
		return err
	}

	ns, err := startup.GetNameserversFromDNSConfig("/host")
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(ns, cs.Properties.NetworkProfile.Nameservers) {
		log.Warnf("overriding Nameservers with values from host resolver difference %s", cmp.Diff(ns, cs.Properties.NetworkProfile.Nameservers))
		cs.Properties.NetworkProfile.Nameservers = ns
	}
	s, err := startup.New(log, cs, api.TestConfig{})
	if err != nil {
		return err
	}

	log.Info("writing startup files")
	return s.WriteFiles(ctx)
}

func start(cfg *cmdConfig) error {
	logger := logrus.New()
	logger.Formatter = &logrus.TextFormatter{FullTimestamp: true}
	logger.SetLevel(log.SanitizeLogLevel(cfg.LogLevel))
	log := logrus.NewEntry(logger)
	log.Info("startup pod starting")

	if cfg.initNetwork {
		if err := runInitNetwork(context.Background(), log); err != nil {
			return err
		}
	} else {
		if err := runStartup(context.Background(), log); err != nil {
			return err
		}
	}

	log.Info("all done successfully")
	return nil
}
