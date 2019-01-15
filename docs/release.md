# OSA Release process

OSA Release process consist of multiple dimensions:
* Plugin code release
* Config release
* VM image release

Update might be one or any combination of all dimensions.
Plugin code release is a new branch and considered a major release. Any VM image or config change is a minor release.

# Versioning

## Major releases

Major release creation flow:
1. Create a branch with a plugin code you wish to release.
2. Commit to update `PluginConfig` referencing release image versions.
3. Git tag major release version (`v3.0`)
4. Build release images (sync, metricsbridge, etc)
5. Make sure VM image is released and is available to be used by plugin code.

## Minor releases

If there is a need to update already released code - minor release is done.
This might happen if we make a mistake, want to update configuration, or not code related fix.

Minor release creation flow:
1. Add commit to release branch with required changes.
2. Git tag minor version (`v3.1`)
3. Build release images (sync, metricsbridge, etc)

Branching model:
```
-------master--*----------*-----------------------
                \          \
                 \-release-v3------T(v3.0)----
                             \-release-v4----T(v4.0)-----T(v4.1)-
```
Note: Branch creation doesn't correspond to release, tags does.

# Plugin Config / Plugin Template 

When initiating plugin code and calling [GenerateConfig](https://github.com/openshift/openshift-azure/blob/master/pkg/api/plugin.go#L106). method we need to
pass in [Plugin Template/PluginConfig](https://github.com/openshift/openshift-azure/blob/master/pkg/api/plugin/api/config.go#L8) as a parameter

Plugin template defines main characteristics of the cluster. In example:
```yaml
imageOffer: osa
imagePublisher: redhat
imageSku: osa_311 
imageVersion: 311.43.20181121 # Node VM image version
images:
  alertManagerBase: registry.access.redhat.com/openshift3/prometheus-alertmanager # operators base images 
  ansibleServiceBroker: registry.access.redhat.com/openshift3/ose-ansible-service-broker:v3.11.43 # specific image version
```

Config file examples can be found in `pluginconfig/pluginconfig-{version}.yaml`. A config file is used to simplify production configuration change without new binary rollout. Struct contains secrets and it may not be wanted to store these in a yaml file.

# Testing architecture consideration

1. Hourly image sync job should be smart enough to detect tags and build "sync list" based on those.
2. CI infrastructure should be able to rebuild all container images retrospectively in case of CI cluster rebuild.
   Q: How we can build container images using git tags instead of branches using CI-Operator?
   Looks like this is not possible, and we might need to develope way to do this using OpenShift build

3. We should publish fakeRP images with same semantic versioning as part of release. These images will be used to 
   test all possible update/upgrade scenarios.
4. Test should stop using emptyDir for passing configuration around and download it from azure. This will enable 
   update tests
5. Prow configuration for testing the releases should be generated automatically using reference file with test
   matrix.

