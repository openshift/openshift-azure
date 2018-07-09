package util

import (
	"bytes"
	"encoding/base64"
	"strconv"
	"text/template"

	acsapi "github.com/Azure/acs-engine/pkg/api"
	"github.com/ghodss/yaml"

	"github.com/jim-minter/azure-helm/pkg/config"
	"github.com/jim-minter/azure-helm/pkg/tls"
)

// TODO: util packages are an anti-pattern, don't do this

func Template(tmpl string, f template.FuncMap, cs *acsapi.ContainerService, c *config.Config, extra interface{}) ([]byte, error) {
	t, err := template.New("").Funcs(template.FuncMap{
		"CertAsBytes":          tls.CertAsBytes,
		"PrivateKeyAsBytes":    tls.PrivateKeyAsBytes,
		"PublicKeyAsBytes":     tls.PublicKeyAsBytes,
		"SSHPublicKeyAsString": tls.SSHPublicKeyAsString,
		"YamlMarshal":          yaml.Marshal,
		"Base64Encode":         base64.StdEncoding.EncodeToString,
		"String":               func(b []byte) string { return string(b) },
		"quote":                strconv.Quote,
	}).Funcs(f).Parse(tmpl)
	if err != nil {
		return nil, err
	}

	b := &bytes.Buffer{}

	err = t.Execute(b, struct {
		ContainerService *acsapi.ContainerService
		Config           *config.Config
		Extra            interface{}
	}{ContainerService: cs, Config: c, Extra: extra})
	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}
