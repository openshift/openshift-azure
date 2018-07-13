clean:
	rm -f sync tunnel

tunnel: clean
	CGO_ENABLED=0 go build ./cmd/tunnel

tunnel-image: tunnel
	go get github.com/openshift/imagebuilder/cmd/imagebuilder
	imagebuilder -f Dockerfile.tunnel -t quay.io/openshift-on-azure/tunnel:latest .

tunnel-push: tunnel-image
	docker push quay.io/openshift-on-azure/tunnel:latest

sync: clean
	go generate ./...
	CGO_ENABLED=0 go build ./cmd/sync

sync-image: sync
	go get github.com/openshift/imagebuilder/cmd/imagebuilder
	imagebuilder -f Dockerfile.sync -t quay.io/openshift-on-azure/sync:latest .

sync-push: sync-image
	docker push quay.io/openshift-on-azure/sync:latest

.PHONY: clean sync-image sync-push tunnel-image tunnel-push

# docker pull quay.io/openshift-on-azure/sync:latest
# docker run --dns=8.8.8.8 -i -v /root/.kube:/.kube:z -e KUBECONFIG=/.kube/config quay.io/openshift-on-azure/sync:latest
