package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"io"
	"io/ioutil"
	"net"
	"net/http"

	"github.com/sirupsen/logrus"
)

// Proxy implements an HTTP reverse proxy, allowing external access to private
// API servers for testing purposes.  To access, we require a valid client
// certificate and we restrict ongoing access to a specific destination subnet.
// Clients send an HTTP `CONNECT ip:port` request to be connected, just like a
// usual HTTPS proxy.

var (
	gitCommit = "unknown"
	certFile  = flag.String("cert", "secrets/proxy-server.pem", "path to pem file")
	keyFile   = flag.String("key", "secrets/proxy-server.key", "path to key file")
	caFile    = flag.String("cacert", "secrets/proxy-ca.pem", "path to ca file")
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
	_, subnet, err := net.ParseCIDR(*subnet)
	if err != nil {
		return err
	}

	// We authorize anyone with a CA-signed client certificate
	caCertPool := x509.NewCertPool()
	caCert, err := ioutil.ReadFile(*caFile)
	if err != nil {
		return err
	}
	caCertPool.AppendCertsFromPEM(caCert)

	// Create the TLS Config with the CA pool and enable Client certificate validation
	tlsConfig := &tls.Config{
		ClientCAs:  caCertPool,
		ClientAuth: tls.RequireAndVerifyClientCert,
	}

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
				log.Error(err)
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}

			if !subnet.Contains(net.ParseIP(ip)) {
				log.Errorf("host %s not in %s", ip, subnet.String())
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}

			handleTunneling(w, r)

		}),
		// Disable HTTP/2.
		TLSNextProto: map[string]func(*http.Server, *tls.Conn, http.Handler){},
	}
	return server.ListenAndServeTLS(*certFile, *keyFile)
}

func handleTunneling(w http.ResponseWriter, r *http.Request) {
	log.Debugf("dial tunnel %s to %s", r.RemoteAddr, r.Host)

	dConn, err := net.Dial("tcp", r.Host)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		log.Error("hijacking not supported")
		http.Error(w, "hijacking not supported", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	cConn, buf, err := hijacker.Hijack()
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	// empty buffer if something was sent already
	go func() {
		_, err := io.Copy(dConn, buf)
		if err != nil {
			log.Error(err)
		}
		err = dConn.(*net.TCPConn).CloseWrite()
		if err != nil {
			log.Error(err)
		}
	}()

	_, err = io.Copy(cConn, dConn)
	if err != nil {
		log.Error(err)
	}
	err = cConn.(*tls.Conn).CloseWrite()
	if err != nil {
		log.Error(err)
	}
}

func main() {
	err := run()
	if err != nil {
		panic(err)
	}
}
