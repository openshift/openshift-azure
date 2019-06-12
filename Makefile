SHELL := /bin/bash
GITCOMMIT=$(shell git describe --tags HEAD)$(shell [[ $$(git status --porcelain) = "" ]] || echo -dirty)
LDFLAGS="-X main.gitCommit=$(GITCOMMIT)"
OPENSHIFT_INSTALL_DATA := ./vendor/github.com/openshift/installer/data/data

azure:
	go build -ldflags ${LDFLAGS} ./cmd/$@

.PHONY: vendor
vendor:
	dep check || dep ensure -update

.PHONY: verify
verify:
	./hack/verify/validate-codecov.sh
	go run ./hack/validate-imports/validate-imports.go cmd hack pkg

.PHONY: create
create: azure
	./hack/create.sh ${RESOURCEGROUP}

