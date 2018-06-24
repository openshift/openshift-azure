sync:
	go generate ./...
	CGO_ENABLED=0 go build ./cmd/sync

image: sync
	go get github.com/openshift/imagebuilder/cmd/imagebuilder
	imagebuilder -t docker.io/jimminter/sync:latest .

push: image
	docker push docker.io/jimminter/sync:latest

.PHONY: image push

# docker pull docker.io/jimminter/sync:latest ; docker run --dns=8.8.8.8 -i -v /root/.kube:/.kube:z -e KUBECONFIG=/.kube/config docker.io/jimminter/sync:latest
