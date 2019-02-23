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
AZURE_CONTROLLERS_IMAGE ?= quay.io/openshift-on-azure/azure-controllers:$(TAG)
ETCDBACKUP_IMAGE ?= quay.io/openshift-on-azure/etcdbackup:$(TAG)
METRICSBRIDGE_IMAGE ?= quay.io/openshift-on-azure/metricsbridge:$(TAG)
SYNC_IMAGE ?= quay.io/openshift-on-azure/sync:$(TAG)

# all is the default target to build everything
all: clean build azure-controllers etcdbackup sync metricsbridge e2e-bin

version:
	echo ${TAG}

build: generate
	go build ./...

clean:
	rm -f coverage.out azure-controllers etcdbackup sync metricsbridge e2e

test: unit e2e

generate:
	go generate ./...

create:
	timeout 1h ./hack/create.sh ${RESOURCEGROUP}

delete:
	./hack/delete.sh ${RESOURCEGROUP}
	rm -rf _data

azure-controllers: generate
	go build -ldflags ${LDFLAGS} ./cmd/azure-controllers

azure-controllers-image: azure-controllers imagebuilder
	imagebuilder -f Dockerfile.azure-controllers -t $(AZURE_CONTROLLERS_IMAGE) .

azure-controllers-push: azure-controllers-image
	docker push $(AZURE_CONTROLLERS_IMAGE)

e2e-bin: generate
	go test -ldflags ${LDFLAGS} -tags e2e -c -o ./e2e ./test/e2e

e2e-image: e2e-bin imagebuilder
	imagebuilder -f Dockerfile.e2e -t $(E2E_IMAGE) .

e2e-push: e2e-image
	docker push $(E2E_IMAGE)

recoveretcdcluster: generate
	go build -ldflags ${LDFLAGS} ./cmd/recoveretcdcluster

etcdbackup: generate
	go build -ldflags ${LDFLAGS} ./cmd/etcdbackup

etcdbackup-image: etcdbackup imagebuilder
	imagebuilder -f Dockerfile.etcdbackup -t $(ETCDBACKUP_IMAGE) .

etcdbackup-push: etcdbackup-image
	docker push $(ETCDBACKUP_IMAGE)

metricsbridge:
	go build -ldflags ${LDFLAGS} ./cmd/metricsbridge

metricsbridge-image: metricsbridge imagebuilder
	imagebuilder -f Dockerfile.metricsbridge -t $(METRICSBRIDGE_IMAGE) .

metricsbridge-push: metricsbridge-image
	docker push $(METRICSBRIDGE_IMAGE)

sync: generate
	go build -ldflags ${LDFLAGS} ./cmd/sync

sync-image: sync imagebuilder
	imagebuilder -f Dockerfile.sync -t $(SYNC_IMAGE) .

sync-push: sync-image
	docker push $(SYNC_IMAGE)

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

imagebuilder:
	docker pull registry.access.redhat.com/rhel7:latest
	go get -u github.com/openshift/imagebuilder/cmd/imagebuilder

.PHONY: clean metricsbridge metricsbridge-image metricsbridge-push sync-image sync-push verify unit e2e imagebuilder
