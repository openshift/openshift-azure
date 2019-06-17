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
