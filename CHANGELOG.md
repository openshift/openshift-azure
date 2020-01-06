## v14.0

- Add metrics-server ([#2099](https://github.com/openshift/openshift-azure/pull/2099), [@olga-mir](https://github.com/olga-mir), 10/12/2019)
- Update to OpenShift [3.11.157](https://docs.openshift.com/container-platform/3.11/release_notes/ocp_3_11_release_notes.html#ocp-3-11-157) ([#2154](https://github.com/openshift/openshift-azure/pull/2154), [@ehashman](https://github.com/ehashman), 19/12/2019)
- Bugfix: Correct the behaviour of RefreshCluster when updating as non-admin, correct the content of OpenShift resolve.conf ([#2155](https://github.com/openshift/openshift-azure/pull/2155), [@asalkeld](https://github.com/asalkeld), 20/12/2019)
- Bugfix: Add imagestream access to the customer-admin-project cluster role ([#2156](https://github.com/openshift/openshift-azure/pull/2156), [@kwoodson](https://github.com/kwoodson), 20/12/2019)
- Fix user syncing when Mail field is set, but UserPrincipalName matches the cluster username ([#2128](https://github.com/openshift/openshift-azure/pull/2128), [@Makdaam](https://github.com/Makdaam), 12/12/2019)
- Fix bug which prevented adding reconcile-protect annotation to SCCs ([#2134](https://github.com/openshift/openshift-azure/pull/2134), [@Makdaam](https://github.com/Makdaam), 11/12/2019)
- Fix guest user syncing with prefix and no Mail ([#2124](https://github.com/openshift/openshift-azure/pull/2124), [@asalkeld](https://github.com/asalkeld), 10/12/2019)
- Fix customer admin role to allow cluster wide pods read-only operations ([#2127](https://github.com/openshift/openshift-azure/pull/2127), [@olga-mir](https://github.com/olga-mir), 09/12/2019)
- Move Log Analytics cluster agent to compute node ([#2122](https://github.com/openshift/openshift-azure/pull/2122), [@olga-mir](https://github.com/olga-mir), 06/12/2019)
- MSFT: update plugin validator to reject upper-case domain names for public hostname and router. ([#2112](https://github.com/openshift/openshift-azure/pull/2112), [@jim-minter](https://github.com/jim-minter), 02/12/2019)
- Plugin now validates AAD service principal authentication and, if customerAdminGroupID is set, that the Directory.Read.All permission is granted. ([#2109](https://github.com/openshift/openshift-azure/pull/2109), [@jim-minter](https://github.com/jim-minter), 29/11/2019)

## v13.0

- Enable use of whitelisted privileged containers on ARO service ([#2023](https://github.com/openshift/openshift-azure/pull/2023), [@Makdaam](https://github.com/Makdaam), 25/11/2019)
- Move GetPrivateAPIServerIPAddress into GenerateConfig ([#2090](https://github.com/openshift/openshift-azure/pull/2090), [@asalkeld](https://github.com/asalkeld), 21/11/2019)
