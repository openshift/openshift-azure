package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const externalMountPoint = "/data"

type ping struct{}

var _ http.Handler = &ping{}

func (p *ping) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	str := fmt.Sprintf("%s/%s : %s. Node: %s", r.RemoteAddr, r.URL, r.UserAgent(), os.Getenv("HOSTNAME"))
	err := ioutil.WriteFile(path.Join(externalMountPoint, "lastrequest.log"), []byte(str), 0600)
	if err != nil {
		fmt.Fprintf(w, err.Error())
	} else {
		fmt.Fprintf(w, str)
	}
}

func (p *ping) readyHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func main() {
	p := ping{}
	http.Handle("/", prometheus.InstrumentHandler("webkv", &p))
	http.Handle("/metrics", promhttp.Handler())
	http.Handle("/healthz/ready", http.HandlerFunc(p.readyHandler))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
