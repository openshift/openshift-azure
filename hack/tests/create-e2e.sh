#!/bin/bash -ex

# TODO: move this into release
exec timeout 2h hack/tests/e2e-create.sh
