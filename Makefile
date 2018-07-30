clean:
	rm -f sync

test:
	go test ./...

sync: clean
	go generate ./...
	CGO_ENABLED=0 go build ./cmd/sync

sync-image: sync
	go get github.com/openshift/imagebuilder/cmd/imagebuilder
	imagebuilder -f Dockerfile.sync -t quay.io/openshift-on-azure/sync:latest .

sync-push: sync-image
	docker push quay.io/openshift-on-azure/sync:latest

verify:
	./hack/validate-generated.sh

.PHONY: clean sync-image sync-push verify
