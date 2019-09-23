# ARO Testing VPN

Azure Red Hat OpenShift has 3 test VPN servers running in
`australiasoutheast`, `eastus` and `westeurope`.

These VPN servers are used to test PrivateCluster functionality.
We use `openvpn` clients to initiate the connection.

## VPN client usage

VPN client can be accessed native openvpn binary.

To initiate VPN connection to `eastus` execute:
```
sudo openvpn secrets/vpn-eastus.ovpn
```


VPN servers are using `172.30.0.0/16` network. If you are using this network for your home/work network - you will have a gateway clash. 

When VPN tunnel is established, you should have new route added
to your laptop routing table for `172.30.{0,8,16}.0/22` to `tun0`.
This would allow you to query and access ARO API server via 
`PrivateLinkService` and `PrivateEndpoint` in management resource 
groups

## Management resource groups

Mangement resource groups contains not only VPN servers,
but PrivateEndpoint subnets and some reserved ones for 
future use.
Networking configuration is defined below:

```
// subnets split logic:
// vnet - contains all network addresses used for manamagement infrastructure.
// at the moment it has 1024 addresses allocated.
// x.x.0.0/22 - vnet network size
//   x.x.1.0/24 - subnet for the gateway
//   x.x.2.0/24 - management subnet, where all EP/PLS resources should be created
//   x.x.3.0/24 - reserved for future extensions
// x.x.4.0/24 - out of the vnet subnet for VPN clients.

Australia:
            cidrVnet: "172.30.0.0/22"
            cidrDefaultSubnet: "172.30.0.0/24"
            cidrGatewaySubnet: "172.30.1.0/24"
            cidrManagmentSubnet: "172.30.2.0/24"
            cidrClients: "172.30.4.0/24"
WestEurope:
            cidrVnet: "172.30.8.0/22"
            cidrDefaultSubnet: "172.30.8.0/24"
            cidrGatewaySubnet: "172.30.9.0/24"
            cidrManagmentSubnet: "172.30.10.0/24"
            cidrClients: "172.30.12.0/24"
EastUS:
            cidrVnet: "172.30.16.0/22"
            cidrDefaultSubnet: "172.30.16.0/24"
            cidrGatewaySubnet: "172.30.17.0/24"
            cidrManagmentSubnet: "172.30.18.0/24"
            cidrClients: "172.30.20.0/24"
```
