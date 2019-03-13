package main

import (
	"context"
	"flag"
	"net"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/util/errors"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/api/validate"
	"github.com/openshift/openshift-azure/pkg/arm"
	"github.com/openshift/openshift-azure/pkg/cluster"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	azureclientstorage "github.com/openshift/openshift-azure/pkg/util/azureclient/storage"
	"github.com/openshift/openshift-azure/pkg/util/cloudprovider"
	"github.com/openshift/openshift-azure/pkg/util/configblob"
	"github.com/openshift/openshift-azure/pkg/util/log"
	"github.com/openshift/openshift-azure/pkg/util/vault"
	"github.com/openshift/openshift-azure/pkg/util/writers"
)

var (
	logLevel  = flag.String("loglevel", "Debug", "valid values are Debug, Info, Warning, Error")
	gitCommit = "unknown"
)

type startup struct {
	log  *logrus.Entry
	cs   *api.OpenShiftManagedCluster
	kvc  azureclient.KeyVaultClient
	blob azureclientstorage.Blob
}

func (s *startup) initClients(ctx context.Context) error {
	cpc, err := cloudprovider.Load("/etc/origin/cloudprovider/azure.conf")
	if err != nil {
		return err
	}

	vaultauthorizer, err := azureclient.NewAuthorizer(cpc.AadClientID, cpc.AadClientSecret, cpc.TenantID, azureclient.KeyVaultEndpoint)
	if err != nil {
		return err
	}

	s.kvc = azureclient.NewKeyVaultClient(ctx, vaultauthorizer)

	bsc, err := configblob.GetService(ctx, cpc)
	if err != nil {
		return errors.Wrap(err, "could not find storage account")
	}
	s.blob = bsc.GetContainerReference(cluster.ConfigContainerName).GetBlobReference(cluster.ConfigBlobName)

	return nil
}

func (s *startup) run(ctx context.Context) error {
	var err error

	s.log.Print("reading config blob")
	s.cs, err = configblob.GetBlob(s.blob)
	if err != nil {
		return errors.Wrap(err, "GetBlob")
	}

	s.log.Print("enriching config blob")
	err = vault.EnrichCSFromVault(ctx, s.kvc, s.cs)
	if err != nil {
		return errors.Wrap(err, "EnrichCSFromVault")
	}

	v := validate.NewAPIValidator(s.cs.Config.RunningUnderTest)
	if errs := v.Validate(s.cs, nil, false); len(errs) > 0 {
		return errors.Wrap(kerrors.NewAggregate(errs), "can not validate config blob")
	}

	hostname, _ := os.Hostname()
	cname, _ := net.LookupCNAME(hostname)
	domainname := strings.SplitN(strings.TrimSuffix(cname, "."), ".", 2)[1]

	if err := arm.WriteTemplatedFiles(s.log, s.cs, writers.NewFilesystemWriter(), hostname, domainname); err != nil {
		return errors.Wrap(err, "writeTemplatedFiles")
	}

	return nil
}

func main() {
	flag.Parse()
	logger := logrus.New()
	logger.Formatter = &logrus.TextFormatter{FullTimestamp: true}
	logger.SetLevel(log.SanitizeLogLevel(*logLevel))
	log := logrus.NewEntry(logger)
	log.Printf("startup pod starting, git commit %s", gitCommit)

	ctx := context.Background()
	s := startup{log: log}
	if err := s.initClients(ctx); err != nil {
		log.Fatalf("initClients %v", err)
	}
	if err := s.run(ctx); err != nil {
		log.Fatalf("run %v", err)
	}
	log.Infoln("all done successfully")
}
