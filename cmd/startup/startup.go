package main

import (
	"context"
	"encoding/json"
	"flag"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/arm"
	"github.com/openshift/openshift-azure/pkg/config"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/log"
	"github.com/openshift/openshift-azure/pkg/util/vault"
	"github.com/openshift/openshift-azure/pkg/util/wait"
	"github.com/openshift/openshift-azure/pkg/util/writers"
)

var (
	logLevel  = flag.String("loglevel", "Debug", "valid values are Debug, Info, Warning, Error")
	gitCommit = "unknown"
)

func run(ctx context.Context, log *logrus.Entry) error {
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
		return err
	}

	hostname, err := os.Hostname()
	if err != nil {
		return err
	}

	cname, err := net.LookupCNAME(hostname)
	if err != nil {
		return err
	}

	domainname := strings.SplitN(strings.TrimSuffix(cname, "."), ".", 2)[1]

	var spp *api.ServicePrincipalProfile
	if config.GetAgentRole(hostname) == api.AgentPoolProfileRoleMaster {
		spp = &cs.Properties.MasterServicePrincipalProfile
	} else {
		spp = &cs.Properties.WorkerServicePrincipalProfile
	}

	if config.GetAgentRole(hostname) == api.AgentPoolProfileRoleMaster {
		log.Info("creating clients")
		vaultauthorizer, err := azureclient.NewAuthorizer(spp.ClientID, spp.Secret, cs.Properties.AzProfile.TenantID, azureclient.KeyVaultEndpoint)
		if err != nil {
			return err
		}

		kvc := azureclient.NewKeyVaultClient(ctx, vaultauthorizer)

		log.Info("enriching config")
		err = vault.EnrichCSFromVault(ctx, kvc, cs)
		if err != nil {
			return err
		}
	}

	// TODO: validate that the config version matches our version

	log.Info("writing startup files")
	return arm.WriteStartupFiles(log, cs, config.GetAgentRole(hostname), writers.NewFilesystemWriter(), hostname, domainname)
}

func main() {
	flag.Parse()
	logger := logrus.New()
	logger.Formatter = &logrus.TextFormatter{FullTimestamp: true}
	logger.SetLevel(log.SanitizeLogLevel(*logLevel))
	log := logrus.NewEntry(logger)
	log.Infof("startup pod starting, git commit %s", gitCommit)

	if err := run(context.Background(), log); err != nil {
		log.Fatal(err)
	}

	log.Info("all done successfully")
}
