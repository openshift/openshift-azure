# all is the default target to build everything
all: clean build sync

build:
	go build ./...

clean:
	rm -f sync

test: unit e2e

generate:
	go generate ./...

logbridge: clean generate
	go build ./cmd/logbridge

logbridge-image: logbridge
	go get github.com/openshift/imagebuilder/cmd/imagebuilder
	imagebuilder -f Dockerfile.logbridge -t quay.io/openshift-on-azure/logbridge:latest .

logbridge-push: logbridge-image
	docker push quay.io/openshift-on-azure/logbridge:latest

sync: clean generate
	go build -ldflags "-X main.gitCommit=$(shell git rev-parse --short HEAD)" ./cmd/sync

TAG ?= $(shell git rev-parse --short HEAD)
SYNC_IMAGE ?= quay.io/openshift-on-azure/sync:$(TAG)

sync-image: sync
	go get github.com/openshift/imagebuilder/cmd/imagebuilder
	imagebuilder -f Dockerfile.sync -t $(SYNC_IMAGE) .

sync-push: sync-image
	docker push $(SYNC_IMAGE)

verify:
	./hack/validate-generated.sh
	go vet ./...

unit: generate
	go test ./...

cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out

e2e: generate
	go test ./test/e2e -tags e2e

.PHONY: clean sync-image sync-push verify unit e2e
