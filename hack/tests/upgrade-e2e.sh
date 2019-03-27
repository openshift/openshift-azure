#!/bin/bash -ex

# TODO: move this into release
exec timeout 3h hack/tests/e2e-upgrade.sh "$1"
