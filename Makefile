COMMIT=$(shell git rev-parse --short HEAD)$(shell [[ $$(git status --porcelain) = "" ]] && echo -clean || echo -dirty)
PLUGIN_VERSION=$(shell awk '/^pluginVersion: /{ print $$2 }' <pluginconfig/pluginconfig-311.yaml)
# if we are on master branch we should always use dev tag
$(info PLUGIN_VERSION is ${PLUGIN_VERSION})
ifeq ($(PLUGIN_VERSION),v0.0)
  TAG := dev
else
  TAG := ${PLUGIN_VERSION}
endif
$(info TAG set to ${TAG})
LDFLAGS="-X main.gitCommit=$(COMMIT)"
E2E_IMAGE ?= quay.io/openshift-on-azure/e2e-tests:$(TAG)
AZURE_CONTROLLERS_IMAGE ?= quay.io/openshift-on-azure/azure-controllers:$(TAG)
ETCDBACKUP_IMAGE ?= quay.io/openshift-on-azure/etcdbackup:$(TAG)
METRICSBRIDGE_IMAGE ?= quay.io/openshift-on-azure/metricsbridge:$(TAG)
SYNC_IMAGE ?= quay.io/openshift-on-azure/sync:$(TAG)
STARTUP_IMAGE ?= quay.io/openshift-on-azure/startup:$(TAG)

ALL_BINARIES = azure-controllers e2e-tests etcdbackup sync metricsbridge startup
ALL_IMAGES = $(addsuffix -image, $(ALL_BINARIES))
ALL_PUSHES = $(addsuffix -push, $(ALL_BINARIES))

IMAGEBUILDER = imagebuilder

.PHONY: $(ALL_PUSHES) $(ALL_IMAGES) all version clean build test unit generate $(IMAGEBUILDER)
# all is the default target to build everything
all: clean build $(ALL_BINARIES)

version:
	echo ${TAG}

build: generate
	go build ./...

clean:
	rm -f coverage.out $(ALL_BINARIES)

generate:
	go generate ./...

test: unit e2e

.PHONY: create delete upgrade
create:
	timeout 1h ./hack/create.sh ${RESOURCEGROUP}

delete:
	./hack/delete.sh ${RESOURCEGROUP}
	rm -rf _data

upgrade:
	./hack/upgrade-e2e.sh release-test-${TAG}-${COMMIT} ${SOURCE}

azure-controllers: generate
	go build -ldflags ${LDFLAGS} ./cmd/$@

azure-controllers-image: azure-controllers $(IMAGEBUILDER)
	$(IMAGEBUILDER) -f images/azure-controllers/Dockerfile -t $(AZURE_CONTROLLERS_IMAGE) .

azure-controllers-push: azure-controllers-image
	docker push $(AZURE_CONTROLLERS_IMAGE)

# for backwards compat
.PHONY: e2e-bin e2e-image e2e-push
e2e-bin: e2e-tests
e2e-image: e2e-tests-image
e2e-push: e2e-tests-push

e2e-tests: generate
	go test -ldflags ${LDFLAGS} -tags e2e -c -o ./e2e-tests ./test/e2e

e2e-tests-image: e2e-tests $(IMAGEBUILDER)
	imagebuilder -f images/e2e-tests/Dockerfile -t $(E2E_IMAGE) .

e2e-tests-push: e2e-tests-image
	docker push $(E2E_IMAGE)

etcdbackup: generate
	go build -ldflags ${LDFLAGS} ./cmd/$@

etcdbackup-image: etcdbackup $(IMAGEBUILDER)
	imagebuilder -f images/etcdbackup/Dockerfile -t $(ETCDBACKUP_IMAGE) .

etcdbackup-push: etcdbackup-image
	docker push $(ETCDBACKUP_IMAGE)

metricsbridge:
	go build -ldflags ${LDFLAGS} ./cmd/$@

metricsbridge-image: metricsbridge $(IMAGEBUILDER)
	$(IMAGEBUILDER) -f images/metricsbridge/Dockerfile -t $(METRICSBRIDGE_IMAGE) .

metricsbridge-push: metricsbridge-image
	docker push $(METRICSBRIDGE_IMAGE)

sync: generate
	go build -ldflags ${LDFLAGS} ./cmd/$@

sync-image: sync $(IMAGEBUILDER)
	$(IMAGEBUILDER) -f images/sync/Dockerfile -t $(SYNC_IMAGE) .

sync-push: sync-image
	docker push $(SYNC_IMAGE)

all-image: $(ALL_IMAGES)

all-push: $(ALL_PUSHES)

startup: generate
	go build -ldflags ${LDFLAGS} ./cmd/$@

startup-image: startup $(IMAGEBUILDER)
	$(IMAGEBUILDER) -f images/startup/Dockerfile -t $(STARTUP_IMAGE) .

startup-push: startup-image
	docker push $(STARTUP_IMAGE)

.PHONY: verify
verify:
	./hack/validate-generated.sh
	go vet ./...
	./hack/verify-code-format.sh
	./hack/validate-util.sh
	go run ./hack/validate-imports/validate-imports.go cmd hack pkg test
	go run ./hack/lint-addons/lint-addons.go -n

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

.PHONY: e2e e2e-prod e2e-etcdbackuprecovery e2e-keyrotation e2e-reimagevm e2e-scaleupdown e2e-forceupdate e2e-vnet
e2e:
	FOCUS="\[CustomerAdmin\]|\[EndUser\]\[Fake\]" TIMEOUT=60m ./hack/e2e.sh

e2e-prod:
	FOCUS="\[Default\]\[Real\]" TIMEOUT=70m ./hack/e2e.sh

e2e-etcdbackuprecovery:
	FOCUS="\[EtcdRecovery\]\[Fake\]" TIMEOUT=70m ./hack/e2e.sh

e2e-keyrotation:
	FOCUS="\[KeyRotation\]\[Fake\]" TIMEOUT=70m ./hack/e2e.sh

e2e-reimagevm:
	FOCUS="\[ReimageVM\]\[Fake\]" TIMEOUT=10m ./hack/e2e.sh

e2e-scaleupdown:
	FOCUS="\[ScaleUpDown\]\[Fake\]" TIMEOUT=30m ./hack/e2e.sh

e2e-forceupdate:
	FOCUS="\[ForceUpdate\]\[Fake\]" TIMEOUT=70m ./hack/e2e.sh

e2e-vnet:
	FOCUS="\[Vnet\]\[Real\]" TIMEOUT=70m ./hack/e2e.sh

$(IMAGEBUILDER):
	docker pull registry.access.redhat.com/rhel7:latest
	go get -u github.com/openshift/imagebuilder/cmd/imagebuilder
