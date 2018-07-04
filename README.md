## azure-helm

### Prerequisites

1. **Utilities**.  You'll need recent versions of
   [Azure CLI](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli),
   [Golang](https://golang.org/dl),
   [Helm client](https://github.com/kubernetes/helm/releases/latest) and
   [Kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl) installed.

1. **AKS cluster**.  You'll need an AKS cluster running the kubenet network
   plugin (currently [AKS#500](https://github.com/Azure/AKS/issues/500) prevents
   use of the azure network plugin for this workload).

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
      application and service principal with secret.  You can use `tools/aad.sh`
      to help with this process.  For AAD authentication to work, the public
      hostname of the OpenShift cluster must match the AAD application created.
      Record the service principal client ID and secret.

### Deploy an OpenShift cluster

1. If you have access to an AKS cluster, run `az aks get-credentials -g
   $AKS_RESOURCEGROUP -n $AKS_NAME -f - >$aks/admin.kubeconfig` or manually
   place the admin kubeconfig in `aks/admin.kubeconfig`.

   If you don't have access to an AKS cluster, run `aks/create.sh
   $AKS_RESOURCEGROUP ~/.ssh/id_rsa.pub` to create a new AKS cluster.

1. Copy the `env.example` file to `env` and edit according to your requirements.
   Source the `env` file: `. ./env`.

1. Run `./create.sh $RESOURCEGROUP` to deploy a cluster.

1. To inspect the OpenShift master pods, run `KUBECONFIG=aks/admin.kubeconfig oc
   get pods -n $RESOURCEGROUP`.

1. To inspect pods running on the OpenShift cluster, run
   `KUBECONFIG=_data/_out/admin.kubeconfig oc get pods -n $RESOURCEGROUP`.

1. To ssh into an OpenShift node (vm-infra-0 or vm-compute-0), run
   `./ssh.sh hostname`.

1. Run `./delete.sh` to delete the deployed cluster.
