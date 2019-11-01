package sanity

import (
	"context"
	"io/ioutil"
	"os"

	"github.com/ghodss/yaml"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/fakerp"
	"github.com/openshift/openshift-azure/pkg/fakerp/client"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/test/e2e/standard"
	testlogger "github.com/openshift/openshift-azure/test/util/log"
)

// Checker is a singleton sanity checker which is used by all the Ginkgo tests
var Checker *standard.SanityChecker

func init() {
	b, err := ioutil.ReadFile("../../_data/containerservice.yaml") // running via `go test`
	if os.IsNotExist(err) {
		b, err = ioutil.ReadFile("_data/containerservice.yaml") // running via compiled test binary
	}
	if err != nil {
		panic(err)
	}

	var cs *api.OpenShiftManagedCluster
	if err := yaml.Unmarshal(b, &cs); err != nil {
		panic(err)
	}
	ctx := context.Background()
	log := testlogger.GetTestLogger()
	if cs.Properties.PrivateAPIServer {
		// privateEndpoint is not serialised, so we need to retrieve it.
		conf, err := client.NewServerConfig(log, cs)
		if err != nil {
			panic(err)
		}
		authorizer, err := azureclient.NewAuthorizerFromEnvironment("")
		if err != nil {
			panic(err)
		}
		ctx = context.WithValue(ctx, api.ContextKeyClientAuthorizer, authorizer)
		cs.Properties.NetworkProfile.PrivateEndpoint, err = fakerp.GetPrivateEndpointIP(ctx, log, cs.Properties.AzProfile.SubscriptionID, conf.ManagementResourceGroup, cs.Properties.AzProfile.ResourceGroup)
		if err != nil {
			panic(err)
		}
	}

	Checker, err = standard.NewSanityChecker(ctx, log, cs)
	if err != nil {
		panic(err)
	}
}
