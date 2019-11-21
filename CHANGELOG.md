## v12.2

- Updating image and OpenShift release to 3.11.154 ([#2089](https://github.com/openshift/openshift-azure/pull/2089), [@kwoodson](https://github.com/kwoodson), 20/11/2019)
- Updating to 311.153.20191113 and creating v10.2,v12.1 ([#2075](https://github.com/openshift/openshift-azure/pull/2075), [@kwoodson](https://github.com/kwoodson), 16/11/2019)
- Move GetPrivateAPIServerIPAddress into GenerateConfig ([#2093](https://github.com/openshift/openshift-azure/pull/2093), [@asalkeld](https://github.com/asalkeld), 21/11/2019)
- Update OMS agent to ciprod11012019 ([#2077](https://github.com/openshift/openshift-azure/pull/2077), [@olga-mir](https://github.com/olga-mir), 14/11/2019)
- Remove support for v11 plugin ([#2076](https://github.com/openshift/openshift-azure/pull/2076), [@asalkeld](https://github.com/asalkeld), 14/11/2019)


- If the vnet nameservers have changed only progress if RefreshCluster is set ([#2050](https://github.com/openshift/openshift-azure/pull/2050), [@asalkeld](https://github.com/asalkeld), 12/11/2019)
- Add Nameservers to the internal API : used to store the currently used nameservers ([#2050](https://github.com/openshift/openshift-azure/pull/2050), [@asalkeld](https://github.com/asalkeld), 12/11/2019)
- Add RefreshCluster to the latest public API ([#2050](https://github.com/openshift/openshift-azure/pull/2050), [@asalkeld](https://github.com/asalkeld), 12/11/2019)
- Don't update the vnet only create it (to prevent conflicting updates by the customer) ([#2050](https://github.com/openshift/openshift-azure/pull/2050), [@asalkeld](https://github.com/asalkeld), 12/11/2019)
- Resolve customer-admin group memberships issue when case of AAD mail domain differs from case of AAD login ([#2063](https://github.com/openshift/openshift-azure/pull/2063), [@jim-minter](https://github.com/jim-minter), 11/11/2019)
- Upgrade log analytics to version ciprod:ciprod10182019 ([#2052](https://github.com/openshift/openshift-azure/pull/2052), [@olga-mir](https://github.com/olga-mir), 06/11/2019)


