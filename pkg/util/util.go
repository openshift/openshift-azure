package util

import (
	"bytes"
	"encoding/base64"
	"text/template"

	"github.com/ghodss/yaml"
	"github.com/jim-minter/azure-helm/pkg/api"
	"github.com/jim-minter/azure-helm/pkg/config"
	"github.com/jim-minter/azure-helm/pkg/tls"
)

// TODO: util packages are an anti-pattern, don't do this

func Template(tmpl string, f template.FuncMap, m *api.Manifest, c *config.Config, extra interface{}) ([]byte, error) {
	t, err := template.New("").Funcs(template.FuncMap{
		"CertAsBytes":          tls.CertAsBytes,
		"PrivateKeyAsBytes":    tls.PrivateKeyAsBytes,
		"PublicKeyAsBytes":     tls.PublicKeyAsBytes,
		"SSHPublicKeyAsString": tls.SSHPublicKeyAsString,
		"YamlMarshal":          yaml.Marshal,
		"Base64Encode":         base64.StdEncoding.EncodeToString,
		"String":               func(b []byte) string { return string(b) },
		"Bytes":                func(s string) []byte { return []byte(s) },
		"JoinBytes":            func(b ...[]byte) []byte { return bytes.Join(b, []byte("\n")) },
	}).Funcs(f).Parse(tmpl)
	if err != nil {
		return nil, err
	}

	b := &bytes.Buffer{}

	err = t.Execute(b, struct {
		Manifest *api.Manifest
		Config   *config.Config
		Extra    interface{}
	}{Manifest: m, Config: c, Extra: extra})
	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}
