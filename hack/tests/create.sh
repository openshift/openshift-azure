#!/bin/bash -e

cleanup() {
    make delete
}

trap cleanup EXIT

. hack/tests/ci-prepare.sh

make create
