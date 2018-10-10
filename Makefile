COMMIT=$(shell git rev-parse --short HEAD)$(shell [[ $$(git status --porcelain --ignored) = "" ]] && echo -clean || echo -dirty)

# all is the default target to build everything
all: clean build sync e2e-bin

build:
	go build ./...

clean:
	rm -f sync
	rm -f logbridge
	rm -f e2e.test

test: unit e2e

generate:
	go generate ./...

logbridge: generate
	go build -ldflags "-X main.gitCommit=$(COMMIT)" ./cmd/logbridge

logbridge-image: logbridge
	go get github.com/openshift/imagebuilder/cmd/imagebuilder
	imagebuilder -f Dockerfile.logbridge -t quay.io/openshift-on-azure/logbridge:latest .

logbridge-push: logbridge-image
	docker push quay.io/openshift-on-azure/logbridge:latest

sync: generate
	go build -ldflags "-X main.gitCommit=$(COMMIT)" ./cmd/sync

TAG ?= $(shell git rev-parse --short HEAD)
SYNC_IMAGE ?= quay.io/openshift-on-azure/sync:$(TAG)
E2E_IMAGE ?= quay.io/openshift-on-azure/e2e-tests:$(TAG)

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
	go test ./...

cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out

e2e: generate
	./hack/e2e.sh

e2e-bin: generate
	go test -tags e2e -ldflags "-X github.com/openshift/openshift-azure/test/e2e.gitCommit=$(shell git rev-parse --short HEAD)" -i -c -o e2e.test ./test/e2e

e2e-image: e2e-bin
	go get github.com/openshift/imagebuilder/cmd/imagebuilder
	imagebuilder -f Dockerfile.e2e -t $(E2E_IMAGE) .

e2e-push: e2e-image
	docker push $(E2E_IMAGE)

.PHONY: clean sync-image sync-push verify unit e2e e2e-bin
