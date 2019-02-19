# OSA release process

OSA is designed to allow customers to choose an update `stream` for their cluster. Currently the only stream is `3.11`, but this may change over time. A single release of the OSA project can in principle support co-existent clusters at multiple streams. In principle, clusters will one day be upgradable between streams. This is not in scope for this document.

An OSA release is a combination of multiple versioned items:

Per stream:

* VM image (operating system, openshift RPMs)
* OpenShift container images
* Microsoft container images (mdsd, mdm, etc.)
* Openshift-azure container images (sync, etcdbackup, azure-controllers, etc.)
* Openshift-azure repository, including plugin and manifest per stream

For each stream, the configuration manifest combines the version numbers of all of the above (except the openshift-azure repository) and is checked into the openshift-azure repository. 
It is set as `clusterVersion` field.

OSA is an evolving service. During normal operations, we aim to release the RP infrastructure on a weekly basis, although emergency changes can be expedited.
Globally, we aim for all production RP release versions to correspond.
The OSA major release (`n`) is incremented with each weekly release.
The OSA minor release (`n.x`) starts at .0 and is incremented by exception, e.g. to publish an emergency release (CVE, critical bug fix) or resolve a handover issue.

OSA releases are defined by git tags on the openshift-azure repository (`vn.x`).
The change to any production cluster implied by a new release varies. It may be nothing; alternatively it is also feasible that many or all versioned items could change in a release.
We aim to ensure that during normal operations, clusters are created at, and upgraded to, versions exactly corresponding with a released manifest according to the cluster’s stream.
Irrespective of the stream, we aim for the spread of production cluster versions to be as small as feasible (e.g. n.0, (n-1).0, (n-1).1, (n-2).0).  Exact spread goal TBD.  A larger spread implies more testing.

Within a stream, we aim to be able to directly upgrade any deployed production cluster to the latest version.
Versioning

## Major releases
The plugin major release (“n”) is incremented with each weekly release.

### Major release creation flow:

1. Create a branch `release-n` with a plugin code you wish to release. It is preferable, but not essential, that we branch master. We expect the commit that is being branched to be passing tests
2. Configuration step for CI so new release branch would be tested with our test suites

Follow minor release flow
The above should be automated as much as possible, e.g. pushing a git tag triggers image builds

## Minor releases

### Minor release creation flow (“n.x”):

1. Make a commit in `release-n`:
   in each stream’s manifest we set the tag of each openshift-azure container image to `:vn.x`
   in each stream’s manifest we set the plugin release version to `vn.x`

2. Git tag release (`vn.x`)
3. Build openshift-azure container image (`sync:vn.x`, etc)
4. If the minor release is a cherry-pick (not a straightforward branch of master), carry out update testing before release to MSFT
5. (TBD) CI test on the result? User documentation release procedure? Release procedure to MSFT?

Branching model
```
-------master--*----------*---------------------->
               \          \
                \           \-release-v4----T(v4.0)-----T(v4.1)->
                 \-release-v3------T(v3.0)---->
```


# Doing a release with current integration:

1. Create a branch from master and push it. This will create branch for us to do release in.

```
git checkout <required commit> 
git checkout -b release-vx
# we create release branch for first weekly only
git push upstream release-vx
```

If it is minor release:
```
git checkout release-vx
git checkout -b release-vx.y-fix
git cherry-pick <git commit id>
```

Open a PR into release branch.

2. Configure testing for release branch in `openshift/release` repository

```
git checkout master
git fetch upstream 
git rebase upstream/master 
git checkout -b osa.vx.y.release
```

Add new branch config file in `ci-operator/config/openshift/openshift-azure/openshift-openshift-azure-release-vx.yaml`

Add new test jobs for release branch:
`ci-operator/jobs/openshift/openshift-azure/openshift-openshift-azure-release-vx-presubmits.yaml`

Run prowgen to validate configuration:
```
docker pull registry.svc.ci.openshift.org/ci/ci-operator-prowgen:latest
docker run -it -v $(pwd)/ci-operator:/ci-operator:z registry.svc.ci.openshift.org/ci/ci-operator-prowgen:latest --from-dir /ci-operator/config/ --to-dir /ci-operator/jobs
```

Merge it to release repository. This will make sure that any PR to release branches is gates by tests.

3. Create release PR into `openshift/openshift-azure` repository release branch:

```
git checkout upstream/release-vx
git checkout -b release-vx-pluginconfig
```

update `pluginconfig/pluginconfig-311.yaml` file with image version for this particular release
```
clusterVersion: vx.y
imageVersion: 311.69.20190214
images:
  ansibleServiceBroker: registry.access.redhat.com/openshift3/ose-ansible-service-broker:v3.11.69
  azureControllers: quay.io/openshift-on-azure/azure-controllers:vx.y
  etcdBackup: quay.io/openshift-on-azure/etcdbackup:vx.y
```

Make sure you update `clusterVersion` field all image version to point to specific version, instead of `latest` tag.

Merge this PR into `release-vx` branch. Test should be using test infrastructure built images.

4. Tag release for the release you just merged in step 3.

This step requires elevated right on git repo. You might need to ask somebody else to execute these commands.

```
git checkout upstream/release-vx
git tag -a vx.y -m 'reason' # where reason is release summary in one sentence
git push upstream tags/vx.y
```

1. Build release images from release tags

This step should be executed only when PR from step 3 is merged and 4 is pushed.

```
git checkout tags/vx.y
# validate script can see right tag
make version
<you should see tag version as an output. It will be used to tag and publish images>
make azure-controllers-push metricsbridge-push sync-push etcdbackup-push
```

6. Update upgrade tests for published release in `openshift/release`

This step can be done together with step 2, but for first release you are doing we recommend to do it as separate PR.

```
git checkout -b osa.release.vx.testing
```

Add vx.y target to `ci-operator/config/openshift/openshift-azure/openshift-openshift-azure-master.yaml`

```
- artifact_dir: /tmp/artifacts
 as: e2e-upgrade-vx.y
 commands: ARTIFACT_DIR=/tmp/artifacts SOURCE=vx.y make upgrade
 secret:
   name: azure
   mount_path: /usr/secrets
 container:
   from: src
```

Run prowgen to generate test jobs:
```
docker pull registry.svc.ci.openshift.org/ci/ci-operator-prowgen:latest 
docker run -it -v $(pwd)/ci-operator:/ci-operator:z registry.svc.ci.openshift.org/ci/ci-operator-prowgen:latest --from-dir /ci-operator/config/ --to-dir /ci-operator/jobs
```

Change generated jobs to `always_run: false` and `optional: true`
We not gonna use automatic generated jobs with `e2e-upgrade-vx.y` because of lack of targets.

Copy existing `upgrade` jobs and create new job with `upgrade-vx.y` syntax.
Merge the PR. After this test command `/test upgrade-vx.y` should work on PR's.
