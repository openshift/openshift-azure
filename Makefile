COMMIT=$(shell git rev-parse --short HEAD)$(shell [[ $$(git status --porcelain --ignored) = "" ]] && echo -clean || echo -dirty)

# all is the default target to build everything
all: clean build azure-controllers etcdbackup sync

build: generate
	go build ./...

clean:
	rm -f coverage.out azure-controllers etcdbackup sync

test: unit e2e

generate:
	go generate ./...

TAG ?= $(shell git rev-parse --short HEAD)
E2E_IMAGE ?= quay.io/openshift-on-azure/e2e-tests:$(TAG)
AZURE_CONTROLLERS_IMAGE ?= quay.io/openshift-on-azure/azure-controllers:$(TAG)
ETCDBACKUP_IMAGE ?= quay.io/openshift-on-azure/etcdbackup:$(TAG)
SYNC_IMAGE ?= quay.io/openshift-on-azure/sync:$(TAG)

azure-controllers: generate
	go build -ldflags "-X main.gitCommit=$(COMMIT)" ./cmd/azure-controllers

azure-controllers-image: azure-controllers
	go get github.com/openshift/imagebuilder/cmd/imagebuilder
	imagebuilder -f Dockerfile.azure-controllers -t $(AZURE_CONTROLLERS_IMAGE) .

azure-controllers-push: azure-controllers-image
	docker push $(AZURE_CONTROLLERS_IMAGE)

etcdbackup: generate
	go build -ldflags "-X main.gitCommit=$(COMMIT)" ./cmd/etcdbackup

etcdbackup-image: etcdbackup
	go get github.com/openshift/imagebuilder/cmd/imagebuilder
	imagebuilder -f Dockerfile.etcdbackup -t $(ETCDBACKUP_IMAGE) .

etcdbackup-push: etcdbackup-image
	docker push $(ETCDBACKUP_IMAGE)

sync: generate
	go build -ldflags "-X main.gitCommit=$(COMMIT)" ./cmd/sync

sync-image: sync
	go get github.com/openshift/imagebuilder/cmd/imagebuilder
	imagebuilder -f Dockerfile.sync -t $(SYNC_IMAGE) .

sync-push: sync-image
	docker push $(SYNC_IMAGE)

verify:
	./hack/validate-generated.sh
	go vet ./...
	./hack/verify-code-format.sh
	./hack/validate-util.sh

unit: generate
	go test ./... -coverprofile=coverage.out
ifneq ($(ARTIFACT_DIR),)
	mkdir -p $(ARTIFACT_DIR)
	cp coverage.out $(ARTIFACT_DIR)
endif

cover: unit
	go tool cover -html=coverage.out

e2e: generate
	./hack/e2e.sh

e2e-prod:
	go test ./test/suites/realrp -tags e2e -test.v -ginkgo.v -ginkgo.randomizeAllSpecs -ginkgo.noColor -ginkgo.focus=Real -timeout 4h

.PHONY: clean sync-image sync-push verify unit e2e e2e-prod
