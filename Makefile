all: clean
	mkdir _out
	go run main.go >_out/values.yaml
	helm template osa -f _out/values.yaml --output-dir _out

push: all
	oc delete -f _out/osa/templates || true
	oc create -f _out/osa/templates

clean:
	rm -rf _out

.PHONY: all push clean
