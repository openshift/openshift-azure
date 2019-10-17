package proxyinfrastructure

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"

	azresources "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/sirupsen/logrus"

	fakerparm "github.com/openshift/openshift-azure/pkg/fakerp/arm"
	"github.com/openshift/openshift-azure/pkg/util/arm"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/resources"
	"github.com/openshift/openshift-azure/pkg/util/template"
	"github.com/openshift/openshift-azure/pkg/util/tls"
)

//go:generate go get github.com/go-bindata/go-bindata/go-bindata
//go:generate go-bindata -nometadata -pkg $GOPACKAGE -prefix data data
//go:generate gofmt -s -l -w bindata.go

const (
	vnetName                 = "vnet"
	vnetSubnetName           = "default"
	vnetManagementSubnetName = "management"
	ipName                   = "ip"
	nsgName                  = "nsg"
	nicName                  = "nic"
	vmName                   = "vm"
	cseName                  = "vm/cse"
	adminUsername            = "cloud-user"
)

type netDefinition struct {
	Vnet             string
	DefaultSubnet    string
	ManagementSubnet string
}

var (
	// The versions referenced here must be kept in lockstep with the imports
	// above.
	versionMap = map[string]string{
		"Microsoft.Compute": "2018-10-01",
		"Microsoft.Network": "2018-07-01",
	}

	// subnets split logic:
	// vnet - contains all network addresses used for manamagement infrastructure.
	// at the moment it has 1024 addresses allocated.
	// x.x.0.0/22 - vnet network size
	//  x.x.0.0/24 - default subnet
	//  x.x.1.0/24 - management subnet, where all EP/PLS resources should be created
	netDefinitions = map[string]netDefinition{
		"australiasoutheast": {
			Vnet:             "172.30.0.0/22",
			DefaultSubnet:    "172.30.0.0/24",
			ManagementSubnet: "172.30.1.0/24",
		},
		"westeurope": {
			Vnet:             "172.30.8.0/22",
			DefaultSubnet:    "172.30.8.0/24",
			ManagementSubnet: "172.30.9.0/24",
		},
		"eastus": {
			Vnet:             "172.30.16.0/22",
			DefaultSubnet:    "172.30.16.0/24",
			ManagementSubnet: "172.30.17.0/24",
		},
	}
)

type Config struct {
	resourceGroup string
	NetDefinition netDefinition
	region        string
	sshKey        *rsa.PrivateKey

	ServerKey  *rsa.PrivateKey
	ServerCert *x509.Certificate
	Ca         *x509.Certificate
}

func Run(ctx context.Context, log *logrus.Entry) error {
	for region, netDefinition := range netDefinitions {
		// generate ssh key
		sshkey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return err
		}

		b, err := tls.PrivateKeyAsBytes(sshkey)
		if err != nil {
			return err
		}

		err = ioutil.WriteFile(fmt.Sprintf("secrets/id_rsa-%s", region), b, 0600)
		if err != nil {
			return err
		}

		// read certs
		b, err = ioutil.ReadFile("secrets/proxy-server.pem")
		if err != nil {
			return err
		}

		serverCert, err := tls.ParseCert(b)
		if err != nil {
			return err
		}

		b, err = ioutil.ReadFile("secrets/proxy-server.key")
		if err != nil {
			return err
		}

		serverKey, err := tls.ParsePrivateKey(b)
		if err != nil {
			return err
		}

		b, err = ioutil.ReadFile("secrets/proxy-ca.pem")
		if err != nil {
			return err
		}

		ca, err := tls.ParseCert(b)
		if err != nil {
			return err
		}

		conf := &Config{
			resourceGroup: "management-" + region,
			NetDefinition: netDefinition,
			region:        region,
			sshKey:        sshkey,
			ServerCert:    serverCert,
			ServerKey:     serverKey,
			Ca:            ca,
		}

		// create resource groups for management vnets
		err = ensureResourceGroup(log, conf)
		if err != nil {
			return err
		}

		err = ensureResources(log, conf)
		if err != nil {
			return err
		}

	}
	return nil
}

// azureclient creates a resource group and returns whether the
// resource group was actually created or not and any error encountered.
func ensureResourceGroup(log *logrus.Entry, conf *Config) error {
	log.Debug("ensureResourceGroup")
	authorizer, err := azureclient.NewAuthorizerFromEnvironment("")
	if err != nil {
		return err
	}
	ctx := context.Background()
	gc := resources.NewGroupsClient(ctx, log, os.Getenv("AZURE_SUBSCRIPTION_ID"), authorizer)

	if _, err := gc.Get(ctx, conf.resourceGroup); err == nil {
		return nil
	}

	_, err = gc.CreateOrUpdate(ctx, conf.resourceGroup, azresources.Group{Location: &conf.region})

	return err
}

// ensureResources creates a resources and returns whether the
// resources were actually created or not and any error encountered.
func ensureResources(log *logrus.Entry, conf *Config) error {
	authorizer, err := azureclient.NewAuthorizerFromEnvironment("")
	if err != nil {
		return err
	}
	ctx := context.Background()
	deployments := resources.NewDeploymentsClient(ctx, log, os.Getenv("AZURE_SUBSCRIPTION_ID"), authorizer)

	template, err := generate(ctx, conf)
	if err != nil {
		return err
	}
	future, err := deployments.CreateOrUpdate(ctx, conf.resourceGroup, "azuredeploy", azresources.Deployment{
		Properties: &azresources.DeploymentProperties{
			Template: template,
			Mode:     azresources.Incremental,
		},
	})
	if err != nil {
		return err
	}

	log.Info("waiting for arm template deployment to complete")
	err = future.WaitForCompletionRef(ctx, deployments.Client())
	if err != nil {
		log.Warnf("deployment failed: %#v", err)
	}

	return nil
}

// Generate generates fakeRP callback function objects for. This function mocks realRP
// impementation for required objects
func generate(ctx context.Context, conf *Config) (map[string]interface{}, error) {
	script, err := template.Template("start.sh", string(MustAsset("start.sh")), nil, map[string]interface{}{
		"Config": conf,
	})
	if err != nil {
		return nil, err
	}

	cs, err := cse(conf, script)
	if err != nil {
		return nil, err
	}

	resources := []interface{}{
		vnet(conf),
		ip(conf),
		nsg(conf),
		nic(conf),
		vm(conf),
		cs,
	}

	template, err := fakerparm.Generate(ctx, os.Getenv("AZURE_SUBSCRIPTION_ID"), conf.resourceGroup, resources)
	if err != nil {
		return nil, err
	}

	arm.FixupAPIVersions(template, versionMap)
	arm.FixupSDKMismatch(template)
	arm.FixupDepends(os.Getenv("AZURE_SUBSCRIPTION_ID"), conf.resourceGroup, template)

	return template, nil
}
