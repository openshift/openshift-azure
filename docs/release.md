# OSA release process

OSA is designed to allow customers to choose an update `stream` for their cluster. Currently the only stream is `3.11`, but this may change over time. A single release of the OSA project can in principle support co-existent clusters at multiple streams. In principle, clusters will one day be upgradable between streams. This is not in scope for this document.

An OSA release is a combination of multiple versioned components:

Per stream:

* VM image (operating system, openshift RPMs)
* OpenShift container images
* Microsoft container images (mdsd, mdm, etc.)
* Openshift-azure "azure" container image
* Openshift-azure repository, including plugin and manifest per stream

For each stream, the configuration manifest combines the version numbers of all of the above (except the openshift-azure repository) and is checked into the openshift-azure repository.
It is set as `clusterVersion` field.

OSA is an evolving service. During normal operations, our goal is to release the RP infrastructure on a three-weekly basis, although emergency changes can be expedited.
Globally, our goal is for all production RP release versions to correspond.
The OSA major release (`n`) is incremented with each three-weekly release.
The OSA minor release (`n.x`) starts at .0 and is incremented by exception, e.g. to publish an emergency release (CVE, critical bug fix) or resolve a QA or handover issue.

OSA releases are defined by git tags on the openshift-azure repository (`vn.x`).
The amount of changes per new release could vary. This range includes little to no changes to many or possibly all versioned items changing.
Our goal is to ensure that during normal operations, clusters are created at, and upgraded to, versions exactly corresponding with a released manifest according to the cluster’s stream.
Irrespective of the stream, our goal for the spread of production cluster versions to be as small as feasible (e.g. n.0, (n-1).0, (n-1).1, (n-2).0).  Exact spread goal TBD. A larger spread implies more testing.

Within a stream, our goal is to be able to directly upgrade any deployed production cluster to the latest version.

# Versioning

## Major releases
The plugin major release (“n”) is incremented with each sprint's release.

### Major release creation flow:

1. Create a branch `release-n` with a plugin code you wish to release. It is preferable, but not essential, that we branch master. We expect the commit that is being branched to be passing tests
2. Configuration step for CI so new release branch would be tested with our test suites
3. Follow minor release flow

## Minor releases

### Minor release creation flow (“n.x”):

1. Make a commit in `release-n`:
   in each stream’s manifest we set the tag of each openshift-azure container image to `:vn.x`
   in each stream’s manifest we set the plugin release version to `vn.x`

2. Git tag release (`vn.x`)
3. Build openshift-azure container image (`sync:vn.x`, etc)
4. If the minor release is a cherry-pick (not a straightforward branch of master), carry out update testing before release to MSFT

Branching model
```
-------master--*----------*---------------------->
               \          \
                \           \-release-v4----T(v4.0)-----T(v4.1)->
                 \-release-v3------T(v3.0)---->
```


# Doing a release with current integration:

0. Build and publish the VM image. For major releases, typically this should
   happen at the end of the second week of the sprint, and the VM image will
   not change during the release. See the [VM building
   SOP.](https://github.com/openshift/azure-sop/blob/master/SOP/eng/releng.asciidoc)

1. Create a release branch from master and push it.

```
git checkout <required commit>
git checkout -b release-vx
# we create release branch for major releases only
git push upstream release-vx
```

If it is minor release:
```
git checkout release-vx
git checkout -b release-vx.y-fix
git cherry-pick <git commit id>
```

Command `/cherrypick release-vx` prow command can be used too.

2. Configure testing for release branch in `openshift/release` repository. This
   will make sure that any PR to release branches is gated by tests.

Add new branch config file in `ci-operator/config/openshift/openshift-azure/openshift-openshift-azure-release-vx.yaml`

Add new test jobs for release branch:
`ci-operator/jobs/openshift/openshift-azure/openshift-openshift-azure-release-vx-presubmits.yaml`

Run prowgen to validate configuration:
```
docker pull registry.svc.ci.openshift.org/ci/ci-operator-prowgen:latest
docker run -it -v $(pwd)/ci-operator:/ci-operator:z registry.svc.ci.openshift.org/ci/ci-operator-prowgen:latest --from-dir /ci-operator/config/ --to-dir /ci-operator/jobs
```

Merge it to release repository. After this is done, all PR's on the release
branch should run all tests.

Sample PR: https://github.com/openshift/release/pull/4440

3. Container images for production are released to ACR. This can be done by
   people who have ACR root secrets. Publishing to ACR is a manual process for
   now.

   These images can be built by checking out `release-vX` branch. Build the
   container image on the release branch (`make azure-image`), tag it for ACR
   (`docker tag quay.io/openshift-on-azure/azure:vx.y
   osarpint.azurecr.io/openshift-on-azure/azure:vx.y`), and push the image to
   ACR (`docker push osarpint.azurecr.io/openshift-on-azure/azure:vx.y`).

4. Create release PR into `openshift/openshift-azure` repository release branch. This is main step, where we configure/edit release specific configuration.

```
git checkout upstream/release-vx
git checkout -b release-vx-pluginconfig
```

Update `pluginconfig/pluginconfig-311.yaml` file with image versions for this
particular release where necessary:

```
clusterVersion: vx.y
imageVersion: 311.69.20190214
images:
  ansibleServiceBroker: registry.access.redhat.com/openshift3/ose-ansible-service-broker:v3.11.69
  azureControllers: quay.io/openshift-on-azure/azure-controllers:vx.y
  etcdBackup: quay.io/openshift-on-azure/azure:vx.y
```

Typically you will only update the quay.io container image versions to pin them
to the latest release. The upstream OpenShift images, supporting software
versions, and VM image should be frozen mid-sprint to ensure sufficient time
for testing.

You will also need to generate release notes. The `GITHUB_TOKEN` used below
does not require any special permissions, so you can generate your own, or use
the team secret if you prefer.

```
export GITHUB_TOKEN=<github_token_from_team_secret>
go run cmd/releasenotes/releasenotes.go -start-sha=v(x.y-1) -end-sha=HEAD -release-version=v(x.y) -output-file=CHANGELOG.md
git add CHANGELOG.md
```

Note about release numbers to use as commitrange with releasenotes above:
* If you're cutting a new major release, (vx.y)-1 means v(x-1)."latest y on the x-1 branch"
* If you're cutting a new minor release, (vx.y)-1 means vx.(y-1)

Merge this PR into `release-vx` branch. If step 2 was completed right, this PR now should run all OSA test suites for it to be merged. Test will be using CI infrastructure built images.

Sample PR: https://github.com/openshift/openshift-azure/pull/1851

5. Tag release on top of the PR you just merged in step 3.

This step requires write access on git repo. You might need to ask somebody
else to execute these commands.

```
git checkout upstream/release-vx
# Better to sign the release, Tagger needs a GPG key setup in github
git tag -a -s -m "Version vx.y" vx.y
git push upstream tags/vx.y
```

6. Add upgrade tests for published release in `openshift/release`

```
git checkout -b osa.release.vx.testing
```

Add vx.y target to
`ci-operator/jobs/openshift/openshift-azure/openshift-openshift-azure-master-presubmits.yaml`
and
`ci-operator/jobs/openshift/openshift-azure/openshift-openshift-azure-release-vx-presubmits.yaml`:

```
- agent: kubernetes
  always_run: true
  branches:
  - master
  context: upgrade-vx.y
  decorate: true
  name: pull-ci-azure-master-upgrade-vx.y
  rerun_command: /test upgrade-vx.y
  spec:
    containers:
    - args:
      - hack/tests/e2e-upgrade.sh
      - vx.y
      image: registry.svc.ci.openshift.org/azure/ci-base:latest
      name: ""
      resources: {}
    serviceAccountName: ci-operator
  trigger: (?m)^/test( | .* )upgrade-vx.y,?($|\s.*)
```

Run prowgen to generate test jobs:
```
docker pull registry.svc.ci.openshift.org/ci/ci-operator-prowgen:latest
docker run -it -v $(pwd)/ci-operator:/ci-operator:z registry.svc.ci.openshift.org/ci/ci-operator-prowgen:latest --from-dir /ci-operator/config/ --to-dir /ci-operator/jobs
```

Merge the PR. After this test command `/test upgrade-vx.y` should work on PR's.

Sample PR: https://github.com/openshift/release/pull/7595

## Start development of the next major version

Prepare a new PR to master which does the following:

* updates pluginconfig-311.yaml as follows:
  - sets pluginVersion to v(x+1).0
  - copies versions/vx.n to versions/v(x+1).0
  - changes versions/vx.n images from `:vx.n` to `:latest` (latest versions can
    be located in the Red Hat container catalog, e.g.
    https://access.redhat.com/containers/?tab=tags#/registry.access.redhat.com/rhel7/etcd )
* copies `pkg/{arm,config,startup,sync}/vx` directories to
  `pkg/{arm,config,startup,sync}/v(x+1)`
* adds new directory mappings for v(x+1).0 in
  `pkg/{arm,config,startup,sync}/{arm,config,startup,sync}.go`
* sets the hashes for vx.n in `pkg/cluster/hash_test.go` as immutable

Sample PR: https://github.com/openshift/openshift-azure/pull/1852

Each step in the instructions above is included as a separate commit in the
sample PR.

# Cleaning old release

When deprecating a release we need to clean old release reference files from `release` repository:

Delete branch specific files:
```
ci-operator/config/openshift/openshift-azure/openshift-openshift-azure-release-vx.yaml
```

Delete test job configurations files:
```
ci-operator/jobs/openshift/openshift-azure/openshift-openshift-azure-release-vx-presubmits.yaml
ci-operator/jobs/openshift/openshift-azure/openshift-openshift-azure-release-vx-postsubmits.yaml
```

Delete reference in master branch config files:
```
# delete anything related to jobs with upgrade-vx.y release. In example job:
# pull-ci-openshift-openshift-azure-master-upgrade-vx.y
ci-operator/jobs/openshift/openshift-azure/openshift-openshift-azure-master-postsubmits.yaml
ci-operator/jobs/openshift/openshift-azure/openshift-openshift-azure-master-presubmits.yaml
```

Github tags/branches and old images inside the registry are left for future reference.

You will also need to clean up code on `openshift/openshift-azure` by following
the steps in the "Start development for the next major version" section in
reverse: that is, removing the old plugin version hashes, removing directory
references, removing old directories, and removing references to that version
in the pluginconfig.

Sample PR: https://github.com/openshift/openshift-azure/pull/2123
