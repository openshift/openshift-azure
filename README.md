# openshift-azure


[![Coverage Status](https://codecov.io/gh/openshift/openshift-azure/branch/master/graph/badge.svg)](https://codecov.io/gh/openshift/openshift-azure)
[![Go Report Card](https://goreportcard.com/badge/github.com/openshift/openshift-azure)](https://goreportcard.com/report/github.com/openshift/openshift-azure)
[![GoDoc](https://godoc.org/github.com/openshift/openshift-azure?status.svg)](https://godoc.org/github.com/openshift/openshift-azure)

## Prerequisites

1. **Utilities**.  You'll need recent versions of [Azure
   CLI](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli) and
   [Golang](https://golang.org/dl) installed.

1. Check out the codebase
   1. If running Linux, ensure you have the systemd-devel RPM installed: `sudo
      dnf -y install systemd-devel`.

   1. Ensure that $GOPATH/bin is in your path: `export
      PATH=$PATH:${GOPATH:-$HOME/go}/bin`.

   1. Check out the codebase: `go get github.com/openshift/openshift-azure/...`.

   1. Navigate to the codebase directory: `cd
      ${GOPATH:-$HOME/go}/src/github.com/openshift/openshift-azure`.

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
   Origin 3.11 on Azure (Staged)` and click the result.  At the bottom of the
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

   1. (optional) For AAD Web-UI sign-in integration to work we will need to have
      second AAD Web-App created, with callback url to OpenShift and right
      permissions enabled. `hack/aad.sh` can help you to do so.

      AAD WebApp Flow:
      1. Create an application (you can use `hack/aad.sh` to create app with
      right permissions)
      2. Add `$AZURE_AAD_CLIENT_ID` variable with application ID to `env` file.
      3. Create the cluster. `create.sh` script will update your application with
      required details.
      4. Get your application permissions approved by organization administrator.
      Without approval cluster will start, just login will not work.

  Once you have application with approved/granted permissions it can be re-used
  for all future clusters.

## Deploy an OpenShift cluster

1. Copy the `env.example` file to `env` and edit according to your requirements.
   Source the `env` file: `. ./env`.

1. Run `./hack/create.sh $RESOURCEGROUP` to deploy a cluster.

1. To inspect pods running on the OpenShift cluster, run
   `KUBECONFIG=_data/_out/admin.kubeconfig oc get pods`.

1. To ssh into any OpenShift master node, run
   `./hack/ssh.sh`. You will be able to jump to other hosts from there.

1. Run `./hack/delete.sh` to delete the deployed cluster.

## Access the cluster
A cluster can be accessed via the `UI` or `CLI`. If it was created using AAD
integration ([Pre-requisites](#prerequisites) 7.iv), you can login using Azure AD. Another option,
which will be deprecated in the future, is `htpasswd`. The username that is used
is `osadmin` and the password is randomly generated. To get the password execute:
```console
./hack/config.sh get-config $RESOURCEGROUP | jq -r .config.adminPasswd
```
You can also get the admin kubeconfig with:
```console
./hack/config.sh get-config $RESOURCEGROUP | jq -r .config.adminKubeconfig
```

### Examples

Basic OpenShift configuration:

```yaml
name: openshift
location: $AZURE_REGION
properties:
  openShiftVersion: v3.11
  authProfile:
    identityProviders:
    - name: Azure AD
      provider:
        kind: AADIdentityProvider
        clientId: $AZURE_AAD_CLIENT_ID
        secret: $AZURE_AAD_CLIENT_SECRET
        tenantId: $AZURE_TENANT_ID
  networkProfile:
    vnetCidr: 10.0.0.0/8
  masterPoolProfile:
    count: 3
    vmSize: Standard_D2s_v3
    subnetCidr: 10.0.0.0/24
  agentPoolProfiles:
  - name: infra
    role: infra
    count: 3
    vmSize: Standard_D2s_v3
    subnetCidr: 10.0.0.0/24
    osType: Linux
  - name: compute
    role: compute
    count: 1
    vmSize: Standard_D2s_v3
    subnetCidr: 10.0.0.0/24
    osType: Linux
```

## Dependency management
To add a new dependency to the project, add the package information to glide.yaml and execute `glide up -v`

## CI infrastructure
Read more about how to work with our CI system [here](https://github.com/openshift/release/blob/master/projects/azure/README.md).

For any infrastructure-related issues, make sure to contact the Developer Productivity
team who is responsible for managing the OpenShift CI Infrastructure at #forum-testplatform
in [Slack](https://coreos.slack.com/).
