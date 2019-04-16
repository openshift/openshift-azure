TAG=$(shell git describe --tags HEAD)
GITSTATUS=$(shell git status --porcelain)
GITCOMMIT=$(TAG)$(shell [ "$(GITSTATUS)" = "" ] && echo -clean || echo -dirty)
LDFLAGS="-X main.gitCommit=$(GITCOMMIT)"

AZURE_IMAGE ?= quay.io/openshift-on-azure/azure:$(TAG)

GOPATH ?= $(HOME)/go
IMAGEBUILDER = ${GOPATH}/bin/imagebuilder

.PHONY: azure-image azure-push all version clean test unit generate pullregistry secrets
# all is the default target to build everything
all: azure

version:
	@echo $(GITCOMMIT)

secrets:
	rm -rf secrets
	mkdir secrets
	oc extract -n azure secret/cluster-secrets-azure --to=secrets

clean:
	rm -f coverage.out azure releasenotes

generate:
	go generate ./...

test: unit e2e

.PHONY: create delete upgrade
create:
	./hack/create.sh ${RESOURCEGROUP}

delete:
	./hack/delete.sh ${RESOURCEGROUP}

upgrade:
	./hack/upgrade.sh ${RESOURCEGROUP}

artifacts:
	./hack/artifacts.sh

azure-image: azure $(IMAGEBUILDER) pullregistry
	$(IMAGEBUILDER) -f images/azure/Dockerfile -t $(AZURE_IMAGE) .

azure-push: azure-image
	docker push $(AZURE_IMAGE)

azure: generate
	go build -ldflags ${LDFLAGS} ./cmd/$@

sync-run: generate
	go run -ldflags ${LDFLAGS} ./cmd/azure sync --run-once --loglevel Debug

.PHONY: sync-run

monitoring: generate
	go build -ldflags ${LDFLAGS} ./cmd/$@

monitoring-run: monitoring
	./hack/monitoring.sh

monitoring-stop:
	./hack/monitoring.sh clean

releasenotes:
	go build -tags releasenotes ./cmd/$@

.PHONY: verify
verify:
	./hack/verify/validate-generated.sh
	go vet ./...
	./hack/verify/validate-code-format.sh
	./hack/verify/validate-util.sh
	./hack/verify/validate-codecov.sh
	go run ./hack/validate-imports/validate-imports.go cmd hack pkg test

unit: generate
	go test ./... -coverprofile=coverage.out -covermode=atomic
ifneq ($(ARTIFACT_DIR),)
	mkdir -p $(ARTIFACT_DIR)
	cp coverage.out $(ARTIFACT_DIR)
endif

.PHONY: cover codecov
cover: unit
	go tool cover -html=coverage.out

codecov: unit
	./hack/codecov-report.sh

.PHONY: e2e e2e-prod e2e-etcdbackuprecovery e2e-keyrotation e2e-reimagevm e2e-changeloglevel e2e-scaleupdown e2e-forceupdate e2e-vnet
e2e:
	FOCUS="\[CustomerAdmin\]|\[EndUser\]\[Fake\]" TIMEOUT=60m ./hack/e2e.sh

e2e-prod:
	FOCUS="\[Default\]\[Real\]" TIMEOUT=70m ./hack/e2e.sh

e2e-etcdbackuprecovery:
	FOCUS="\[EtcdRecovery\]\[Fake\]" TIMEOUT=180m ./hack/e2e.sh

e2e-keyrotation:
	FOCUS="\[KeyRotation\]\[Fake\]" TIMEOUT=180m ./hack/e2e.sh

e2e-reimagevm:
	FOCUS="\[ReimageVM\]\[Fake\]" TIMEOUT=40m ./hack/e2e.sh

e2e-changeloglevel:
	FOCUS="\[ChangeLogLevel\]\[Fake\]" TIMEOUT=180m ./hack/e2e.sh

e2e-scaleupdown:
	FOCUS="\[ScaleUpDown\]\[Fake\]" TIMEOUT=50m ./hack/e2e.sh

e2e-forceupdate:
	FOCUS="\[ForceUpdate\]\[Fake\]" TIMEOUT=180m ./hack/e2e.sh

e2e-vnet:
	FOCUS="\[Vnet\]\[Real\]" TIMEOUT=70m ./hack/e2e.sh

$(IMAGEBUILDER):
	go get github.com/openshift/imagebuilder/cmd/imagebuilder

pullregistry: $(IMAGEBUILDER)
	docker pull registry.access.redhat.com/rhel7:latest

vmimage:
	./hack/vmimage.sh

.PHONY: vmimage
