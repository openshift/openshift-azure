package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"text/template"

	"github.com/ghodss/yaml"
)

type prow struct {
	Presubmits struct {
		OpenshiftOpenshiftAzure []struct {
			RerunCommand string `json:"rerun_command"`
		} `json:"openshift/openshift-azure"`
	} `json:"presubmits"`
}

func get(url string) ([]byte, error) {
	/* #nosec - this helper is supposed to take an arbitrary url */
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	return ioutil.ReadAll(resp.Body)
}

func run() error {
	b, err := get("https://raw.githubusercontent.com/openshift/release/master/ci-operator/jobs/openshift/openshift-azure/openshift-openshift-azure-master-presubmits.yaml")
	if err != nil {
		return err
	}

	var prow prow
	err = yaml.Unmarshal(b, &prow)
	if err != nil {
		return err
	}

	var commands []string
	for _, c := range prow.Presubmits.OpenshiftOpenshiftAzure {
		commands = append(commands, c.RerunCommand)
	}

	return template.Must(template.New("commands.md").ParseFiles("hack/generate-test-commands/commands.md")).Execute(os.Stdout, commands)
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}
