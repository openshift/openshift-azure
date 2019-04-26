#!/bin/bash -ex

if [[ $# -ne 1 ]]; then
    echo "usage: $0 source_version"
    exit 1
fi

cleanup() {
    set +e

    if [[ -n "$ARTIFACT_DIR" ]]; then
        exec &>"$ARTIFACT_DIR/cleanup"
    fi

    stop_monitoring
    make artifacts

    if [[ -n "$T" ]]; then
        rm -rf "$T"
    fi

    if [[ -n "$NO_DELETE" ]]; then
        return
    fi
    make delete
    az group delete -g "$RESOURCEGROUP" --yes --no-wait
}
trap cleanup EXIT

. hack/tests/ci-operator-prepare.sh

T="$(mktemp -d)"
start_monitoring $T/src/github.com/openshift/openshift-azure/_data/containerservice.yaml

git clone -b "$1" https://github.com/openshift/openshift-azure.git $T/src/github.com/openshift/openshift-azure
ln -sf "$PWD/secrets" "$T/src/github.com/openshift/openshift-azure"
(
    set +x
    export AZURE_MASTER_CLIENT_ID=$AZURE_LEGACY_MASTER_CLIENT_ID
    export AZURE_MASTER_CLIENT_SECRET=$AZURE_LEGACY_MASTER_CLIENT_SECRET
    export AZURE_WORKER_CLIENT_ID=$AZURE_LEGACY_WORKER_CLIENT_ID
    export AZURE_WORKER_CLIENT_SECRET=$AZURE_LEGACY_WORKER_CLIENT_SECRET
    set -x
    cd "$T/src/github.com/openshift/openshift-azure"
    # TODO: remove after v3.2 goes away
    if [[ "$1" == "v3.2" ]]; then
        GOPATH="$T" go get github.com/golang/mock/mockgen
    fi
    GOPATH="$T" make create
)

cp -a "$T/src/github.com/openshift/openshift-azure/_data" .

set_build_images

# try upgrading just a single image to latest - TODO: need to improve this hack
WEBCONSOLE=$(python -c 'import yaml; c=yaml.safe_load(open("pluginconfig/pluginconfig-311.yaml")); print c["versions"][c["pluginVersion"]]["images"]["webConsole"]')
cat >/tmp/admin-update-console.yaml <<EOF
config:
  images:
    webConsole: $WEBCONSOLE
EOF
ADMIN_MANIFEST=/tmp/admin-update-console.yaml make upgrade
RUNNING_WEBCONSOLE=$(KUBECONFIG=_data/_out/admin.kubeconfig oc get deployment -n openshift-web-console webconsole -o jsonpath='{.spec.template.spec.containers[0].image}')
if [[ "$WEBCONSOLE" != "$RUNNING_WEBCONSOLE" ]]; then
    echo "expected webconsole image $WEBCONSOLE, got $RUNNING_WEBCONSOLE"
    exit 1
fi

# now upgrade the whole lot
ADMIN_MANIFEST=test/manifests/fakerp/admin-update.yaml make upgrade e2e
