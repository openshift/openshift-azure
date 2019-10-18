#!/bin/bash -e

make content
git status
go generate ./...

. hack/tests/ci-prepare.sh

git add --all pkg/sync/$(shell go run hack/dev-version/dev-version.go)
git add --all docs/
GIT_COMMITTER_NAME=openshift-azure-robot GIT_COMMITTER_EMAIL=aos-azure@redhat.com git commit --no-gpg-sign --author 'openshift-azure-robot <aos-azure@redhat.com>' -m "Content update" || exit 0

git push https://openshift-azure-robot:$GITHUB_TOKEN@github.com/openshift-azure-robot/openshift-azure.git HEAD:content-update -f

git reset HEAD^

curl -u openshift-azure-robot:$GITHUB_TOKEN -H "Content-Type:application/json" -d @- -so /dev/null https://api.github.com/repos/openshift/openshift-azure/pulls <<'EOF'
{
    "title": "Content update",
    "body": "```release-notes\r\nNONE\r\n```",
    "head": "openshift-azure-robot:content-update",
    "base": "master"
}
EOF
