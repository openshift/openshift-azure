## v15.0

- Bugfix: fix custom DNS issues ([#2213](https://github.com/openshift/openshift-azure/pull/2213) and [#2230](https://github.com/openshift/openshift-azure/pull/2230), [@kwoodson](https://github.com/kwoodson), 13/02/2020)
- Allow `osa-customer-admins` group members to run `who-can`, as in `oc adm policy who-can create user` ([#2197](https://github.com/openshift/openshift-azure/pull/2197), [@nilsanderselde](https://github.com/nilsanderselde), 06/02/2020)
- Intended as a soft launch: support larger cluster creation up to 100 nodes based on previous performance tests now that the timeout issues for cluster creation of this size have been mitigated ([#2204](https://github.com/openshift/openshift-azure/pull/2204), [@mjudeikis](https://github.com/mjudeikis), 30/01/2020)
- Fix RBAC for ILB ([#2201](https://github.com/openshift/openshift-azure/pull/2201), [@mjudeikis](https://github.com/mjudeikis), 30/01/2020)
- Update vmimage, upgrading to [OpenShift 3.11.161](https://docs.openshift.com/container-platform/3.11/release_notes/ocp_3_11_release_notes.html#ocp-3-11-161) ([#2185](https://github.com/openshift/openshift-azure/pull/2185), [@kwoodson](https://github.com/kwoodson), 29/01/2020)
- Change cloud-provider config rate limits ([#2199](https://github.com/openshift/openshift-azure/pull/2199), [@mjudeikis](https://github.com/mjudeikis), 28/01/2020)
- Hide config from update return errors ([#2191](https://github.com/openshift/openshift-azure/pull/2191), [@mjudeikis](https://github.com/mjudeikis), 22/01/2020)
- Add client id for use in unit tests which disables roleLister ([#2187](https://github.com/openshift/openshift-azure/pull/2187), [@jim-minter](https://github.com/jim-minter), 22/01/2020)
- Bugfix: Enable enduser to manage Kafka clusters ([#2129](https://github.com/openshift/openshift-azure/pull/2129), [@olga-mir](https://github.com/olga-mir), 21/01/2020)
- Bugfix: fix oc binary build regression ([#2170](https://github.com/openshift/openshift-azure/pull/2170), [@pnasrat](https://github.com/pnasrat), 21/01/2020)
- Rename Log Analytics to omsagent and upgrade omsagent to 12042019 ([#2133](https://github.com/openshift/openshift-azure/pull/2133), [@olga-mir](https://github.com/olga-mir), 08/01/2020)
