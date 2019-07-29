## Release Notes

This repo contains tooling to generate releases notes from merge commits for a given repository.

### Installation

```
$ go get github.com/openshift/openshift-azure/cmd/releasenotes
```

### Configuration
The release notes tool can be configured with the following parameters:

- `github-token` - GitHub access token
- `github-org` - Name of github organization
- `github-repo` - Name of github repository
- `start-sha` - The commit hash, branch or tag to start at
- `end-sha` - The commit hash, branch or tag to end at
- `release-version` - The release version for which the notes are being generated. This will be written at the top of the `output-file`
- `output-file` - The path to a file where the release notes will be written. If this is empty, the notes will be written to `Stdout`


### Release Notes sample scrape of the 'openshift/openshift-azure' Github repo

```
$ export GITHUB_TOKEN=<github_personal_access_token>
$ releasenotes -start-sha=v5.2.1 -end-sha=master -release-version=v6.0 -output-file=CHANGELOG.md

INFO[2019-07-23T11:17:24+02:00] parsing flags                                
INFO[2019-07-23T11:17:24+02:00] validating config                            
INFO[2019-07-23T11:17:24+02:00] creating release notes scraper               
INFO[2019-07-23T11:17:24+02:00] fetching release notes v5.2.1..master from github.com/openshift/openshift-azure 
INFO[2019-07-23T11:17:27+02:00] writing release notes to file CHANGELOG.md   
```

### Release Notes sample scrape of an arbitrary Github repo

```
$ export GITHUB_TOKEN=<github_personal_access_token>
$ releasenotes -github-repo release -start-sha=sttts-origin-owners -end-sha=master -release-version=v6.0 -output-file=CHANGELOG.md

INFO[2019-07-23T11:36:54+02:00] parsing flags                                
INFO[2019-07-23T11:36:54+02:00] validating config                            
INFO[2019-07-23T11:36:54+02:00] creating release notes scraper               
INFO[2019-07-23T11:36:54+02:00] fetching release notes sttts-origin-owners..master from github.com/openshift/release 
INFO[2019-07-23T11:37:23+02:00] writing release notes to file CHANGELOG.md  
```
