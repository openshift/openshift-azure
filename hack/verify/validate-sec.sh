#!/bin/bash -ex

PATH="$(go env GOPATH)/bin":$PATH

go get -u github.com/securego/gosec/cmd/gosec
gosec -severity medium -confidence medium -exclude G304,G110 -quiet  ./...
