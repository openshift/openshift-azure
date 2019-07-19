# v6.0

## Release prep (4.4, 5.1) ([#1645](https://github.com/openshift/openshift-azure/pull/1645), [@mjudeikis](https://github.com/mjudeikis), 16/05/2019)

Prepare for release v4.4, v5.1
Add new VM image with security updates 


## Make some api properties updatable ([#1629](https://github.com/openshift/openshift-azure/pull/1629), [@asalkeld](https://github.com/asalkeld), 23/05/2019)

Properties.AuthProfile.IdentityProviders[x].Provider and Properties.AgentPoolProfiles[x].VMSize can now be changed by the user using CreateOrUpdate.


## removing v3 from the openshift-azure code ([#1693](https://github.com/openshift/openshift-azure/pull/1693), [@Makdaam](https://github.com/Makdaam), 03/06/2019)

MSFT: Removing references to v3 from code


## Revert docker build strategy restriction ([#1742](https://github.com/openshift/openshift-azure/pull/1742), [@ehashman](https://github.com/ehashman), 24/06/2019)

Add support for docker build strategy


## image hardening ([#1671](https://github.com/openshift/openshift-azure/pull/1671), [@kwoodson](https://github.com/kwoodson), 25/06/2019)

Added Auditd


## Adding FIPS mode ([#1749](https://github.com/openshift/openshift-azure/pull/1749), [@kwoodson](https://github.com/kwoodson), 27/06/2019)

Enable FIPS


## Update allowed cluster names to match ARM validation ([#1756](https://github.com/openshift/openshift-azure/pull/1756), [@ehashman](https://github.com/ehashman), 29/06/2019)

Update cluster name validation to match ARM RG regex


## Add support for etcd ca rotation ([#1679](https://github.com/openshift/openshift-azure/pull/1679), [@asalkeld](https://github.com/asalkeld), 02/07/2019)

Microsoft: Add geneva actions for rotating certificates and certificatesAndSecrets.


## group sync fix to include external users ([#1768](https://github.com/openshift/openshift-azure/pull/1768), [@mjudeikis](https://github.com/mjudeikis), 05/07/2019)

fix group sync for guest accounts


## adding permissions to the customer-admin-cluster and cusomer-admin-project roles ([#1776](https://github.com/openshift/openshift-azure/pull/1776), [@Makdaam](https://github.com/Makdaam), 05/07/2019)

Customer admin role gains the following permissions
- creating and managing daemonsets in customer created projects
- listing and deleting existing OAuth client authorizations
- SAR and RAR checks for user permission verification
- listing and viewing nodes, events, pods, cluster networks, netnamespaces for more low level insight
- listing SCCs
- listing available images and image tags
- listing all builds and build configs in a cluster


## Removes release v4 ([#1784](https://github.com/openshift/openshift-azure/pull/1784), [@m1kola](https://github.com/m1kola), 08/07/2019)

Release v4 was removed


## Rapid mitigation of CVEs with local yum updates ([#1787](https://github.com/openshift/openshift-azure/pull/1787), [@charlesakalugwu](https://github.com/charlesakalugwu), 17/07/2019)

Add support for cluster hot patches in response to CVEs


## Add project-request, self-provisioner, managed-resource capabilities ([#1783](https://github.com/openshift/openshift-azure/pull/1783), [@mjudeikis](https://github.com/mjudeikis), 18/07/2019)

Add the ability to exclude certain objects from azure controller reconciliation loop.
Add project-request template into openshift namespace and enable customer-admin to change it
Add ability to modify self-provisioner clusterrolebinding for cluster-admin. 
Add ability to modify shared resources in openshift namespace
