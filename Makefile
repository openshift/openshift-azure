SHELL := /bin/bash
GITCOMMIT=$(shell git describe --tags HEAD)$(shell [[ $$(git status --porcelain) = "" ]] || echo -dirty)
LDFLAGS="-X main.gitCommit=$(GITCOMMIT)"

.PHONY: vendor
vendor:
	dep check || dep ensure -update

.PHONY: verify
verify:
	./hack/verify/validate-codecov.sh
	go run ./hack/validate-imports/validate-imports.go cmd hack pkg

.PHONY: create
create:
	./hack/create.sh ${RESOURCEGROUP}
