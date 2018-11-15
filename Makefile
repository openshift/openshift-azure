COMMIT=$(shell git rev-parse --short HEAD)$(shell [[ $$(git status --porcelain --ignored) = "" ]] && echo -clean || echo -dirty)

# all is the default target to build everything
all: clean build sync

build: generate
	go build ./...

clean:
	rm -f azure-reader.log coverage.out end-user.log e2e.test sync

test: unit e2e

generate:
	go generate ./...

TAG ?= $(shell git rev-parse --short HEAD)
SYNC_IMAGE ?= quay.io/openshift-on-azure/sync:$(TAG)
E2E_IMAGE ?= quay.io/openshift-on-azure/e2e-tests:$(TAG)

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
	go test ./test/e2erp -tags e2erp -test.v -ginkgo.v -ginkgo.randomizeAllSpecs -ginkgo.noColor -ginkgo.focus=Real -timeout 4h

.PHONY: clean sync-image sync-push verify unit e2e e2e-prod
