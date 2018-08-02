## openshift-azure

### Prerequisites

1. **Utilities**.  You'll need recent versions of
   [Azure CLI](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli),
   [Golang](https://golang.org/dl),
   [Kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl) installed.

1. **Azure CLI access**.  You'll need to be logged into Azure using the CLI.

1. **Subscription and Tenant ID**.  You'll need to know the subscription and
   tenant IDs of the Azure subscription where your OpenShift nodes will run.

1. **Node image**.  Your Azure subscription will need to contain an OpenShift
   node Image resource, or to have whitelisted access to the OpenShift node
   marketplace image.

   Before you deploy for the first time in a subscription using whitelisted
   access to the OpenShift node marketplace image, you will need to enable
   programmatic deployment of the image.

   In the Azure web console, click `Create a resource`.  Search for `OpenShift
   Origin 3.10 on Azure (Staged)` and click the result.  At the bottom of the
   resulting screen, click `Want to deploy programmatically?  Get started`.
   Click `Enable`, then `Save`.

1. **DNS domain**.  You'll need a DNS domain hosted using Azure DNS in the same
   subscription.  Deploying a cluster will create a dedicated child DNS zone
   resource for the cluster.  It is assumed that the name of the DNS zone
   resource for the parent DNS domain matches the name of the DNS domain.

1. **AAD Application / Service principal**.  The deployed OpenShift cluster
   needs a valid AAD application and service principal to call back into the
   Azure API, and optionally in order to enable AAD authentication.  There are a
   few options here:

   1. (Ask your Azure subscription administrator to) precreate a generic AAD
      application and service principal with secret and grant it *Contributor*
      access to the subscription.  Record the service principal client ID and
      secret.  Good enough to deploy OpenShift clusters, but AAD authentication
      won't work.

   1. Automatically create an AAD application and service principal.  Your Azure
      user will need *Contributor* and *User Access Administrator* roles, and
      your AAD will need to have *Users can register applications* enabled.

   1. (Ask your Azure subscription administrator to) precreate a specific AAD
      application and service principal with secret.  You can use `hack/aad.sh`
      to help with this process.  For AAD authentication to work, the public
      hostname of the OpenShift cluster must match the AAD application created.
      Record the service principal client ID and secret.

### Deploy an OpenShift cluster

1. Copy the `env.example` file to `env` and edit according to your requirements.
   Source the `env` file: `. ./env`.

1. Run `./create.sh $RESOURCEGROUP` to deploy a cluster.

1. To inspect pods running on the OpenShift cluster, run
   `KUBECONFIG=_data/_out/admin.kubeconfig oc get pods -n $RESOURCEGROUP`.

1. To ssh into an OpenShift node (vm-infra-0 or vm-compute-0), run
   `./ssh.sh hostname`.

1. Run `./delete.sh $RESOURCEGROUP` to delete the deployed cluster.

### Examples

Basic OpenShift configuration:

```yaml
name: openshift
location: eastus
properties:
  openShiftVersion: v3.10
  publicHostname: openshift.$RESOURCEGROUP.$DNS_DOMAIN
  routerProfiles:
  - name: default
    publicSubdomain: $RESOURCEGROUP.$DNS_DOMAIN
  agentPoolProfiles:
  - name: master
    role: master
    count: 3
    vmSize: Standard_D2s_v3
    osType: Linux
  - name: infra
    role: infra
    count: 1
    vmSize: Standard_D2s_v3
    osType: Linux
  - name: compute
    role: compute
    count: 1
    vmSize: Standard_D2s_v3
    osType: Linux
  servicePrincipalProfile:
    clientID: $AZURE_CLIENT_ID
    secret: $AZURE_CLIENT_SECRET
```

OpenShift with BYO VNET configuration:

```yaml
name: openshift
location: eastus
properties:
  openShiftVersion: v3.10
  publicHostname: openshift.$RESOURCEGROUP.$DNS_DOMAIN
  routerProfiles:
  - name: default
    publicSubdomain: $RESOURCEGROUP.$DNS_DOMAIN
  agentPoolProfiles:
  - name: master
    role: master
    count: 3
    vmSize: Standard_D2s_v3
    osType: Linux
    vnetSubnetID: /subscriptions/SUB_ID/resourceGroups/RG_NAME/providers/Microsoft.Network/virtualNetworks/VNET_NAME/subnets/SUBNET_NAME
  - name: infra
    role: infra
    count: 1
    vmSize: Standard_D2s_v3
    osType: Linux
    vnetSubnetID: /subscriptions/SUB_ID/resourceGroups/RG_NAME/providers/Microsoft.Network/virtualNetworks/VNET_NAME/subnets/SUBNET_NAME
  - name: compute
    role: compute
    count: 1
    vmSize: Standard_D2s_v3
    osType: Linux
    vnetSubnetID: /subscriptions/SUB_ID/resourceGroups/RG_NAME/providers/Microsoft.Network/virtualNetworks/VNET_NAME/subnets/SUBNET_NAME
  servicePrincipalProfile:
    clientID: $AZURE_CLIENT_ID
    secret: $AZURE_CLIENT_SECRET
```

You can create BYO VNET and subnet with commands:

```bash
az network vnet create -n $VNET_NAME -l $LOCATION -g $RESOURCEGROUP
az network vnet subnet create -n $SUBNET_NAME -g $RESOURCEGROUP --vnet-name $VNET_NAME --address-prefix 10.0.0.0/24
```
