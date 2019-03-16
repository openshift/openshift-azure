package main

import (
	"context"
	"flag"
	"net"
	"os"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/arm"
	"github.com/openshift/openshift-azure/pkg/config"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/log"
	"github.com/openshift/openshift-azure/pkg/util/managedcluster"
	"github.com/openshift/openshift-azure/pkg/util/vault"
	"github.com/openshift/openshift-azure/pkg/util/writers"
)

var (
	logLevel  = flag.String("loglevel", "Debug", "valid values are Debug, Info, Warning, Error")
	gitCommit = "unknown"
)

func run(ctx context.Context, log *logrus.Entry) error {
	log.Infof("reading config from %s", os.Args[1])
	cs, err := managedcluster.ReadConfig(os.Args[1])
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

	// TODO: validate that the config version matches our version

	log.Info("writing startup files")
	return arm.WriteStartupFiles(log, cs, writers.NewFilesystemWriter(), hostname, domainname)
}

func main() {
	flag.Parse()
	logger := logrus.New()
	logger.Formatter = &logrus.TextFormatter{FullTimestamp: true}
	logger.SetLevel(log.SanitizeLogLevel(*logLevel))
	log := logrus.NewEntry(logger)
	log.Infof("startup pod starting, git commit %s", gitCommit)

	if len(os.Args) != 2 {
		log.Fatalf("usage: %s /path/to/config.blob", os.Args[0])
	}

	if err := run(context.Background(), log); err != nil {
		log.Fatal(err)
	}

	log.Info("all done successfully")
}
