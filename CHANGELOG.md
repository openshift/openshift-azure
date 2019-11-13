## v12.0

- If the vnet nameservers have changed only progress if RefreshCluster is set ([#2050](https://github.com/openshift/openshift-azure/pull/2050), [@asalkeld](https://github.com/asalkeld), 12/11/2019)
- Add Nameservers to the internal API : used to store the currently used nameservers ([#2050](https://github.com/openshift/openshift-azure/pull/2050), [@asalkeld](https://github.com/asalkeld), 12/11/2019)
- Add RefreshCluster to the latest public API ([#2050](https://github.com/openshift/openshift-azure/pull/2050), [@asalkeld](https://github.com/asalkeld), 12/11/2019)
- Don't update the vnet only create it (to prevent conflicting updates by the customer) ([#2050](https://github.com/openshift/openshift-azure/pull/2050), [@asalkeld](https://github.com/asalkeld), 12/11/2019)
- Resolve customer-admin group memberships issue when case of AAD mail domain differs from case of AAD login ([#2063](https://github.com/openshift/openshift-azure/pull/2063), [@jim-minter](https://github.com/jim-minter), 11/11/2019)
- Upgrade log analytics to version ciprod:ciprod10182019 ([#2052](https://github.com/openshift/openshift-azure/pull/2052), [@olga-mir](https://github.com/olga-mir), 06/11/2019)


