package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/ghodss/yaml"
	"github.com/microsoft/ApplicationInsights-Go/appinsights"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/blackbox"
	"github.com/openshift/openshift-azure/pkg/util/log"
)

var (
	gitCommit  = "unknown"
	logLevel   = flag.String("loglevel", "Debug", "valid values are Debug, Info, Warning, Error")
	interval   = flag.Duration("interval", 100*time.Millisecond, "check interval with dimension. Example: 1000ms ")
	logerrors  = flag.Bool("logerrors", false, "log initial errors")
	outputdir  = flag.String("outputdir", "./", "output directory")
	configfile = flag.String("configfile", "_data/containerservice.yaml", "container services config file location")
)

type monitor interface {
	name() string
	getHostnames(ctx context.Context, oc *api.OpenShiftManagedCluster) ([]string, error)
	getDialContext() func(ctx context.Context, network, address string) (net.Conn, error)
}

type instance struct {
	hostname string
	b        *blackbox.Config
}

func getBlackboxConfig(log *logrus.Entry, m monitor, icli appinsights.TelemetryClient, hostname string) (*blackbox.Config, error) {
	log.Debugf("initiating blackbox monitor for %s", hostname)
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://%s/healthz", hostname), nil)
	if err != nil {
		return nil, err
	}

	// Monitoring opens a lot of short-lived TCP connections to the LB IP
	// and to backend IPs simultaneously.  Note the following:
	//
	// 1. Linux allows reuse of local TCP source ports as long as the
	//    remote IP/port tuples are unique across the local IP/port tuple.
	//
	// 2. Azure external LBs work by NATing, not by proxying.  The LB
	//    backend always sees the client IP/port tuple unchanged.
	//
	// The `LocalAddr` setting below, which is essential, causes
	// bind("0.0.0.0:0") to be called before connect() for each new
	// monitoring connection.  This guarantees that the local source port
	// for the monitoring connection in question is unique across all
	// connections on the local system.
	//
	// If bind() is not called here, it is possible for a local->backend
	// monitoring connection and a local->LB connection, perhaps in the fake
	// RP or in a test process, to share the same source port.  This is
	// acceptable to the client, but because of rule (2) above, the remote
	// end will see two different connections with identical endpoints.
	//
	// If there is an existing local->LB connection and a subsequent
	// local->backend connection is attempted, the remote end ignores the
	// new connection attempt (as it's on a connection which as far as it's
	// concerned already exists) and resends an acknowledgement on the
	// original connection.  However, for some reason the acknowledgement
	// packet is not NATed (the incoming connection attempt has reset the LB
	// NAT table?) and appears to the client to be an erroneous packet on
	// the new connection.  The client replies with a reset, which bypasses
	// the LB and has the effect of resetting the old connection on the
	// server end.
	//
	// At this point, the server is not aware of any live connections.  The
	// client believes the old connection is still live.  The client now
	// attempts the new connection again, and the second time around it
	// succeeds.
	//
	// All further local->LB packets on the old connection are silently
	// dropped by the LB (the public preview "TCP Reset on Idle" feature
	// would probably help us here).  Dependent on the setting of
	// tcp_retries2 (see tcp(7)), a timeout error is returned to the client
	// around 15 minutes later.
	return &blackbox.Config{
		Cli: &http.Client{
			Transport: &http.Transport{
				// see net/http/transport.go: all Dialer values are default
				// except for LocalAddr.
				DialContext: m.getDialContext(),
				/* #nosec - connecting to external IP of a FakeRP cluster, expect self signed cert */
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
				DisableKeepAlives: true,
			},
			Timeout: 5 * time.Second,
		},
		ICli:             icli,
		Req:              req,
		Interval:         *interval,
		LogInitialErrors: *logerrors,
	}, nil
}

func run(ctx context.Context, log *logrus.Entry, ms []monitor, icli appinsights.TelemetryClient, oc *api.OpenShiftManagedCluster) error {
	var instances []instance
	for _, m := range ms {
		log.Infof("fetching %s URLs", m.name())
		hostnames, err := m.getHostnames(ctx, oc)
		if err != nil {
			return err
		}
		for _, hostname := range hostnames {
			bbc, err := getBlackboxConfig(log, m, icli, hostname)
			if err != nil {
				return err
			}

			instances = append(instances, struct {
				hostname string
				b        *blackbox.Config
			}{
				hostname: hostname,
				b:        bbc,
			})
		}
	}
	for _, mon := range instances {
		mon.b.Start(ctx)
	}

	log.Info("collecting metrics... CTRL+C to stop")
	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch
	// persist blackbox monitors
	return persist(log, instances)

}

func persist(log *logrus.Entry, instances []instance) error {
	for _, mon := range instances {
		path := path.Join(*outputdir, fmt.Sprintf("%s.log", mon.hostname))
		log.Infof("writing %s", path)
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		mon.b.Stop(f)
		f.Close()
	}
	return nil
}

func loadOCConfig() (*api.OpenShiftManagedCluster, error) {
	b, err := ioutil.ReadFile(*configfile)
	if err != nil {
		return nil, err
	}

	var cs *api.OpenShiftManagedCluster
	if err := yaml.Unmarshal(b, &cs); err != nil {
		return nil, err
	}

	return cs, nil
}

func main() {
	flag.Parse()
	logger := logrus.New()
	logger.Formatter = &logrus.TextFormatter{FullTimestamp: true}
	logger.SetLevel(log.SanitizeLogLevel(*logLevel))
	log := logrus.NewEntry(logger).WithField("component", "monitor")
	log.Printf("monitoring pod starting, git commit %s", gitCommit)

	ctx := context.Background()
	authorizer, err := azureclient.NewAuthorizerFromEnvironment("")
	if err != nil {
		log.Fatalf("Cannot get authorizer from envionment: %v", err)
	}
	ctx = context.WithValue(ctx, api.ContextKeyClientAuthorizer, authorizer)

	oc, err := loadOCConfig()
	if err != nil {
		log.Fatalf("Cannot load clusterConfiguration from %s: %v", *configfile, err)
	}

	var mAPI monitor
	if oc.Properties.PrivateAPIServer {
		mAPI, err = newPrivateAPI(ctx, log, oc)
	} else {
		mAPI, err = newPublicAPI(ctx, log, oc)
	}
	if err != nil {
		log.Fatalf("Cannot initialize API monitor: %v", err)
	}
	mApps, err := newPublicApps(ctx, log)
	if err != nil {
		log.Fatalf("Cannot initialize apps monitor: %v", err)
	}

	var icli appinsights.TelemetryClient
	if os.Getenv("AZURE_APP_INSIGHTS_KEY") != "" {
		log.Info("application insights configured")
		icli = appinsights.NewTelemetryClient(os.Getenv("AZURE_APP_INSIGHTS_KEY"))
		icli.Context().CommonProperties["type"] = "monitoring"
		icli.Context().CommonProperties["resourcegroup"] = os.Getenv("RESOURCEGROUP")
	}
	if err := run(ctx, log, []monitor{mAPI, mApps}, icli, oc); err != nil {
		panic(err)
	}
}
