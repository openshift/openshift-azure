package main

import (
	"context"
	"flag"
	"io/ioutil"
	"net"
	"os"
	"path"
	"strings"
	texttemplate "text/template"

	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/util/errors"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/api/validate"
	"github.com/openshift/openshift-azure/pkg/arm"
	"github.com/openshift/openshift-azure/pkg/cluster"
	"github.com/openshift/openshift-azure/pkg/tls"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/cloudprovider"
	"github.com/openshift/openshift-azure/pkg/util/configblob"
	"github.com/openshift/openshift-azure/pkg/util/log"
	"github.com/openshift/openshift-azure/pkg/util/template"
)

var (
	logLevel  = flag.String("loglevel", "Debug", "valid values are Debug, Info, Warning, Error")
	gitCommit = "unknown"
)

type startup struct {
	log *logrus.Entry
	cs  *api.OpenShiftManagedCluster
	cpc *cloudprovider.Config
}

func (s *startup) getSecretFromVault(blockType, kvURL string) (string, error) {
	vaultURL, certName, err := azureclient.GetURLCertNameFromFullURL(kvURL)
	if err != nil {
		return "", err
	}
	cfg := auth.NewClientCredentialsConfig(s.cpc.AadClientID, s.cpc.AadClientSecret, s.cpc.TenantID)
	kvc, err := azureclient.NewKeyVaultClient(cfg, vaultURL)
	if err != nil {
		return "", err
	}
	bundle, err := kvc.GetSecret(context.Background(), vaultURL, certName, "")
	if err != nil {
		return "", err
	}
	return tls.GetPemBlock([]byte(*bundle.Value), blockType)
}

func (s *startup) writeTemplatedFiles() error {
	tc := api.TestConfig{RunningUnderTest: s.cs.Config.RunningUnderTest}
	hostname, _ := os.Hostname()
	cname, _ := net.LookupCNAME(hostname)
	domainname := strings.SplitN(strings.TrimSuffix(cname, "."), ".", 2)[1]

	for _, templateFileName := range arm.AssetNames() {
		if hostname != "master-000000" && templateFileName == "etc/origin/node/pods/sync.yaml" {
			continue
		}
		if templateFileName == "master-startup.sh" || templateFileName == "node-startup.sh" {
			continue
		}
		s.log.Debugf("processing template %s", templateFileName)
		templateFile, err := arm.Asset(templateFileName)
		if err != nil {
			return errors.Wrapf(err, "Asset(%s)", templateFileName)
		}

		b, err := template.Template(string(templateFile),
			texttemplate.FuncMap{
				"SecretFromVault": s.getSecretFromVault,
			}, s.cs, map[string]interface{}{
				"Hostname":    hostname,
				"DNSHostname": domainname,
				"TestConfig":  tc,
			})
		if err != nil {
			return errors.Wrapf(err, "Template(%s)", templateFileName)
		}
		destination := "/" + templateFileName
		parentDir := path.Dir(destination)
		err = os.MkdirAll(parentDir, 0755)
		if err != nil {
			return errors.Wrapf(err, "MkdirAll(%s)", parentDir)
		}
		var perm os.FileMode = 0666
		if path.Ext(destination) == ".key" || path.Ext(destination) == ".kubeconfig" {
			perm = 0600
		}

		err = ioutil.WriteFile(destination, b, perm)
		if err != nil {
			return errors.Wrapf(err, "WriteFile(%s)", destination)
		}
	}
	return nil
}

func (s *startup) startup(ctx context.Context) error {
	cpc, err := cloudprovider.Load("/etc/origin/cloudprovider/azure.conf")
	if err != nil {
		return err
	}
	s.cpc = cpc

	bsc, err := configblob.GetService(ctx, cpc)
	if err != nil {
		return errors.Wrap(err, "could not find storage account")
	}
	blob := bsc.GetContainerReference(cluster.ConfigContainerName).GetBlobReference(cluster.ConfigBlobName)

	s.log.Print("reading config blob")
	s.cs, err = configblob.GetBlob(blob)
	if err != nil {
		return errors.Wrap(err, "GetBlob")
	}
	v := validate.NewAPIValidator(s.cs.Config.RunningUnderTest)
	if errs := v.Validate(s.cs, nil, false); len(errs) > 0 {
		return errors.Wrap(kerrors.NewAggregate(errs), "can not validate config blob")
	}
	if err := s.writeTemplatedFiles(); err != nil {
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
	if err := s.startup(ctx); err != nil {
		log.Fatalf("startup %v", err)
	} else {
		log.Infoln("all done successfully")
	}
}
