## v7.0

- Azure Security Pack support ([#1817](https://github.com/openshift/openshift-azure/pull/1817), [@thekad](https://github.com/thekad), 12/08/2019)
    * Moved Geneva logging stack from openshift (running as a daemonset in the `openshift-azure-logging` namespace) to run in the hosts themselves as systemd services
    * Added Azure Security Pack logging to all hosts
    * Using fluentd from RHEL repos instead of td-agent
- Base VM image now has mdsd/azsecd/azsecmond installed (but disabled) by default ([#1838](https://github.com/openshift/openshift-azure/pull/1838), [@thekad](https://github.com/thekad), 09/08/2019)
- Bugfix: ensure that Geneva and PackageRepository certificates can be updated once set. ([#1831](https://github.com/openshift/openshift-azure/pull/1831), [@asalkeld](https://github.com/asalkeld), 06/08/2019)
- ETCD backups are now run prior to updating ([#1796](https://github.com/openshift/openshift-azure/pull/1796), [@asalkeld](https://github.com/asalkeld), 29/07/2019)
