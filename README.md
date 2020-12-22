# openshift-azure


[![Coverage Status](https://codecov.io/gh/openshift/openshift-azure/branch/master/graph/badge.svg)](https://codecov.io/gh/openshift/openshift-azure)
[![Go Report Card](https://goreportcard.com/badge/github.com/openshift/openshift-azure)](https://goreportcard.com/report/github.com/openshift/openshift-azure)
[![GoDoc](https://godoc.org/github.com/openshift/openshift-azure?status.svg)](https://godoc.org/github.com/openshift/openshift-azure)

## Prerequisites

Note that this README is targeted at AOS-Azure contributors. If you are not a
member of this team, these instructions may not work as they will assume you
have permissions that you may not have.

1. **Utilities**.  Install the following:
   1. [Golang 1.11.6](https://golang.org/dl) (can also use package manager)
   1. Latest [Azure
      CLI](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli)
   1. [OpenShift Origin 3.11 client
      tools](https://github.com/openshift/origin/releases/tag/v3.11.0) (can also
      use package manager)
   1. Latest [Glide](https://github.com/Masterminds/glide/releases).  Note:
      Glide 0.13.1 is known to be broken.
   1. [jq](https://stedolan.github.io/jq/) (can also use package manager)

   Development helper scripts assume an up-to-date GNU tools environment. Recent Linux distros should work out-of-the-box.

   macOS ships with outdated BSD-based tools. We recommend installing [macOS GNU tools](https://www.topbug.net/blog/2013/04/14/install-and-use-gnu-command-line-tools-in-mac-os-x).

1. **Environment variables**.  Ensure that $GOPATH/bin is in your path:

   `export PATH=$PATH:${GOPATH:-$HOME/go}/bin`.

1. **Azure CLI access**.  Log into Azure using the CLI using `az login` and your
   credentials.

1. **OpenShift CI cluster access**.  Log in to the [CI
   cluster](https://api.ci.openshift.org/console/catalog) using `oc login` and a
   token from the CI cluster web interface. You can copy the required command by
   clicking on your username and the "Copy Login Command" option in the web
   portal.

1. **Codebase**.  Check out the codebase:

   `go get github.com/openshift/openshift-azure/...`

1. **Secrets**.  Retrieve cluster creation secrets from the CI cluster:
   ```
   cd ${GOPATH:-$HOME/go}/src/github.com/openshift/openshift-azure
   make secrets
   ```

1. **Environment file**.  Create an environment file:

   `cp env.example env`.

1. **AAD Application / Service principal**.  Create a personal AAD Application:
   1. `hack/aad.sh app-create user-$USER-aad aro-team-shared`
   1. Update env to include the AZURE_AAD_CLIENT_ID and AZURE_AAD_CLIENT_SECRET
      values output by aad.sh.
   1. Ask an AAD administrator to grant permissions to your application.

## Deploy an OpenShift cluster

1. Source the `env` file: `. ./env`.

1. Determine an appropriate resource group name for your cluster (e.g. for a test
   cluster, you could call it `$USER-test`). Then `export RESOURCEGROUP` and run
   `./hack/create.sh $RESOURCEGROUP` to deploy a cluster.

1. Access the web console via the link printed by create.sh, logging in with
   your Azure credentials.

1. To inspect pods running on the OpenShift cluster, run
   `KUBECONFIG=_data/_out/admin.kubeconfig oc get pods`.

1. To ssh into any OpenShift master node, run `./hack/ssh.sh`.  You can directly
   ssh to any other host from the master.  `sudo -i` will give root.

1. Run `./hack/delete.sh` to delete the deployed cluster.

### Examples

Basic OpenShift configuration (also see test/manifests/fakerp/create.yaml):

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

## CI infrastructure
Read more about how to work with our CI system [here](https://github.com/openshift/release/blob/master/projects/azure/README.md).

For any infrastructure-related issues, make sure to contact the Developer Productivity
team who is responsible for managing the OpenShift CI Infrastructure at #forum-testplatform
in [Slack](https://coreos.slack.com/).
