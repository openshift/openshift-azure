SHELL := /bin/bash
GITCOMMIT=$(shell git describe --always --tags HEAD)$(shell [[ $$(git status --porcelain) = "" ]] || echo -dirty)
LDFLAGS="-X main.gitCommit=$(GITCOMMIT)"
OPENSHIFT_INSTALL_DATA := ./vendor/github.com/openshift/installer/data/data


.PHONY: all
all: clean azure

.PHONY: clean
clean:
	rm -f azure

azure:
	go build -ldflags ${LDFLAGS} ./cmd/$@

unit:
	go test ./... -coverprofile=coverage.out -covermode=atomic

.PHONY: vendor
vendor:
	dep check || dep ensure -update

.PHONY: verify
verify:
	./hack/verify/validate-codecov.sh
	./hack/verify/validate-code-format.sh
	go run ./hack/validate-imports/validate-imports.go cmd hack pkg test

.PHONY: create
create:
	./hack/create.sh ${RESOURCEGROUP}

.PHONY: delete
delete:
	./hack/delete.sh ${RESOURCEGROUP}
