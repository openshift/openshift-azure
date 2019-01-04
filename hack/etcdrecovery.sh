#!/bin/bash

go generate ./...
go run cmd/recoveretcdcluster/recoveretcdcluster.go "$@"
