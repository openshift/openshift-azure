## Release v3.0

- add missing master role definition "Microsoft.Network/loadBalancers/backendAddressPools/join/action" ([#1301](https://github.com/openshift/openshift-azure/pull/1301), [@jim-minter](https://github.com/jim-minter))
- Enable ETCD metrics via tls-proxy container  ([#1176](https://github.com/openshift/openshift-azure/pull/1176), [@mjudeikis](https://github.com/mjudeikis))
- support different master and worker service principals https://github.com/openshift/openshift-azure/pull/1226 has example role definitions which can be used for these. ([#1226](https://github.com/openshift/openshift-azure/pull/1226), [@asalkeld](https://github.com/asalkeld))
- api.PluginConfig is removed, in prod RP api.TestConfig{} should be passed to NewPlugin() instead. ([#1261](https://github.com/openshift/openshift-azure/pull/1261), [@jim-minter](https://github.com/jim-minter))
- A new container image with httpd is used to serve the CSS stylesheet for the web console branding.  ([#1219](https://github.com/openshift/openshift-azure/pull/1219), [@y-cote](https://github.com/y-cote))
- improves version control of images laid down by operators.  Some straightforward changes to the plugin config format, need to check the RP config file ([#1239](https://github.com/openshift/openshift-azure/pull/1239), [@mjudeikis](https://github.com/mjudeikis))
- updates plugin to expect FQDNs, Public{Hostname,Subdomain} set by RP and to honour these ([#1251](https://github.com/openshift/openshift-azure/pull/1251), [@jim-minter](https://github.com/jim-minter))
- improvements in validation: with externalOnly set false, plugin Validate() should now validate all RP-set fields in addition to user-set fields (which it did before). ([#1243](https://github.com/openshift/openshift-azure/pull/1243), [@jim-minter](https://github.com/jim-minter))
- increase infra node count to 3. Important: CLI needs to be updated to reflect this. ([#1246](https://github.com/openshift/openshift-azure/pull/1246), [@mjudeikis](https://github.com/mjudeikis))
- Add egressnetworkpolicies to customer admin.  Added tests for limitranges, resourceQuotas, and egressnetworkpolicies. ([#1221](https://github.com/openshift/openshift-azure/pull/1221), [@kwoodson](https://github.com/kwoodson))
- adds rate calculation to metricsbridge ([#1235](https://github.com/openshift/openshift-azure/pull/1235), [@Makdaam](https://github.com/Makdaam))
- plugin now reads apiserver and router certificates from keyvault.  API changes: ([#1192](https://github.com/openshift/openshift-azure/pull/1192), [@asalkeld](https://github.com/asalkeld))
- add ca-bundle support for external certificates ([#1270](https://github.com/openshift/openshift-azure/pull/1270), [@mjudeikis](https://github.com/mjudeikis))
- split the config blob into 3 blobs:
  * the sync blob (read by the sync pod; just a rename of the blob)
  * the master-startup blob (read by the startup pod on masters) - identical to
    the sync blob but changed at a different point in the update process
  * the worker-startup blob (read by the startup pod on workers) - whitelisted
    so that workers can't see secrets they shouldn't
    this is done (a) so that we can completely separate the process of updating
    nodes and the sync pod, (b) to homogenise the startup process so that the
    startup pod is used in all cases ([#1282](https://github.com/openshift/openshift-azure/pull/1282), [@jim-minter](https://github.com/jim-minter))
- modify the startup pod so that it runs on both masters and workers, reading
  its blob (as appropriate) via a SAS URL (on masters, the startup pod also
  reads certificates from the key vault) ([#1282](https://github.com/openshift/openshift-azure/pull/1282), [@jim-minter](https://github.com/jim-minter))
- modify the sync pod so that it detects readiness of components it manages,
  including Routes, instead of the plugin doing this.  Hopefully this will
  provide a clearer delineation between the plugin and sync pod to make
  development easier post-GA ([#1282](https://github.com/openshift/openshift-azure/pull/1282), [@jim-minter](https://github.com/jim-minter))
- use static pods instead of DaemonSets for OVS and SDN pods.  This makes the
  path to master Nodes being Ready simpler, and stops the sync pod being
  essential for that to happen ([#1282](https://github.com/openshift/openshift-azure/pull/1282), [@jim-minter](https://github.com/jim-minter))
- run the sync pod as a Deployment ([#1282](https://github.com/openshift/openshift-azure/pull/1282), [@jim-minter](https://github.com/jim-minter))
- remove the openshift-node DaemonSet - it is completely unnecessary within the
  OSA management model ([#1282](https://github.com/openshift/openshift-azure/pull/1282), [@jim-minter](https://github.com/jim-minter))
- fix a serious Plugin architectural issue: the Plugin (which is theoretically
  meant to be reusable for concurrent calls against multiple clusters) was
  embedding the clusterUpdater, which implicitly had clients bound to a single
  *OpenShiftManagedCluster.  This could have caused corruption in cases where
  calls against different clusters had been made against a single Plugin
  instance ([#1282](https://github.com/openshift/openshift-azure/pull/1282), [@jim-minter](https://github.com/jim-minter))
- as a knock-on effect of the above, simplify client creation code (remove
  Initialize, CreateClients) ([#1282](https://github.com/openshift/openshift-azure/pull/1282), [@jim-minter](https://github.com/jim-minter))
- introduce code to determine when the sync pod needs to be reconfigured and
  restarted (via hashing its output).  Correct sync pod reconfiguration should
  now in principle be possible without VM rotation ([#1282](https://github.com/openshift/openshift-azure/pull/1282), [@jim-minter](https://github.com/jim-minter))
- move as much code out of the master and node startup scripts as possible and
  into the startup container.  A little more can be done on this front, moving
  customisations into the VM image ([#1282](https://github.com/openshift/openshift-azure/pull/1282), [@jim-minter](https://github.com/jim-minter))
- Adds a new "azure-file" storageClass to support azureFile persistent storage as per https://docs.openshift.com/container-platform/3.11/install_config/persistent_storage/persistent_storage_azure_file.html ([#1262](https://github.com/openshift/openshift-azure/pull/1262), [@thekad](https://github.com/thekad))
- 311.82.20190311 Image release ([#1277](https://github.com/openshift/openshift-azure/pull/1277), [@jim-minter](https://github.com/jim-minter))
- Remove customer reader capability, including from all APIs ([#1242](https://github.com/openshift/openshift-azure/pull/1242), [@jim-minter](https://github.com/jim-minter))
