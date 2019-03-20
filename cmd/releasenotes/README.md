# Release Notes

This repo contains tooling to generates releases notes from merge commits given a repository.

## Installation

- install libgit2-devel system package (dnf/apt/pacman)

```
$ go get github.com/openshift/openshift-azure/cmd/releasenotes
```

## Release Notes Gathering Example for the 'openshift/openshift-azure' Github repo

```
$ export GITHUB_TOKEN=<github_personal_access_token>
$ releasenotes -repopath . -commitrange v2.5..master > release-notes.md
```

For a repo different than the default (openshift-azure) try:

```
$ releasenotes -reponame orgORuser/reponame -repopath . -commitrange v2.5..master > release-notes.md
```
