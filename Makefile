TAG=$(shell git describe --tags HEAD)
GITCOMMIT=$(TAG)$(shell [[ $$(git status --porcelain) = "" ]] && echo -clean || echo -dirty)
LDFLAGS="-X main.gitCommit=$(GITCOMMIT)"

AZURE_IMAGE ?= quay.io/openshift-on-azure/arho:$(TAG)

ALL_BINARIES = azure-controllers e2e-tests etcdbackup sync metricsbridge startup tlsproxy canary 
ALL_IMAGES = azure-image e2e-tests-image
ALL_PUSHES = azure-push e2e-tests-push

GOPATH ?= $(HOME)/go
IMAGEBUILDER = ${GOPATH}/bin/imagebuilder

.PHONY: $(ALL_PUSHES) $(ALL_IMAGES) all version clean test unit generate pullregistry secrets
# all is the default target to build everything
all: $(ALL_BINARIES)

version:
	@echo $(GITCOMMIT)

secrets:
	rm -rf secrets
	mkdir secrets
	oc extract -n azure secret/cluster-secrets-azure --to=secrets

clean:
	rm -f coverage.out $(ALL_BINARIES) releasenotes

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

# for backwards compat
.PHONY: e2e-bin e2e-image e2e-push
e2e-bin: e2e-tests
e2e-image: e2e-tests-image
e2e-push: e2e-tests-push

e2e-tests: generate
	go test -ldflags ${LDFLAGS} -tags e2e -c -o ./e2e-tests ./test/e2e

e2e-tests-image: e2e-tests $(IMAGEBUILDER) pullregistry
	$(IMAGEBUILDER) -f images/e2e-tests/Dockerfile -t $(E2E_IMAGE) .

e2e-tests-push: e2e-tests-image
	docker push $(E2E_IMAGE)

azure-image: azure-controllers etcdbackup tlsproxy metricsbridge startup sync
	$(IMAGEBUILDER) -f images/azure/Dockerfile -t $(AZURE_IMAGE) .

azure-push: azure-image
	docker push $(AZURE_IMAGE)

azure-controllers: generate
	go build -ldflags ${LDFLAGS} ./cmd/$@

etcdbackup: generate
	go build -ldflags ${LDFLAGS} ./cmd/$@

tlsproxy: generate
	go build -ldflags ${LDFLAGS} ./cmd/$@

metricsbridge:
	go build -ldflags ${LDFLAGS} ./cmd/$@

startup: generate
	go build -ldflags ${LDFLAGS} ./cmd/$@

canary: generate
	go build -ldflags ${LDFLAGS} ./cmd/$@

sync: generate
	go build -ldflags ${LDFLAGS} ./cmd/$@

sync-run: generate
	go run -ldflags ${LDFLAGS} ./cmd/sync -run-once -loglevel Debug

.PHONY: sync-run

monitoring: generate
	go build -ldflags ${LDFLAGS} ./cmd/$@

monitoring-run: monitoring
	./hack/monitoring.sh

monitoring-stop:
	./hack/monitoring.sh clean

all-image: $(ALL_IMAGES)

all-push: $(ALL_PUSHES)

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
