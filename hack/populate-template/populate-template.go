package main

import (
	"crypto/rsa"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ghodss/yaml"

	pluginapi "github.com/openshift/openshift-azure/pkg/api/plugin/api"
	"github.com/openshift/openshift-azure/pkg/fakerp/shared"
	"github.com/openshift/openshift-azure/pkg/tls"
)

const (
	SecretsDirectory       = "secrets/"
	TemplatesDirectory     = "test/templates/"
	DefaultTemplateExample = "template.yaml.example.311.43"
)

var (
	output = flag.String("output", fmt.Sprintf("%stemplate.yaml", TemplatesDirectory), "Specify output template file full path")
	input  = flag.String("intput", DefaultTemplateExample, "Specify input template file full path")
)

func main() {

	artifactDir, err := shared.FindDirectory(TemplatesDirectory)
	if err != nil {
		panic(err)
	}
	data, err := readFile(filepath.Join(artifactDir, *input))
	if err != nil {
		panic(err)
	}
	var template *pluginapi.Config
	if err := yaml.Unmarshal(data, &template); err != nil {
		panic(err)
	}

	artifactDir, err = shared.FindDirectory(SecretsDirectory)
	if err != nil {
		panic(err)
	}
	logCert, err := readCert(filepath.Join(artifactDir, "logging-int.cert"))
	if err != nil {
		panic(err)
	}
	logKey, err := readKey(filepath.Join(artifactDir, "logging-int.key"))
	if err != nil {
		panic(err)
	}
	metCert, err := readCert(filepath.Join(artifactDir, "metrics-int.cert"))
	if err != nil {
		panic(err)
	}
	metKey, err := readKey(filepath.Join(artifactDir, "metrics-int.key"))
	if err != nil {
		panic(err)
	}
	pullSecret, err := readFile(filepath.Join(artifactDir, ".dockerconfigjson"))
	if err != nil {
		panic(err)
	}
	template.Certificates.GenevaLogging.Cert = logCert
	template.Certificates.GenevaLogging.Key = logKey
	template.Certificates.GenevaMetrics.Cert = metCert
	template.Certificates.GenevaMetrics.Key = metKey
	template.Images.GenevaImagePullSecret = pullSecret

	b, err := yaml.Marshal(template)
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(*output, b, 0666)
	if err != nil {
		panic(err)
	}
}

func readCert(path string) (*x509.Certificate, error) {
	b, err := readFile(path)
	if err != nil {
		return nil, err
	}
	return tls.ParseCert(b)
}

func readKey(path string) (*rsa.PrivateKey, error) {
	b, err := readFile(path)
	if err != nil {
		return nil, err
	}
	return tls.ParsePrivateKey(b)
}

func readFile(path string) ([]byte, error) {
	if fileExist(path) {
		return ioutil.ReadFile(path)
	}
	return []byte{}, fmt.Errorf("file %s does not exist", path)
}

func fileExist(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
