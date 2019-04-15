# CI job babysitting

First, ensure that you familiarize yourself with [OSA CI](https://github.com/openshift/release/blob/master/projects/azure/README.md).

The expectation is that each of the following jobs should be green, and should have run in the last 24h.

## Branch jobs

If these fail, it is indicative of a flaking or completely broken test.

* [https://openshift-gce-devel.appspot.com/builds/origin-ci-test/logs/branch-ci-openshift-azure-misc-master-images/]
* [https://openshift-gce-devel.appspot.com/builds/origin-ci-test/logs/branch-ci-openshift-openshift-azure-master-e2e-azure/]
* [https://openshift-gce-devel.appspot.com/builds/origin-ci-test/logs/branch-ci-openshift-openshift-azure-master-images/]
* [https://openshift-gce-devel.appspot.com/builds/origin-ci-test/logs/branch-ci-openshift-openshift-azure-master-unit/]
* [https://openshift-gce-devel.appspot.com/builds/origin-ci-test/logs/branch-ci-openshift-openshift-azure-master-verify/]

## VM image builds

* [https://openshift-gce-devel.appspot.com/builds/origin-ci-test/logs/periodic-ci-azure-base-image-rhel]
* [https://openshift-gce-devel.appspot.com/builds/origin-ci-test/logs/periodic-ci-azure-node-image-rhel-311]

## Periodic jobs

* [https://openshift-gce-devel.appspot.com/builds/origin-ci-test/logs/periodic-ci-azure-etcdbackuprecovery-e2e/]
* [https://openshift-gce-devel.appspot.com/builds/origin-ci-test/logs/periodic-ci-azure-key-rotation-e2e/]
* [https://openshift-gce-devel.appspot.com/builds/origin-ci-test/logs/periodic-ci-azure-prod-e2e/]
* [https://openshift-gce-devel.appspot.com/builds/origin-ci-test/logs/periodic-ci-azure-scaleupdown-e2e/]
* [https://openshift-gce-devel.appspot.com/builds/origin-ci-test/logs/periodic-ci-azure-vnet-e2e/]
* [https://openshift-gce-devel.appspot.com/builds/origin-ci-test/logs/periodic-ci-openshift-openshift-azure-master-conformance/]
* [https://openshift-gce-devel.appspot.com/builds/origin-ci-test/logs/periodic-ci-openshift-openshift-azure-master-bushslicer/]

Also triage the [cron jobs in the azure namespace](https://github.com/openshift/release/blob/master/projects/azure/README.md#other-cron-jobs) in the [CI cluster](https://api.ci.openshift.org/console/). It is expected that all except for `token-refresh` should be regularly succeeding.

Close [test flake issues in openshift-azure](https://github.com/openshift/openshift-azure/issues?utf8=%E2%9C%93&q=is%3Aissue+is%3Aopen+label%3Akind%2Ftest-flake) where these no longer appear to be occurring.  Open issues for new test flakes. The expectation is that you do a first pass on root cause, fix the issue if straightforward, otherwise assign the issue to someone who can fix it. Please refer to the [testing flakes](https://github.com/openshift/openshift-azure/blob/master/docs/testing-flakes.md) document for more information about how to proceed when confronted with testing flakes.
