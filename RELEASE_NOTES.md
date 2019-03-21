# Release v3.0

## Use a vault to store public certs ([#1192](https://github.com/openshift/openshift-azure/pull/1192), [@asalkeld](https://github.com/asalkeld), 03/03/2019)

plugin now reads apiserver and router certificates from keyvault.  API changes:
1. Properties.ServicePrincipalProfile removed in favour of Properties.MasterServicePrincipalProfile and Properties.WorkerServicePrincipalProfile.
2. Properties.APICertProfile and Properties.RouterProfiles.CertProfile added, must contain links to applicable keyvault secrets.
3. ContextKeyVaultClientAuthorizer context key added: must be set to an Authorizer which can read secrets from the customer keyvault (see azureclient.NewAuthorizer)


## metricsbridge - adding an option to calculate rate of a metric ([#1235](https://github.com/openshift/openshift-azure/pull/1235), [@Makdaam](https://github.com/Makdaam), 05/03/2019)

- adds rate calculation to metricsbridge


## remove customer-reader capability, will pursue alternative arrangement post-GA ([#1242](https://github.com/openshift/openshift-azure/pull/1242), [@jim-minter](https://github.com/jim-minter), 05/03/2019)

Remove customer reader capability, including from all APIs


## Adding egress ability for customer admin.  labeling default namespace with default. ([#1221](https://github.com/openshift/openshift-azure/pull/1221), [@kwoodson](https://github.com/kwoodson), 05/03/2019)

Add egressnetworkpolicies to customer admin.  Added tests for limitranges, resourceQuotas, and egressnetworkpolicies.


## increase infra node count 3 ([#1246](https://github.com/openshift/openshift-azure/pull/1246), [@mjudeikis](https://github.com/mjudeikis), 06/03/2019)

MSFT: increase infra node count to 3. Important: CLI needs to be updated to reflect this.


## Validate updates ([#1243](https://github.com/openshift/openshift-azure/pull/1243), [@jim-minter](https://github.com/jim-minter), 07/03/2019)

MSFT: improvements in validation: with externalOnly set false, plugin Validate() should now validate all RP-set fields in addition to user-set fields (which it did before).


## DNS updates for TLS work ([#1251](https://github.com/openshift/openshift-azure/pull/1251), [@jim-minter](https://github.com/jim-minter), 07/03/2019)

MSFT: updates plugin to expect FQDNs, Public{Hostname,Subdomain} set by RP and to honour these


## Move pluginConfig to "master" and enable more granular base image versioning  ([#1239](https://github.com/openshift/openshift-azure/pull/1239), [@mjudeikis](https://github.com/mjudeikis), 07/03/2019)

MSFT: improves version control of images laid down by operators.  Some straightforward changes to the plugin config format, need to check the RP config file


## run command geneva action ([#1236](https://github.com/openshift/openshift-azure/pull/1236), [@jim-minter](https://github.com/jim-minter), 07/03/2019)

MSFT: provides run command geneva action for kubelet and networkmanager restart


## Rename ClusterVersion -> PluginVersion, add ClusterVersion ([#1257](https://github.com/openshift/openshift-azure/pull/1257), [@jim-minter](https://github.com/jim-minter), 08/03/2019)

MSFT: renames existing Config.ClusterVersion -> Config.PluginVersion; adds additional Properties.ClusterVersion field


## Serve Azure Red Hat OpenShift CSS logo. ([#1219](https://github.com/openshift/openshift-azure/pull/1219), [@y-cote](https://github.com/y-cote), 09/03/2019)

- A new container image with httpd is used to serve the CSS stylesheet for the web console branding.


## Geneva action get rp plugin versions ([#1261](https://github.com/openshift/openshift-azure/pull/1261), [@jim-minter](https://github.com/jim-minter), 11/03/2019)

MSFT: api.PluginConfig is removed, in prod RP api.TestConfig{} should be passed to NewPlugin() instead.
MSFT: plugin template (from pluginconfig.yaml) should now be passed solely to NewPlugin() instead of to ValidatePluginTemplate(), GenerateConfig(), RotateClusterSecrets().


## use new master and worker service principals ([#1226](https://github.com/openshift/openshift-azure/pull/1226), [@asalkeld](https://github.com/asalkeld), 13/03/2019)

MSFT: support different master and worker service principals https://github.com/openshift/openshift-azure/pull/1226 has example role definitions which can be used for these.


## update to 311.82.20190311 ([#1277](https://github.com/openshift/openshift-azure/pull/1277), [@jim-minter](https://github.com/jim-minter), 13/03/2019)

311.82.20190311 Image release


## Azure file support ([#1262](https://github.com/openshift/openshift-azure/pull/1262), [@thekad](https://github.com/thekad), 13/03/2019)

Adds a new "azure-file" storageClass to support azureFile persistent storage as per https://docs.openshift.com/container-platform/3.11/install_config/persistent_storage/persistent_storage_azure_file.html


## CA Bundle for certs ([#1270](https://github.com/openshift/openshift-azure/pull/1270), [@mjudeikis](https://github.com/mjudeikis), 13/03/2019)

Add router and OpenShift console certificate chain support
Add CA-Bundle and add external certificates to it.
Configure SA to use CA-Bundle for e2e trust


## etcd metrics shim ([#1176](https://github.com/openshift/openshift-azure/pull/1176), [@mjudeikis](https://github.com/mjudeikis), 14/03/2019)

Internal: Enable ETCD metrics via tls-proxy container


## add missing role, needed for ILBs ([#1301](https://github.com/openshift/openshift-azure/pull/1301), [@jim-minter](https://github.com/jim-minter), 19/03/2019)

MSFT: add missing master role definition "Microsoft.Network/loadBalancers/backendAddressPools/join/action"


## many improvements for reconfiguration ([#1282](https://github.com/openshift/openshift-azure/pull/1282), [@jim-minter](https://github.com/jim-minter), 19/03/2019)

* split the config blob into 3 blobs:
  * the sync blob (read by the sync pod; just a rename of the blob)

  * the master-startup blob (read by the startup pod on masters) - identical to
    the sync blob but changed at a different point in the update process

  * the worker-startup blob (read by the startup pod on workers) - whitelisted
    so that workers can't see secrets they shouldn't

  this is done (a) so that we can completely separate the process of updating
  nodes and the sync pod, (b) to homogenise the startup process so that the
  startup pod is used in all cases

* modify the startup pod so that it runs on both masters and workers, reading
  its blob (as appropriate) via a SAS URL (on masters, the startup pod also
  reads certificates from the key vault)

* modify the sync pod so that it detects readiness of components it manages,
  including Routes, instead of the plugin doing this.  Hopefully this will
  provide a clearer delineation between the plugin and sync pod to make
  development easier post-GA

* use static pods instead of DaemonSets for OVS and SDN pods.  This makes the
  path to master Nodes being Ready simpler, and stops the sync pod being
  essential for that to happen

* run the sync pod as a Deployment

* remove the openshift-node DaemonSet - it is completely unnecessary within the
  OSA management model

* fix a serious Plugin architectural issue: the Plugin (which is theoretically
  meant to be reusable for concurrent calls against multiple clusters) was
  embedding the clusterUpdater, which implicitly had clients bound to a single
  *OpenShiftManagedCluster.  This could have caused corruption in cases where
  calls against different clusters had been made against a single Plugin
  instance

* as a knock-on effect of the above, simplify client creation code (remove
  Initialize, CreateClients)

* introduce code to determine when the sync pod needs to be reconfigured and
  restarted (via hashing its output).  Correct sync pod reconfiguration should
  now in principle be possible without VM rotation

* move as much code out of the master and node startup scripts as possible and
  into the startup container.  A little more can be done on this front, moving
  customisations into the VM image


# Release v3.1

## remove deepcopy otherwise sas uri not available when vmss are rotated ([#1316](https://github.com/openshift/openshift-azure/pull/1316), [@jim-minter](https://github.com/jim-minter), 21/03/2019)

resolve bug whereby SAS URIs were not being calculated correctly during the vmss rotation procedure


## prevent update from succeeding unless cluster version matches plugin version ([#1320](https://github.com/openshift/openshift-azure/pull/1320), [@jim-minter](https://github.com/jim-minter), 21/03/2019)

prevent update from succeeding unless cluster version matches plugin version
