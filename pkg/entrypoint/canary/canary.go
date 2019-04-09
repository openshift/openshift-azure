package canary

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/util/log"
)

const externalMountPoint = "/data"

type ping struct{}

var _ http.Handler = &ping{}

func start(cfg *Config) error {
	logrus.SetLevel(log.SanitizeLogLevel(cfg.LogLevel))
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	log := logrus.NewEntry(logrus.StandardLogger())

	log.Print("canary pod starting")

	p := ping{}
	http.Handle("/", prometheus.InstrumentHandler("webkv", &p))
	http.Handle("/metrics", promhttp.Handler())
	http.Handle("/healthz/ready", http.HandlerFunc(p.readyHandler))
	return http.ListenAndServe(":8080", nil)
}

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
