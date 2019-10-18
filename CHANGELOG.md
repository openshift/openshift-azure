## v10.1

- Add PrivateAPIServer field to MasterPoolProfile  ([#2004](https://github.com/openshift/openshift-azure/pull/2004), [@asalkeld](https://github.com/asalkeld), 17/10/2019)
- Migrate container images to ACR ([@ehashman](https://github.com/ehashman), 17/10/2019)

## v10.0

- Move ImagePullSecret to ARM and use combined secret ([#1949](https://github.com/openshift/openshift-azure/pull/1949), [@mjudeikis](https://github.com/mjudeikis), 11/10/2019)
- `ACS_RESOURCE_NAME` is corrected to be `AKS_RESOURCE_ID` (the value is the full resourceId) and added `AKS_REGION` ([#1994](https://github.com/openshift/openshift-azure/pull/1994), [@asalkeld](https://github.com/asalkeld), 11/10/2019)
- Add Private Link functionality for cluster monitoring  ([#1906](https://github.com/openshift/openshift-azure/pull/1906), [@mjudeikis](https://github.com/mjudeikis), 02/10/2019)
- Store the monitoring workspace key in the secrets not in the clusterConfig ([#1977](https://github.com/openshift/openshift-azure/pull/1977), [@asalkeld](https://github.com/asalkeld), 30/09/2019)
- Update to [OpenShift 3.11.146](https://docs.openshift.com/container-platform/3.11/release_notes/ocp_3_11_release_notes.html#ocp-3-11-146) ([#1979](https://github.com/openshift/openshift-azure/pull/1979), [@ehashman](https://github.com/ehashman), 27/09/2019)
