# v5.2

## Kernel security fix

* release new VM image

# v5.1

## Security fix for Microarchitectural Data Sampling (MDS)

* release new VM image

# v5.0

## latch all configurables that come from the plugin config ([#1407](https://github.com/openshift/openshift-azure/pull/1407), [@jim-minter](https://github.com/jim-minter), 04/04/2019)

* latch all configurables that come from the plugin config, such that a plugin config change won't cause unexpected cluster rotations
* secret rotation geneva action stub re-reads geneva certs from plugin config, among resetting a few other secrets


## Export 'Log' from the SanityChecker struct. ([#1422](https://github.com/openshift/openshift-azure/pull/1422), [@y-cote](https://github.com/y-cote), 04/04/2019)

- E2E: standard exports the Log field from the SanityChecker for better reusability


## add geneva action to list backup blobs ([#1432](https://github.com/openshift/openshift-azure/pull/1432), [@jim-minter](https://github.com/jim-minter), 05/04/2019)

* add geneva action to list backup blobs


## allow admin API to change image versions ([#1456](https://github.com/openshift/openshift-azure/pull/1456), [@jim-minter](https://github.com/jim-minter), 08/04/2019)

make vm image configuration and individual container image configuration writeable via the admin API


## fluentd config for split kubernetes logs ([#1406](https://github.com/openshift/openshift-azure/pull/1406), [@mjudeikis](https://github.com/mjudeikis), 09/04/2019)

Geneva logging improvement


## fluentd rollback and cluster upgrade integration fixes ([#1483](https://github.com/openshift/openshift-azure/pull/1483), [@jim-minter](https://github.com/jim-minter), 10/04/2019)

* roll back change which caused NPE in fluentd for non-container logs
* modify cluster upgrade mechanism - set clusterVersion: latest, not pluginVersion: latest.  PluginVersion remains in admin API, but r/o


## validate canary image empty on v3.2 ([#1477](https://github.com/openshift/openshift-azure/pull/1477), [@jim-minter](https://github.com/jim-minter), 11/04/2019)

validate canary image empty on v3.2


## fluentd log split - update with nil check ([#1486](https://github.com/openshift/openshift-azure/pull/1486), [@mjudeikis](https://github.com/mjudeikis), 11/04/2019)

Split kubernetes containers logs in geneva


## don't output secret if validation fails ([#1502](https://github.com/openshift/openshift-azure/pull/1502), [@jim-minter](https://github.com/jim-minter), 12/04/2019)

* don't output secret if validation fails


## Enable cloud API rate limiting for nodes ([#1492](https://github.com/openshift/openshift-azure/pull/1492), [@ehashman](https://github.com/ehashman), 12/04/2019)

Enable cloud API rate limiting for nodes


## collapse image to one image ([#1464](https://github.com/openshift/openshift-azure/pull/1464), [@mjudeikis](https://github.com/mjudeikis), 13/04/2019)

Release a single azure container image instead of seven different ones


## bump v5 to 3.11.98 ([#1496](https://github.com/openshift/openshift-azure/pull/1496), [@jim-minter](https://github.com/jim-minter), 15/04/2019)

bump to OpenShift 3.11.98


## add tool to update imagestreams and templates, and do so ([#1433](https://github.com/openshift/openshift-azure/pull/1433), [@jim-minter](https://github.com/jim-minter), 16/04/2019)

update openshift imagestreams and templates to 04/04/19


## Update insights import ([#1526](https://github.com/openshift/openshift-azure/pull/1526), [@jim-minter](https://github.com/jim-minter), 17/04/2019)

bump insights import to a version which is available both in azure sdk v24 and azure sdk v26


## increase maximum nodes to 30 ([#1528](https://github.com/openshift/openshift-azure/pull/1528), [@jim-minter](https://github.com/jim-minter), 17/04/2019)

Increase maximum node count to 30


## Add validation for cluster names and locations ([#1525](https://github.com/openshift/openshift-azure/pull/1525), [@ehashman](https://github.com/ehashman), 17/04/2019)

Validate cluster names to ensure they will comply with the DNS spec.
Validate location names to ensure they match the standard naming scheme.


## Branding: select 'azure' branding option for cluster console. ([#1541](https://github.com/openshift/openshift-azure/pull/1541), [@y-cote](https://github.com/y-cote), 17/04/2019)

- Web interface completely re-branded as Azure Red Hat OpenShift


## Adding geneva command to restart docker and tests ([#1560](https://github.com/openshift/openshift-azure/pull/1560), [@kwoodson](https://github.com/kwoodson), 25/04/2019)

Adding run command to restart docker


## Add default values to external APIs ([#1548](https://github.com/openshift/openshift-azure/pull/1548), [@ehashman](https://github.com/ehashman), 25/04/2019)

Set some default values (e.g. RouterProfile) when not specified upon creating a new cluster config


## Adding session affinity to router service loadbalancer. ([#1558](https://github.com/openshift/openshift-azure/pull/1558), [@kwoodson](https://github.com/kwoodson), 25/04/2019)

Update router service load balancer to use session affinity


## Default MasterProfilePool count to 3 ([#1579](https://github.com/openshift/openshift-azure/pull/1579), [@ehashman](https://github.com/ehashman), 26/04/2019)

Default MasterProfilePool count to 3 when not specified in new cluster configs


## check for role equality in admin update validation ([#1580](https://github.com/openshift/openshift-azure/pull/1580), [@jim-minter](https://github.com/jim-minter), 26/04/2019)

check for role equality in admin update validation, otherwise an error occurs when attempting to admin update legacy clusters with only 2 infra nodes'


## Master config fixups ([#1594](https://github.com/openshift/openshift-azure/pull/1594), [@jim-minter](https://github.com/jim-minter), 30/04/2019)

Retry on API server ECONNREFUSED errors


## Add some default settings for AgentPoolProfiles ([#1588](https://github.com/openshift/openshift-azure/pull/1588), [@ehashman](https://github.com/ehashman), 01/05/2019)

Add a default AgentPoolProfile with name and role "infra" when not specified in new cluster configs
Set default Count to 3 for the "infra" AgentPoolProfile
Set default OSType to Linux in AgentPoolProfiles


## prevent "Job for docker.service canceled" error on startup ([#1604](https://github.com/openshift/openshift-azure/pull/1604), [@jim-minter](https://github.com/jim-minter), 01/05/2019)

prevent "Job for docker.service canceled" error on startup


