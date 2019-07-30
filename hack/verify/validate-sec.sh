#!/bin/bash -e

go get -u github.com/securego/gosec/cmd/gosec
gosec -severity medium -confidence medium -exclude G304 -quiet  ./...
