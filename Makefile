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

sync-image: sync
	go get github.com/openshift/imagebuilder/cmd/imagebuilder
	imagebuilder -f Dockerfile.sync -t quay.io/openshift-on-azure/sync:latest .

sync-push: sync-image
	docker push quay.io/openshift-on-azure/sync:latest

verify:
	./hack/validate-generated.sh

unit:
	go test ./...

.PHONY: clean sync-image sync-push verify unit
