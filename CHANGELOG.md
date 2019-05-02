# v4.0

## Canary app ([#1260](https://github.com/openshift/openshift-azure/pull/1260), [@mjudeikis](https://github.com/mjudeikis), 20/03/2019)

Add canary application to track cluster health from end-user perspective when upgrading


## Add release notes generator ([#1311](https://github.com/openshift/openshift-azure/pull/1311), [@y-cote](https://github.com/y-cote), 21/03/2019)

- Add the cmd/releasenotes/ program to scrape merge commits for PRs and generate a release note log to be included in release Changelog.md file.


## remove deepcopy otherwise sas uri not available when vmss are rotated ([#1316](https://github.com/openshift/openshift-azure/pull/1316), [@jim-minter](https://github.com/jim-minter), 21/03/2019)

resolve bug whereby SAS URIs were not being calculated correctly during the vmss rotation procedure


## Monitoring shim for updates ([#1300](https://github.com/openshift/openshift-azure/pull/1300), [@mjudeikis](https://github.com/mjudeikis), 21/03/2019)

Add fakeRP blackbox monitoring tool/monitoring shim


## prevent update from succeeding unless cluster version matches plugin version ([#1320](https://github.com/openshift/openshift-azure/pull/1320), [@jim-minter](https://github.com/jim-minter), 21/03/2019)

prevent update from succeeding unless cluster version matches plugin version


## make hash invariant of SAS URIs ([#1329](https://github.com/openshift/openshift-azure/pull/1329), [@jim-minter](https://github.com/jim-minter), 22/03/2019)

Fix an issue which was causing node rotations on scale; hardened testing


## splitting out pkg/sync and pkg/startup for multi-version plugin ([#1313](https://github.com/openshift/openshift-azure/pull/1313), [@jim-minter](https://github.com/jim-minter), 23/03/2019)

Work towards ability to update clusters which are on a supported old version
with respect to the RP/plugin.

* Set master branch to work towards v4.0; we will reset this to v5.0 when we
  branch release-v4

* Make pluginconfig support multiple versions in parallel - see
  /pluginconfig/pluginconfig-311.yaml as an example

* Rename pkg/addons -> pkg/sync, simplify to an interface and make multiversion

* Split pkg/arm -> pkg/startup, simplify to an interface and make multiversion

* Move lint-addons to a unit test

* Reduce dependencies on pkg/cluster in preparation for making it multiversion

* Miscellaneous sync pod performance, behaviour and code style fixes


## update to 311.88.20190322 ([#1337](https://github.com/openshift/openshift-azure/pull/1337), [@jim-minter](https://github.com/jim-minter), 25/03/2019)

update OpenShift to 3.11.88


## make pkg/arm and pkg/config version aware ([#1331](https://github.com/openshift/openshift-azure/pull/1331), [@jim-minter](https://github.com/jim-minter), 26/03/2019)

* make pkg/arm and pkg/config version aware
* ensure update functionality for cluster versions >= 3.0
* allow admin to write "latest" to plugin version to upgrade cluster
* allow admin to scale up infra nodes from 2->3
* remove RunningUnderTest field in config struct - no longer needed as sync/startup pods don't validate input now


## Allow v3 on v4 ([#1348](https://github.com/openshift/openshift-azure/pull/1348), [@jim-minter](https://github.com/jim-minter), 26/03/2019)

Allow v3.2 clusters to be supported by newer master


## v2019-04-30 GA API ([#1346](https://github.com/openshift/openshift-azure/pull/1346), [@mjudeikis](https://github.com/mjudeikis), 28/03/2019)

- Add GA API v2019-04-30
- Move converters inside api sub-packages and rename them to be more informative.
foo.ConvertTo->foo.FromInternal; foo.ConvertFrom->foo.ToInternal
- Collapse api packages to one directory:  github.com/openshift/openshift-azure/pkg/api/2019-04-30/api -> `github.com/openshift/openshift-azure/pkg/api/2019-04-30`


## restrict #pods/node to 50 for now ([#1383](https://github.com/openshift/openshift-azure/pull/1383), [@jim-minter](https://github.com/jim-minter), 01/04/2019)

restrict #pods/node to 50 as a starting point for GA


## make TestConfig optional in NewPlugin ([#1394](https://github.com/openshift/openshift-azure/pull/1394), [@jim-minter](https://github.com/jim-minter), 02/04/2019)

make TestConfig optional in NewPlugin


# v4.1

## latch all configurables that come from the plugin config ([#1407](https://github.com/openshift/openshift-azure/pull/1407), [@jim-minter](https://github.com/jim-minter), 04/04/2019)

* latch all configurables that come from the plugin config, such that a plugin config change won't cause unexpected cluster rotations
* secret rotation geneva action stub re-reads geneva certs from plugin config, among resetting a few other secrets


## Export 'Log' from the SanityChecker struct. ([#1422](https://github.com/openshift/openshift-azure/pull/1422), [@y-cote](https://github.com/y-cote), 04/04/2019)

- E2E: standard exports the Log field from the SanityChecker for better reusability


## add geneva action to list backup blobs ([#1432](https://github.com/openshift/openshift-azure/pull/1432), [@jim-minter](https://github.com/jim-minter), 05/04/2019)

* add geneva action to list backup blobs


## allow admin API to change image versions ([#1456](https://github.com/openshift/openshift-azure/pull/1456), [@jim-minter](https://github.com/jim-minter), 08/04/2019)

make vm image configuration and individual container image configuration writeable via the admin API


## validate canary image empty on v3.2 ([#1477](https://github.com/openshift/openshift-azure/pull/1477), [@jim-minter](https://github.com/jim-minter), 09/04/2019)

validate canary image empty on v3.2


# v4.2

## fluentd rollback and cluster upgrade integration fixes ([#1483](https://github.com/openshift/openshift-azure/pull/1483), [@jim-minter](https://github.com/jim-minter), 09/04/2019)

* roll back change which caused NPE in fluentd for non-container logs
* modify cluster upgrade mechanism - set clusterVersion: latest, not pluginVersion: latest.  PluginVersion remains in admin API, but r/o


# v4.3

## prevent "Job for docker.service canceled" error on startup ([#1604](https://github.com/openshift/openshift-azure/pull/1604), [@jim-minter](https://github.com/jim-minter), 01/05/2019)

prevent "Job for docker.service canceled" error on startup

