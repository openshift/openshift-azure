all:
	mkdir -p _out
	go run main.go >_out/values.yaml
	helm template osa -f _out/values.yaml --output-dir _out

clean:
	rm -rf _out

.PHONY: all clean
