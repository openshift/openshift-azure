SHELL := /bin/bash
GITCOMMIT=$(shell git describe --tags HEAD)$(shell [[ $$(git status --porcelain) = "" ]] || echo -dirty)
LDFLAGS="-X main.gitCommit=$(GITCOMMIT)"

.PHONY: verify

verify:
	./hack/verify/validate-codecov.sh
	go run ./hack/validate-imports/validate-imports.go cmd hack pkg test

