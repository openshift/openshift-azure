#!/bin/bash -e

if [[ $# -eq 0 && ! -e _data/containerservice.yaml ]]; then
    echo error: _data/containerservice.yaml must exist
    exit 1
fi

if [[ $# -eq 1 ]]; then
    export RESOURCEGROUP=$1
else
    export RESOURCEGROUP=$(awk '/^    resourceGroup:/ { print $2 }' <_data/containerservice.yaml)
fi

if [[ -z "${ADMIN_MANIFEST}${TEST_IN_PRODUCTION}" ]]; then
    echo ""
    echo "ADMIN_MANIFEST is not set, this will only do an empty update"
    echo ""
    echo "To do an upgrade set ADMIN_MANIFEST=test/manifests/fakerp/admin-update.yaml"
    echo "To scale up compute by one set ADMIN_MANIFEST=test/manifests/fakerp/scale.yaml"
    echo ""
fi

if [[ -n "$TEST_IN_PRODUCTION" ]]; then
    TEST_IN_PRODUCTION="-use-prod=true"
else
    [[ -e /var/run/secrets/kubernetes.io ]] || go generate ./...
    # building takes time, so if we use "go run" we will not get the PID of
    # the server as it won't have started yet.
    go build ./cmd/fakerp
    ./fakerp &
    trap 'return_id=$?; set +ex; kill $(lsof -t -i :8080); wait $(lsof -t -i :8080); exit $return_id' EXIT
fi
if [[ -n "$ADMIN_MANIFEST" ]]; then
    ADMIN_MANIFEST="-admin-manifest=$ADMIN_MANIFEST"
fi

go run cmd/createorupdate/createorupdate.go ${TEST_IN_PRODUCTION:-} ${ADMIN_MANIFEST:-}
