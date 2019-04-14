GITCOMMIT=$(shell git describe --tags HEAD)$(shell [[ $$(git status --porcelain) = "" ]] || echo -dirty)
LDFLAGS="-X main.gitCommit=$(GITCOMMIT)"

AZURE_IMAGE ?= quay.io/openshift-on-azure/azure:$(GITCOMMIT)

.PHONY: azure-image azure-push all version clean test unit generate secrets artifacts
# all is the default target to build everything
all: azure

version:
	@echo $(GITCOMMIT)

secrets:
	@rm -rf secrets
	@mkdir secrets
	@oc extract -n azure secret/cluster-secrets-azure --to=secrets >/dev/null

clean:
	rm -f coverage.out azure releasenotes

generate:
	@[[ -e /var/run/secrets/kubernetes.io ]] || go generate ./...

test: unit e2e

.PHONY: create delete upgrade
create:
	./hack/create.sh ${RESOURCEGROUP}

delete:
	./hack/delete.sh ${RESOURCEGROUP}

upgrade:
	./hack/upgrade.sh ${RESOURCEGROUP}

artifacts:
	./hack/artifacts.sh

azure-image: azure
	./hack/image-build.sh images/azure/Dockerfile $(AZURE_IMAGE)

azure-push: azure-image
	docker push $(AZURE_IMAGE)

azure: generate
	go build -ldflags ${LDFLAGS} ./cmd/$@

sync-run: generate
	go run -ldflags ${LDFLAGS} ./cmd/azure sync --run-once --loglevel Debug

.PHONY: sync-run

monitoring:
	go build -ldflags ${LDFLAGS} ./cmd/$@
.PHONY: monitoring

monitoring-run: monitoring
	./hack/monitoring.sh

monitoring-stop:
	./hack/monitoring.sh clean

.PHONY: monitoring-run monitoring-stop

releasenotes:
	go build -tags releasenotes ./cmd/$@

.PHONY: verify
verify:
	./hack/verify/validate-generated.sh
	go vet ./...
	./hack/verify/validate-code-format.sh
	./hack/verify/validate-util.sh
	./hack/verify/validate-codecov.sh
	go run ./hack/validate-imports/validate-imports.go cmd hack pkg test

unit: generate
	go test ./... -coverprofile=coverage.out -covermode=atomic
ifneq ($(ARTIFACTS),)
	mkdir -p $(ARTIFACTS)
	cp coverage.out $(ARTIFACTS)
endif

.PHONY: cover codecov
cover: unit
	go tool cover -html=coverage.out

codecov: unit
	./hack/codecov-report.sh

.PHONY: e2e e2e-prod e2e-etcdbackuprecovery e2e-keyrotation e2e-reimagevm e2e-changeloglevel e2e-scaleupdown e2e-forceupdate e2e-vnet
e2e:
	FOCUS="\[CustomerAdmin\]|\[EndUser\]\[Fake\]" TIMEOUT=60m ./hack/e2e.sh

e2e-prod:
	FOCUS="\[Default\]\[Real\]" TIMEOUT=70m ./hack/e2e.sh

e2e-etcdbackuprecovery:
	FOCUS="\[EtcdRecovery\]\[Fake\]" TIMEOUT=180m ./hack/e2e.sh

e2e-keyrotation:
	FOCUS="\[KeyRotation\]\[Fake\]" TIMEOUT=180m ./hack/e2e.sh

e2e-reimagevm:
	FOCUS="\[ReimageVM\]\[Fake\]" TIMEOUT=40m ./hack/e2e.sh

e2e-changeloglevel:
	FOCUS="\[ChangeLogLevel\]\[Fake\]" TIMEOUT=180m ./hack/e2e.sh

e2e-scaleupdown:
	FOCUS="\[ScaleUpDown\]\[Fake\]" TIMEOUT=50m ./hack/e2e.sh

e2e-forceupdate:
	FOCUS="\[ForceUpdate\]\[Fake\]" TIMEOUT=180m ./hack/e2e.sh

e2e-vnet:
	FOCUS="\[Vnet\]\[Real\]" TIMEOUT=70m ./hack/e2e.sh

vmimage:
	./hack/vmimage.sh

.PHONY: vmimage
