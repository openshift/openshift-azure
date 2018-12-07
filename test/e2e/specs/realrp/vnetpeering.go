package realrp

import (
	"context"
	"errors"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api/2018-09-30-preview/api"
	"github.com/openshift/openshift-azure/pkg/fakerp/client"
	"github.com/openshift/openshift-azure/test/clients/azure"
)

var _ = Describe("Peer Vnet tests [Vnet]", func() {
	var (
		ctx          = context.Background()
		cli          *azure.Client
		rg           = os.Getenv("RESOURCEGROUP")
		vnetPeerName = "vnetPeer"
		region       = os.Getenv("AZURE_REGION")
		clientID     = os.Getenv("AZURE_CLIENT_ID")
		clientSecret = os.Getenv("AZURE_CLIENT_SECRET")
	)

	BeforeEach(func() {
		var err error
		cli, err = azure.NewClientFromEnvironment(false)
		Expect(err).ToNot(HaveOccurred())
		if os.Getenv("AZURE_REGION") == "" {
			Expect(errors.New("AZURE_REGION is not set")).ToNot(HaveOccurred())
		}
		if os.Getenv("RESOURCEGROUP") == "" {
			Expect(errors.New("RESOURCEGROUP is not set")).ToNot(HaveOccurred())
		}
	})

	AfterEach(func() {
		result, _ := cli.Groups.Delete(ctx, rg)
		_ = result.WaitForCompletionRef(ctx, cli.Groups.Client())
	})

	It("should create the vnet and cluster and verify peering", func() {
		logrus.SetLevel(logrus.DebugLevel)
		logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
		logrus.SetOutput(GinkgoWriter)
		log := logrus.NewEntry(logrus.StandardLogger())

		// create a resource group and a vnet
		exists, err := cli.Groups.CheckExistence(ctx, rg)
		Expect(err).ToNot(HaveOccurred())
		if !exists {
			now := time.Now().String()
			ttl := "4h"
			_, err := cli.Groups.CreateOrUpdate(ctx, rg, resources.Group{
				Name:     &rg,
				Location: &region,
				Tags: map[string]*string{
					"now": &now,
					"ttl": &ttl,
				},
			})
			Expect(err).ToNot(HaveOccurred())
		}
		subnetName := "vnetPeerSubnet"
		subnetAddressPrefix := "192.168.0.0/24"
		future, err := cli.VirtualNetworks.CreateOrUpdate(ctx, rg, vnetPeerName, network.VirtualNetwork{
			VirtualNetworkPropertiesFormat: &network.VirtualNetworkPropertiesFormat{
				AddressSpace: &network.AddressSpace{
					AddressPrefixes: &[]string{
						"192.168.0.0/24",
					},
				},
				Subnets: &[]network.Subnet{
					{
						Name: &subnetName,
						SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
							AddressPrefix: &subnetAddressPrefix,
						},
					},
				},
			},
			Location: &region,
		})
		Expect(err).ToNot(HaveOccurred())
		err = future.WaitForCompletionRef(ctx, cli.VirtualNetworks.Client())
		Expect(err).ToNot(HaveOccurred())

		vnet, err := cli.VirtualNetworks.Get(ctx, rg, vnetPeerName, "")
		Expect(err).ToNot(HaveOccurred())
		Expect(len(*vnet.VirtualNetworkPeerings)).To(BeEquivalentTo(0))

		// load cluster config
		config, err := client.LoadClusterConfigFromManifest(log, "../../test/manifests/normal/create.yaml")
		Expect(err).ToNot(HaveOccurred())
		// Set clientid and secret if not set
		for _, ip := range config.Properties.AuthProfile.IdentityProviders {
			switch provider := ip.Provider.(type) {
			case (*api.AADIdentityProvider):
				if *provider.ClientID == "" {
					provider.ClientID = &clientID
				}
				if *provider.Secret == "" {
					provider.Secret = &clientSecret
				}
			}
		}
		// Set pre-created peer vnetid in cluster config
		config.Properties.NetworkProfile.PeerVnetID = vnet.ID

		// create a cluster with the peerVnet
		_, err = cli.OpenShiftManagedClusters.CreateOrUpdateAndWait(ctx, rg, rg, *config)
		Expect(err).NotTo(HaveOccurred())

		clusterVnet, err := cli.VirtualNetworks.Get(ctx, rg, vnetPeerName, "")
		Expect(err).NotTo(HaveOccurred())
		Expect(len(*clusterVnet.VirtualNetworkPeerings)).To(BeEquivalentTo(1))
		for _, vnetPeering := range *clusterVnet.VirtualNetworkPeerings {
			Expect(vnetPeering.PeeringState).To(BeEquivalentTo("Connected"))
			Expect(vnetPeering.Name).To(BeEquivalentTo("OSACustomerVNetPeer"))
		}
	})
})
