# Initiating Plugin - Versioning model

When initiating plugin code and calling `GenerateConfig` method we need to
pass in `Plugin Template/PluginConfig` as a parameter. 

Plugin template defines main characteristics for the cluster. In example 
cluster, images versions, VM image, external integration details.

Config file examples can be found in `pluginconfig/pluginconfig-{version}.yaml`, but ideally config object should be constructed using `pkg/api/plugin/config.go Config` struct.

`openshift-azure` repository code always contains stable newest version of file references.

Before updating example file and code base for new release aka. code cut for MSFT RP, make sure you tag code with corresponding release tag version

```
master--311.43.20181121--*--R1----*--R2------R3---
                         /       /
update1--311.51.20190103-       /
                               /
update2--311.85.20180205 ------
```

R1 tag -> `311.43.2018112`
R2 tag -> `311.51.20190103`
R3 tag should be `311.85.20180205` if no new image version will be released before code cut. 


Where:
* R1 will be reference to released 3.11.43 code base 
* R2 will be reference to released 3.11.51 code base. 

Note: Branch updates with newer version of the code base might (and in most cases will not) match with release tag. This is because in the end we are interested only in code base running in realRP.

This will allow us to create same version of the cluster using fakeRP by
check-out previous version of the code. And check how cluster, created
using older version of the code, will interact with newer version of the 
code during upgrades.

If needed, previous release might be branched into separate branch to enable additional code fixes. But this should not be practice.
