package realrp

import (
	"context"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"

	"github.com/openshift/openshift-azure/pkg/api/2018-09-30-preview/api"
	"github.com/openshift/openshift-azure/pkg/fakerp/client"
	"github.com/openshift/openshift-azure/test/clients/azure"
	tlog "github.com/openshift/openshift-azure/test/util/log"
)

var _ = Describe("Peer Vnet tests [Vnet][Real][LongRunning]", func() {
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
		Expect(err).NotTo(HaveOccurred())
		Expect(region).NotTo(BeEmpty())
		Expect(rg).NotTo(BeEmpty())
	})

	AfterEach(func() {
		result, _ := cli.Groups.Delete(ctx, rg)
		_ = result.WaitForCompletionRef(ctx, cli.Groups.Client())
	})

	It("should create the vnet and cluster and verify peering", func() {
		log := tlog.GetTestLogger()

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
		Expect(future).ToNot(BeNil())
		err = future.WaitForCompletionRef(ctx, cli.VirtualNetworks.Client())
		Expect(err).ToNot(HaveOccurred())

		vnet, err := cli.VirtualNetworks.Get(ctx, rg, vnetPeerName, "")
		Expect(err).ToNot(HaveOccurred())
		Expect(vnet).ToNot(BeNil())
		Expect(len(*vnet.VirtualNetworkPeerings)).To(Equal(0))

		// load cluster config
		config, err := client.LoadClusterConfigFromManifest(log, "../../test/manifests/normal/create.yaml")
		Expect(err).ToNot(HaveOccurred())
		Expect(config).ToNot(BeNil())
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

		vnetPeer, err := cli.VirtualNetworks.Get(ctx, rg, vnetPeerName, "")
		Expect(err).ToNot(HaveOccurred())
		Expect(vnetPeer).ToNot(BeNil())
		Expect(len(*vnetPeer.VirtualNetworkPeerings)).To(BeEquivalentTo(1))
		for _, vnetPeering := range *vnetPeer.VirtualNetworkPeerings {
			Expect(vnetPeering.PeeringState).To(BeEquivalentTo("Connected"))
			Expect(*vnetPeering.Name).To(BeEquivalentTo("OSACustomerVNetPeer"))
		}
	})
})
