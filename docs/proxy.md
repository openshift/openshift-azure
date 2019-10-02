# ARO Testing Proxy

Azure Red Hat OpenShift has 3 test Proxy servers running in
`australiasoutheast`, `eastus` and `westeurope`.

These Proxy servers are used to test PrivateCluster functionality.

## Management resource groups

Mangement resource groups contains not only Proxy servers,
but PrivateEndpoint subnets and some reserved ones for 
future use.
Networking configuration is defined below:

```
// subnets split logic:
// vnet - contains all network addresses used for manamagement infrastructure.
// at the moment it has 1024 addresses allocated.
// x.x.0.0/22 - vnet network size
//   x.x.1.0/24 - default subnet network
//   x.x.2.0/24 - management subnet, where all EP/PLS resources should be created

Australia:
            cidrVnet: "172.30.0.0/22"
            cidrDefaultSubnet: "172.30.0.0/24"
            cidrManagmentSubnet: "172.30.1.0/24"
WestEurope:
            cidrVnet: "172.30.8.0/22"
            cidrDefaultSubnet: "172.30.8.0/24"
            cidrManagmentSubnet: "172.30.9.0/24"
EastUS:
            cidrVnet: "172.30.16.0/22"
            cidrDefaultSubnet: "172.30.16.0/24"
            cidrManagmentSubnet: "172.30.17.0/24"
```
