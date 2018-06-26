clean:
	rm -f proxy sync

proxy: clean
	go generate ./...
	CGO_ENABLED=0 go build ./cmd/proxy

proxy-image: proxy
	go get github.com/openshift/imagebuilder/cmd/imagebuilder
	imagebuilder -f Dockerfile.proxy -t docker.io/jimminter/proxy:latest .

proxy-push: proxy-image
	docker push docker.io/jimminter/proxy:latest

# docker pull docker.io/jimminter/proxy:latest
# docker run --privileged --network=host -i docker.io/jimminter/proxy:latest 172.29.255.254:443 www.google.com:443

sync: clean
	go generate ./...
	CGO_ENABLED=0 go build ./cmd/sync

sync-image: sync
	go get github.com/openshift/imagebuilder/cmd/imagebuilder
	imagebuilder -f Dockerfile.sync -t docker.io/jimminter/sync:latest .

sync-push: sync-image
	docker push docker.io/jimminter/sync:latest

.PHONY: clean proxy-image proxy-push sync-image sync-push

# docker pull docker.io/jimminter/sync:latest
# docker run --dns=8.8.8.8 -i -v /root/.kube:/.kube:z -e KUBECONFIG=/.kube/config docker.io/jimminter/sync:latest
