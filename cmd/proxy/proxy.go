package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	gitCommit = "unknown"
	pem       = flag.String("pem", "certs/server.pem", "path to pem file")
	key       = flag.String("key", "certs/server.key", "path to key file")
	ca        = flag.String("ca", "certs/ca.pem", "path to ca file")
	subnet    = flag.String("subnet", "172.30.10.0/24", "allowed proxy connect subnet")

	log *logrus.Entry
)

func init() {
	flag.Parse()
	logger := logrus.New()
	logger.Formatter = &logrus.TextFormatter{FullTimestamp: true}
	logger.SetLevel(logrus.DebugLevel)
	log = logrus.NewEntry(logger)
	log.Infof("proxy starting, git commit %s", gitCommit)
}

func run() error {
	_, subnet, _ := net.ParseCIDR(*subnet)

	// Create a CA certificate pool and add cert.pem to it
	caCert, err := ioutil.ReadFile(*ca)
	if err != nil {
		log.Fatal(err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Create the TLS Config with the CA pool and enable Client certificate validation
	tlsConfig := &tls.Config{
		ClientCAs:  caCertPool,
		ClientAuth: tls.RequireAndVerifyClientCert,
	}
	tlsConfig.BuildNameToCertificate()

	server := &http.Server{
		Addr:      ":8443",
		TLSConfig: tlsConfig,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// early validation
			if r.Method != http.MethodConnect {
				log.Debug("r.Method != http.MethodConnect")
				http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
				return
			}

			ip, _, err := net.SplitHostPort(r.Host)
			if err != nil {
				log.Errorf(err.Error())
				http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
				return
			}
			host := net.ParseIP(ip)

			if !subnet.Contains(host) {
				log.Errorf("host %s not in %s", host.String(), subnet.String())
				http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
				return
			}

			handleTunneling(w, r)

		}),
		// Disable HTTP/2.
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}
	return server.ListenAndServeTLS(*pem, *key)
}

func handleTunneling(w http.ResponseWriter, r *http.Request) {
	log.Debugf("dial tunnel %s to %s", r.RemoteAddr, r.Host)

	dConn, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
	if err != nil {
		log.Error(err.Error())
		http.Error(w, "ups", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		log.Error("Hijacking not supported")
		http.Error(w, "ups", http.StatusInternalServerError)
		return
	}

	cConn, _, err := hijacker.Hijack()
	if err != nil {
		log.Error(err.Error())
		http.Error(w, "ups", http.StatusServiceUnavailable)
	}
	go transfer(dConn, cConn)
	go transfer(cConn, dConn)
}

func transfer(destination io.WriteCloser, source io.ReadCloser) {
	defer destination.Close()
	defer source.Close()
	io.Copy(destination, source)
}

func main() {
	err := run()
	if err != nil {
		panic(err)
	}
}
