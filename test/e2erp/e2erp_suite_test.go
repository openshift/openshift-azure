//+build e2erp

package e2erp

import (
	"context"
	"flag"
	"fmt"
	"testing"

	"github.com/kelseyhightower/envconfig"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/log"
)

var (
	c            *testClient
	azureConf    AzureConfig
	gitCommit    = "unknown"
	logger       *logrus.Entry
	ctx          context.Context
	pluginConfig *api.PluginConfig
	manifest     = flag.String("manifest", "../../_data/manifest.yaml", "Path to the manifest to send to the RP")
	configBlob   = flag.String("configBlob", "../../_data/containerservice.yaml", "Path on disk where the OpenShift internal config blob should be written")
	logLevel     = flag.String("logLevel", "Debug", "The log level to use")
)

var _ = BeforeSuite(func() {
	err := envconfig.Process("", &azureConf)
	if err != nil {
		panic(err)
	}
	c = newTestClient(azureConf)
	pluginConfig = newPluginConfig()

	ctx = context.Background()
	ctx = context.WithValue(ctx, api.ContextKeyClientID, azureConf.ClientID)
	ctx = context.WithValue(ctx, api.ContextKeyClientSecret, azureConf.ClientSecret)
	ctx = context.WithValue(ctx, api.ContextKeyTenantID, azureConf.TenantID)

	logrus.SetLevel(log.SanitizeLogLevel(*logLevel))
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	logger = logrus.WithFields(logrus.Fields{"location": c.location, "resourceGroup": c.resourceGroup})
	logger.Debugf("manifest path: %s", *manifest)
	logger.Debugf("config blob path: %s", *configBlob)
})

func TestE2eRP(t *testing.T) {
	flag.Parse()
	fmt.Printf("E2E Resource Provider tests starting, git commit %s\n", gitCommit)
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2E Resource Provider Suite")
}
