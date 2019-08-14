package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"flag"
	"io/ioutil"
	"os"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/resources"
	"github.com/openshift/openshift-azure/pkg/util/log"
	"github.com/openshift/openshift-azure/pkg/util/random"
	"github.com/openshift/openshift-azure/pkg/util/resourceid"
	"github.com/openshift/openshift-azure/pkg/util/tls"
	"github.com/openshift/openshift-azure/pkg/vmimage"
)

var (
	gitCommit = "unknown"

	timestamp = time.Now().UTC().Format("200601021504")

	logLevel                   = flag.String("loglevel", "Debug", "Valid values are Debug, Info, Warning, Error")
	location                   = flag.String("location", "eastus", "location")
	buildResourceGroup         = flag.String("buildResourceGroup", "vmimage-"+timestamp, "build resource group")
	preserveBuildResourceGroup = flag.Bool("preserveBuildResourceGroup", false, "preserve build resource group after build")
	image                      = flag.String("image", "", "image name")
	imageResourceGroup         = flag.String("imageResourceGroup", "images", "image resource group")
	imageStorageAccount        = flag.String("imageStorageAccount", "openshiftimages", "image storage account")
	imageContainer             = flag.String("imageContainer", "images", "image container")
	imageSku                   = flag.String("imageSku", "", "image SKU")
	imageVersion               = flag.String("imageVersion", "", "image version")
	clientKey                  = flag.String("clientKey", "secrets/client-key.pem", "cdn client key")
	clientCert                 = flag.String("clientCert", "secrets/client-cert.pem", "cdn client cert")
	validate                   = flag.Bool("validate", false, "If set, will create VM with provided image and will try to update it")
)

func run(ctx context.Context, log *logrus.Entry) error {
	b, err := ioutil.ReadFile(*clientKey)
	if err != nil {
		return err
	}

	clientKey, err := tls.ParsePrivateKey(b)
	if err != nil {
		return err
	}

	b, err = ioutil.ReadFile(*clientCert)
	if err != nil {
		return err
	}

	clientCert, err := tls.ParseCert(b)
	if err != nil {
		return err
	}

	authorizer, err := azureclient.NewAuthorizerFromEnvironment("")
	if err != nil {
		return err
	}

	sshkey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	b, err = tls.PrivateKeyAsBytes(sshkey)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile("id_rsa", b, 0600)
	if err != nil {
		return err
	}

	domainNameLabel, err := random.LowerCaseAlphaString(20)
	if err != nil {
		return err
	}

	builder := vmimage.Builder{
		GitCommit:                  gitCommit,
		Log:                        log,
		Deployments:                resources.NewDeploymentsClient(ctx, log, os.Getenv("AZURE_SUBSCRIPTION_ID"), authorizer),
		Groups:                     resources.NewGroupsClient(ctx, log, os.Getenv("AZURE_SUBSCRIPTION_ID"), authorizer),
		SubscriptionID:             os.Getenv("AZURE_SUBSCRIPTION_ID"),
		Location:                   *location,
		BuildResourceGroup:         *buildResourceGroup,
		PreserveBuildResourceGroup: *preserveBuildResourceGroup,
		DomainNameLabel:            domainNameLabel,
		Image:                      *image,
		ImageResourceGroup:         *imageResourceGroup,
		ImageStorageAccount:        *imageStorageAccount,
		ImageContainer:             *imageContainer,
		ImageSku:                   *imageSku,
		ImageVersion:               *imageVersion,
		SSHKey:                     sshkey,
		ClientKey:                  clientKey,
		ClientCert:                 clientCert,
		Validate:                   *validate,
	}

	err = builder.ValidateFields()
	if err != nil {
		return err
	}

	err = builder.Run(ctx)
	if err != nil {
		return err
	}

	// for debug purposes
	if *preserveBuildResourceGroup {
		return nil
	}

	return os.Remove("id_rsa")
}

func main() {
	flag.Parse()
	logrus.SetLevel(log.SanitizeLogLevel(*logLevel))
	logrus.SetReportCaller(true)
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	log := logrus.NewEntry(logrus.StandardLogger())
	log.Printf("vmimage starting, git commit %s", gitCommit)

	err := run(context.Background(), log)
	if err != nil {
		log.Fatal(err)
	}

	log.Infof("built image %s", resourceid.ResourceID(os.Getenv("AZURE_SUBSCRIPTION_ID"), *imageResourceGroup, "providers/Microsoft.Compute/images", *image))
}
