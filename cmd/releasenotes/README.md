# Release Notes

This repo contains tooling to generates releases notes from merge commits given a repository.

## Installation

```
$ go get github.com/y-cote/releasenotes/cmd/releasenotes
```

## Release Notes Gathering Example for the 'openshift/openshift-azure' Github repo

```
$ releasenotes -repopath . -commitrange v2.5..master > release-notes.md
```

For a repo different than the default (openshift-azure) try:

```
$ releasenotes -reponame orgORuser/reponame -repopath . -commitrange v2.5..master > release-notes.md
```
