# openshift-azure V4



## Create

To create a cluster follow instructions below

set Image override:
```
export OPENSHIFT_INSTALL_OS_IMAGE_OVERRIDE="/resourceGroups/rhcosimages/providers/Microsoft.Compute/images/rhcos-410.8.20190504.0-azure.vhd"
```

set env variables:
```
source secrets/secret
```

set RESOURCEGROUP variable:
```
export RESOURCEGROUP=clustername
```

create a cluster:
```
make create
```

## SSH into the VM

First, cluster has to be created with `RUNNING_UNDER_TEST` flag to have ssh enabled.
Export resource group variable:
```
export RESOURCEGROUP=clustername
```

SSH into Bastion: `./hack/ssh.sh`
SSH into Master0: `./hack.ssh.sh 1`
