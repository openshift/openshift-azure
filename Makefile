COMMIT=$(shell git rev-parse --short HEAD)$(shell [[ $$(git status --porcelain) = "" ]] && echo -clean || echo -dirty)
CLUSTER_VERSION=$(shell awk '/^clusterVersion: /{ print $$2 }' <pluginconfig/pluginconfig-311.yaml)
# if we are on master branch we should always use dev tag
$(info CLUSTER_VERSION is ${CLUSTER_VERSION})
ifeq ($(CLUSTER_VERSION),v0.0)
  TAG := dev
else
  TAG := ${CLUSTER_VERSION}
endif
$(info TAG set to ${TAG})
LDFLAGS="-X main.gitCommit=$(COMMIT)"
E2E_IMAGE ?= quay.io/openshift-on-azure/e2e-tests:$(TAG)
E2E_BIN_IMAGE = $(E2E_IMAGE)
AZURE_CONTROLLERS_IMAGE ?= quay.io/openshift-on-azure/azure-controllers:$(TAG)
ETCDBACKUP_IMAGE ?= quay.io/openshift-on-azure/etcdbackup:$(TAG)
METRICSBRIDGE_IMAGE ?= quay.io/openshift-on-azure/metricsbridge:$(TAG)
SYNC_IMAGE ?= quay.io/openshift-on-azure/sync:$(TAG)
STARTUP_IMAGE ?= quay.io/openshift-on-azure/startup:$(TAG)

name2var = $(shell echo $1 | tr a-z A-Z | tr - _)
get_image_url = $(shell echo $($(1)_IMAGE))

ALL_BINARIES = azure-controllers e2e-bin etcdbackup sync metricsbridge startup
ALL_BUILDS = $(addsuffix .build, $(ALL_BINARIES))
ALL_IMAGES = $(addsuffix .image, $(ALL_BINARIES))
ALL_PUSHES = $(addsuffix .push, $(ALL_BINARIES))

IMAGEBUILDER = ${GOPATH}/bin/imagebuilder

# all is the default target to build everything
all: clean build $(ALL_BUILDS)

version:
	echo ${TAG}

build: generate
	go build ./...

clean:
	rm -f coverage.out e2e $(ALL_BINARIES)

test: unit e2e

generate:
	go generate ./...

create:
	timeout 1h ./hack/create.sh ${RESOURCEGROUP}

delete:
	./hack/delete.sh ${RESOURCEGROUP}
	rm -rf _data

e2e-bin.build: generate
	go test -ldflags ${LDFLAGS} -tags e2e -c -o ./e2e ./test/e2e

%.build: cmd/%/*.go generate
	go build -ldflags ${LDFLAGS} ./cmd/$(subst .build,,$@)

%.image: %.build $(IMAGEBUILDER)
	$(IMAGEBUILDER) -f Dockerfile.$(subst -bin,,$(subst .image,,$@)) -t $(call get_image_url,$(call name2var,$(subst .image,,$@))) .

%.push: %.image
	@echo docker push $(call get_image_url,$(call name2var,$(subst .push,,$@)))

all-image: $(ALL_IMAGES)

all-push: $(ALL_PUSHES)

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

cover: unit
	go tool cover -html=coverage.out

codecov: unit
	./hack/codecov-report.sh

upgrade:
	./hack/upgrade-e2e.sh release-test-${TAG}-${COMMIT} ${SOURCE}

e2e:
	FOCUS="\[AzureClusterReader\]|\[CustomerAdmin\]|\[EndUser\]\[Fake\]" TIMEOUT=60m ./hack/e2e.sh

e2e-prod:
	FOCUS="\[Default\]\[Real\]" TIMEOUT=70m ./hack/e2e.sh

e2e-etcdbackuprecovery:
	FOCUS="\[EtcdRecovery\]\[Fake\]" TIMEOUT=70m ./hack/e2e.sh

e2e-keyrotation:
	FOCUS="\[KeyRotation\]\[Fake\]" TIMEOUT=70m ./hack/e2e.sh

e2e-scaleupdown:
	FOCUS="\[ScaleUpDown\]\[Fake\]" TIMEOUT=30m ./hack/e2e.sh

e2e-forceupdate:
	FOCUS="\[ForceUpdate\]\[Fake\]" TIMEOUT=70m ./hack/e2e.sh

e2e-vnet:
	FOCUS="\[Vnet\]\[Real\]" TIMEOUT=70m ./hack/e2e.sh

$(IMAGEBUILDER):
	docker pull registry.access.redhat.com/rhel7:latest
	go get -u github.com/openshift/imagebuilder/cmd/imagebuilder

.PHONY: clean verify generate unit create delete upgrade e2e all-image all-push
