package admissioncontroller

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"net/http"
	"testing"

	"github.com/sirupsen/logrus"

	utiltls "github.com/openshift/openshift-azure/pkg/util/tls"
)

func TestSecurity(t *testing.T) {
	l := newTestListener()
	defer l.Close()

	ac := &admissionController{
		log: logrus.NewEntry(logrus.StandardLogger()),
		l:   l,
		cs:  cs,
	}

	go ac.run()

	pool := x509.NewCertPool()
	pool.AddCert(cs.Config.Certificates.Ca.Cert)

	selfsignedkey, selfsignedcert, err := utiltls.NewCert(&utiltls.CertParams{
		Subject:     pkix.Name{CommonName: "client"},
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	})
	if err != nil {
		t.Fatal(err)
	}

	othersignedkey, othersignedcert, err := utiltls.NewCert(&utiltls.CertParams{
		Subject:     pkix.Name{CommonName: "client"},
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		SigningKey:  cs.Config.Certificates.Ca.Key,
		SigningCert: cs.Config.Certificates.Ca.Cert,
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		name           string
		url            string
		key            *rsa.PrivateKey
		cert           *x509.Certificate
		wantStatusCode int
	}{
		{
			name:           "empty url, no client certificate",
			url:            "https://server/",
			wantStatusCode: http.StatusNotFound,
		},
		{
			name:           "ready url, no client certificate",
			url:            "https://server/healthz/ready",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "sccs url, no TLS certificate",
			url:            "https://server/sccs",
			wantStatusCode: http.StatusForbidden,
		},
		{
			name:           "podwhitelist url, no TLS certificate",
			url:            "https://server/podwhitelist",
			wantStatusCode: http.StatusForbidden,
		},
		{
			name:           "sccs url, self signed TLS certificate",
			url:            "https://server/sccs",
			key:            selfsignedkey,
			cert:           selfsignedcert,
			wantStatusCode: http.StatusForbidden,
		},
		{
			name:           "sccs url, other signed TLS certificate",
			url:            "https://server/sccs",
			key:            othersignedkey,
			cert:           othersignedcert,
			wantStatusCode: http.StatusForbidden,
		},
		{
			name:           "sccs url, valid TLS certificate",
			url:            "https://server/sccs",
			key:            cs.Config.Certificates.AroAdmissionControllerClient.Key,
			cert:           cs.Config.Certificates.AroAdmissionControllerClient.Cert,
			wantStatusCode: http.StatusMethodNotAllowed,
		},
		{
			name:           "podwhitelist url, valid TLS certificate",
			url:            "https://server/podwhitelist",
			key:            cs.Config.Certificates.AroAdmissionControllerClient.Key,
			cert:           cs.Config.Certificates.AroAdmissionControllerClient.Cert,
			wantStatusCode: http.StatusMethodNotAllowed,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tlsConfig := &tls.Config{
				RootCAs: pool,
			}
			if tt.cert != nil && tt.key != nil {
				tlsConfig.Certificates = []tls.Certificate{
					{
						Certificate: [][]byte{
							tt.cert.Raw,
						},
						PrivateKey: tt.key,
					},
				}
			}

			c := &http.Client{
				Transport: &http.Transport{
					Dial:            l.Dial,
					TLSClientConfig: tlsConfig,
				},
			}

			resp, err := c.Get(tt.url)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatusCode {
				t.Error(resp.StatusCode)
			}
		})
	}
}
