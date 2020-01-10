SHELL := /bin/bash
GITCOMMIT=$(shell git describe --tags HEAD)$(shell [[ $$(git status --porcelain) = "" ]] || echo -dirty)
LDFLAGS="-X main.gitCommit=$(GITCOMMIT)"

AZURE_IMAGE ?= quay.io/openshift-on-azure/azure:$(GITCOMMIT)
LATEST_PLUGIN_VERSION=$(shell go run hack/dev-version/dev-version.go)

.PHONY: all artifacts azure-image azure-push clean create delete e2e generate monitoring monitoring-run monitoring-stop secrets sync-run test testinsights unit upgrade verify vmimage

all: azure

secrets:
	@rm -rf secrets
	@mkdir secrets
	@oc extract -n azure secret/cluster-secrets-azure --to=secrets >/dev/null

clean:
	rm -f coverage.out azure releasenotes testinsights fakerp

generate:
	@[[ -e /var/run/secrets/kubernetes.io ]] || go generate ./...

test: unit e2e

create:
	./hack/create.sh ${RESOURCEGROUP}

delete:
	./hack/delete.sh ${RESOURCEGROUP}

upgrade:
	./hack/upgrade.sh ${RESOURCEGROUP}

artifacts:
	./hack/artifacts.sh

azure-image: azure
	./hack/image-build.sh images/azure/Dockerfile $(AZURE_IMAGE)

azure-push: azure-image
	docker push $(AZURE_IMAGE)

azure: generate
	go build -ldflags ${LDFLAGS} ./cmd/$@

sync-run: generate
	go run -ldflags ${LDFLAGS} ./cmd/azure sync --run-once --loglevel Debug

monitoring:
	go build -ldflags ${LDFLAGS} ./cmd/$@

monitoring-run: monitoring
	./hack/monitoring.sh

monitoring-stop:
	./hack/monitoring.sh clean

releasenotes:
	go build -tags releasenotes ./cmd/$@

content:
	go test -timeout=300s -tags=content -run=TestContent ./pkg/sync/$(LATEST_PLUGIN_VERSION)
	go run ./hack/generate-test-commands >docs/commands.md
	go generate ./pkg/sync/$(LATEST_PLUGIN_VERSION)

verify:
	go test -c -tags=content -run=TestContent ./pkg/sync/$(LATEST_PLUGIN_VERSION) && rm $(LATEST_PLUGIN_VERSION).test
	./hack/verify/validate-generated.sh
	go vet ./...
	./hack/verify/validate-code-format.sh
	./hack/verify/validate-util.sh
	./hack/verify/validate-codecov.sh
	go run ./hack/validate-imports/validate-imports.go cmd hack pkg test
	go run ./hack/verify/validate_pluginconfig.go
	./hack/verify/validate-sec.sh

testinsights:
	go build -ldflags ${LDFLAGS} ./cmd/$@

unit: generate testinsights
	go version
	go test ./... -coverprofile=coverage.out -covermode=atomic -json | ./testinsights

e2e:
	FOCUS="\[CustomerAdmin\]|\[EndUser\]" TIMEOUT=60m ./hack/e2e.sh

vmimage:
	./hack/vmimage.sh

vmimage-validate:
	./hack/vmimage-validate.sh
