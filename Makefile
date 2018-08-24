# all is the default target to build everything
all: clean build sync

build:
	go build ./...

clean:
	rm -f sync

test:
	go test ./...

generate:
	go generate ./...

sync: clean generate
	CGO_ENABLED=0 go build ./cmd/sync

SYNC_IMAGE ?= quay.io/openshift-on-azure/sync:latest

sync-image: sync
	go get github.com/openshift/imagebuilder/cmd/imagebuilder
	imagebuilder -f Dockerfile.sync -t $(SYNC_IMAGE) .

sync-push: sync-image
	docker push $(SYNC_IMAGE)

verify:
	./hack/validate-generated.sh

unit:
	go test ./...

.PHONY: clean sync-image sync-push verify unit
