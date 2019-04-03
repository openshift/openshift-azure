# OSA release process

OSA is designed to allow customers to choose an update `stream` for their cluster. Currently the only stream is `3.11`, but this may change over time. A single release of the OSA project can in principle support co-existent clusters at multiple streams. In principle, clusters will one day be upgradable between streams. This is not in scope for this document.

An OSA release is a combination of multiple versioned components:

Per stream:

* VM image (operating system, openshift RPMs)
* OpenShift container images
* Microsoft container images (mdsd, mdm, etc.)
* Openshift-azure container images (sync, etcdbackup, azure-controllers, etc.)
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

Command `/cherrypick release-vx` prow command can be used too.

Open a PR into release branch.

2. Configure testing for release branch in `openshift/release` repository. This will make sure that any PR to release branches is gated by tests.

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

Merge it to release repository. After this is done, all PR's should run all tests.


3. Create release PR into `openshift/openshift-azure` repository release branch. This is main step, where we configure/edit release specific configuration.

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

create `CHANGELOG.md` at the root of the project starting with the version
```
# vx.y
         <- additional blank line
```

scrape and append release-notes to `CHANGELOG.md`
```
make releasenotes
export GITHUB_TOKEN=<github_token_from_team_secret>
./releasenotes -repopath . -commitrange (vx.y)-1..HEAD >CHANGELOG.md
git add CHANGELOG.md
```
note about release numbers to use as commitrange with releasenotes above:
* If you're cutting a new major release, (vx.y)-1 means v(x-1)."latest y on the x-1 branch"
* If you're cutting a new minor release, (vx.y)-1 means vx.(y-1)

Make sure you update `clusterVersion` field all image version to point to specific version, instead of `latest` tag.

Merge this PR into `release-vx` branch. If step 2 was completed right, this PR now should run all OSA test suites for it to be merged. Test will be using CI infrastructure built images.

4. Tag release for the release you just merged in step 3.

This step requires elevated right on git repo. You might need to ask somebody else to execute these commands.

```
git checkout upstream/release-vx
# Better to sign the release, Tagger needs a GPG key setup in github
git tag -a -s -m "Version vx.y" vx.y
git push upstream tags/vx.y
```

5. Build release images from release tags

This step should be executed only when PR from step 3 is merged and 4 is pushed.

```
git checkout tags/vx.y
# validate script can see right tag
make version
<you should see tag version as an output. It will be used to tag and publish images>
make all-push
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

Change generated jobs to `always_run: false` and `optional: true`. This will make upgrade tests on master branch optional!

We are not going to use automatic generated jobs with `e2e-upgrade-vx.y` because of lack of targets in the jobs. This should change in the future. See https://github.com/openshift/ci-operator/issues/276 for more details.

Copy existing `upgrade` jobs and create new job with `upgrade-vx.y` syntax. This is consequence of the issue above. `e2e-upgrade-vx.y` jobs lacks necessary targets so they cant complete if ran independently.
Merge the PR. After this test command `/test upgrade-vx.y` should work on PR's.

7. Start development of v(x+1).0!

Prepare a new PR to master which does the following:

* updates pluginconfig-311.yaml as follows:
  - sets pluginVersion to v(x+1).0
  - copies versions/vx.n to versions/v(x+1).0
  - changes versions/vx.n images from :latest to :vx.n
* copies pkg/{arm,config,startup,sync}/vx directories to
  pkg/{arm,config,startup,sync}/v(x+1)
* adds new directory mappings for v(x+1).0 in
   pkg/{arm,config,startup,sync}/{arm,config,startup,sync}.New()
* marks the hashes for vx.n in pkg/cluster/hash_test.go as immutable
* copies those hashes to v(x+1).0

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
