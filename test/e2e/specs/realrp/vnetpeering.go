package realrp

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"

	v20190430 "github.com/openshift/openshift-azure/pkg/api/2019-04-30"
	"github.com/openshift/openshift-azure/pkg/fakerp/client"
	"github.com/openshift/openshift-azure/test/clients/azure"
	"github.com/openshift/openshift-azure/test/util/log"
	tlog "github.com/openshift/openshift-azure/test/util/log"
)

var _ = Describe("Peer Vnet tests [Vnet][Real][LongRunning]", func() {
	var (
		cfg          *client.Config
		cli          *azure.Client
		vnetPeerName = "vnetPeer"
	)

	BeforeEach(func() {
		var err error
		cli, err = azure.NewClientFromEnvironment(context.Background(), log.GetTestLogger(), false)
		Expect(err).NotTo(HaveOccurred())

		cfg, err = client.NewConfig(tlog.GetTestLogger())
		Expect(err).NotTo(HaveOccurred())

		// create a new resource group
		err = client.EnsureResourceGroup(cfg)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		ctx, cancelFn := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancelFn()
		By(fmt.Sprintf("deleting resource group %s", cfg.ResourceGroup))
		err := cli.Groups.Delete(ctx, cfg.ResourceGroup)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should create the vnet and cluster and verify peering", func() {
		ctx, cancelFn := context.WithTimeout(context.Background(), time.Hour)
		defer cancelFn()

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

		By("setting the custom vnet id in the osa request for peering")
		vnet, err := cli.VirtualNetworks.Get(ctx, cfg.ResourceGroup, vnetPeerName, "")
		Expect(err).ToNot(HaveOccurred())
		Expect(len(*vnet.VirtualNetworkPeerings)).To(Equal(0))

		// load cluster config
		var config v20190430.OpenShiftManagedCluster
		err = client.GenerateManifest(cfg, "../../test/manifests/realrp/create.yaml", &config)
		Expect(err).ToNot(HaveOccurred())

		// Set pre-created peer vnetid in cluster config
		config.Properties.NetworkProfile.PeerVnetID = vnet.ID

		// create a cluster with the peerVnet
		By("creating an OSA cluster")
		_, err = cli.OpenShiftManagedClusters.CreateOrUpdateAndWait(ctx, cfg.ResourceGroup, cfg.ResourceGroup, config)
		Expect(err).NotTo(HaveOccurred())

		By("ensuring the OSA cluster vnet is peered with the custom vnet")
		vnetPeer, err := cli.VirtualNetworks.Get(ctx, cfg.ResourceGroup, vnetPeerName, "")
		Expect(err).ToNot(HaveOccurred())
		Expect(len(*vnetPeer.VirtualNetworkPeerings)).To(Equal(1))
		for _, vnetPeering := range *vnetPeer.VirtualNetworkPeerings {
			Expect(vnetPeering.PeeringState).To(Equal("Connected"))
			Expect(*vnetPeering.Name).To(Equal("OSACustomerVNetPeer"))
		}
	})
})
