package realrp

import (
	"context"
	"fmt"
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
		cfg          *client.Config
		ctx          = context.Background()
		cli          *azure.Client
		vnetPeerName = "vnetPeer"
	)

	BeforeEach(func() {
		var err error
		cli, err = azure.NewClientFromEnvironment(false)
		Expect(err).NotTo(HaveOccurred())
		cfg, err = client.NewConfig(tlog.GetTestLogger(), true)
		Expect(err).NotTo(HaveOccurred())

		// create a new resource group
		now := time.Now().String()
		ttl := "4h"
		By(fmt.Sprintf("creating resource group %s", cfg.ResourceGroup))
		_, err = cli.Groups.CreateOrUpdate(ctx, cfg.ResourceGroup, resources.Group{
			Name:     &cfg.ResourceGroup,
			Location: &cfg.Region,
			Tags: map[string]*string{
				"now": &now,
				"ttl": &ttl,
			},
		})
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		By(fmt.Sprintf("deleting resource group %s", cfg.ResourceGroup))
		result, err := cli.Groups.Delete(ctx, cfg.ResourceGroup)
		Expect(err).NotTo(HaveOccurred())
		result.WaitForCompletionRef(ctx, cli.Groups.Client())
	})

	It("should create the vnet and cluster and verify peering", func() {
		// create a vnet
		subnetName := "vnetPeerSubnet"
		subnetAddressPrefix := "192.168.0.0/24"
		By("creating a custom vnet")
		future, err := cli.VirtualNetworks.CreateOrUpdate(ctx, cfg.ResourceGroup, vnetPeerName, network.VirtualNetwork{
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
			Location: &cfg.Region,
		})
		Expect(err).ToNot(HaveOccurred())
		err = future.WaitForCompletionRef(ctx, cli.VirtualNetworks.Client())
		Expect(err).ToNot(HaveOccurred())

		vnet, err := cli.VirtualNetworks.Get(ctx, cfg.ResourceGroup, vnetPeerName, "")
		Expect(err).ToNot(HaveOccurred())
		Expect(len(*vnet.VirtualNetworkPeerings)).To(Equal(0))

		// load cluster config
		config, err := client.LoadClusterConfigFromManifest(tlog.GetTestLogger(), "../../test/manifests/normal/create.yaml")
		Expect(err).ToNot(HaveOccurred())
		// Set clientid and secret if not set
		for _, ip := range config.Properties.AuthProfile.IdentityProviders {
			switch provider := ip.Provider.(type) {
			case (*api.AADIdentityProvider):
				if *provider.ClientID == "" {
					provider.ClientID = &cfg.ClientID
				}
				if *provider.Secret == "" {
					provider.Secret = &cfg.ClientSecret
				}
			}
		}
		// Set pre-created peer vnetid in cluster config
		config.Properties.NetworkProfile.PeerVnetID = vnet.ID

		// create a cluster with the peerVnet
		By("creating an OSA cluster")
		_, err = cli.OpenShiftManagedClusters.CreateOrUpdateAndWait(ctx, cfg.ResourceGroup, cfg.ResourceGroup, *config)
		Expect(err).NotTo(HaveOccurred())

		By("ensuring the OSA cluster vnet is peered with the custom vnet")
		vnetPeer, err := cli.VirtualNetworks.Get(ctx, cfg.ResourceGroup, vnetPeerName, "")
		Expect(err).ToNot(HaveOccurred())
		Expect(len(*vnetPeer.VirtualNetworkPeerings)).To(BeEquivalentTo(1))
		for _, vnetPeering := range *vnetPeer.VirtualNetworkPeerings {
			Expect(vnetPeering.PeeringState).To(BeEquivalentTo("Connected"))
			Expect(*vnetPeering.Name).To(BeEquivalentTo("OSACustomerVNetPeer"))
		}
	})
})
