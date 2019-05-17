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

	"github.com/microsoft/ApplicationInsights-Go/appinsights"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/network"
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

type monitor struct {
	log    *logrus.Entry
	pipcli network.PublicIPAddressesClient
	icli   appinsights.TelemetryClient

	resourceGroup  string
	subscriptionID string

	instances []instance
}

type instance struct {
	hostname string
	b        *blackbox.Config
}

func (m *monitor) init(ctx context.Context, log *logrus.Entry) error {
	m.log = log
	authorizer, err := azureclient.NewAuthorizerFromEnvironment("")
	if err != nil {
		return err
	}
	if os.Getenv("RESOURCEGROUP") == "" {
		return fmt.Errorf("RESOURCEGROUP environment variable must be set")
	}
	m.resourceGroup = os.Getenv("RESOURCEGROUP")
	m.subscriptionID = os.Getenv("AZURE_SUBSCRIPTION_ID")
	m.pipcli = network.NewPublicIPAddressesClient(ctx, log, m.subscriptionID, authorizer)

	if os.Getenv("AZURE_APP_INSIGHTS_KEY") != "" {
		m.log.Info("application insights configured")
		m.icli = appinsights.NewTelemetryClient(os.Getenv("AZURE_APP_INSIGHTS_KEY"))
		m.icli.Context().CommonProperties["type"] = "monitoring"
		m.icli.Context().CommonProperties["resourcegroup"] = os.Getenv("RESOURCEGROUP")
	}

	return nil
}

func (m *monitor) listResourceGroupMonitoringHostnames(ctx context.Context, subscriptionID, resourceGroup string) (hostnames []string, err error) {
	// get dedicated routes we want to monitor
	m.log.Debug("waiting for OpenShiftManagedCluster config to be persisted")
	// TODO: once we can read provisioningState from disk file, remove network
	// polls
	err = wait.PollImmediateInfinite(time.Second, func() (bool, error) {
		oc, err := loadOCConfig()
		if err != nil {
			return false, nil
		}

		hostnames = append(hostnames, fmt.Sprintf("canary-openshift-azure-monitoring.%s", oc.Properties.RouterProfiles[0].PublicSubdomain))
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	// get all external IP's used by VMSS
	m.log.Debug("waiting for ss-masters ip addresses")
	err = wait.PollImmediateInfinite(10*time.Second, func() (bool, error) {
		var ips []string
		for iter, err := m.pipcli.ListVirtualMachineScaleSetPublicIPAddressesComplete(ctx, resourceGroup, "ss-master"); iter.NotDone(); err = iter.Next() {
			if err != nil {
				m.log.Error(err)
				return false, nil
			}

			if iter.Value().IPAddress != nil {
				ips = append(ips, *iter.Value().IPAddress)
			}
		}
		if len(ips) == 3 {
			hostnames = append(hostnames, ips...)
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return nil, err
	}

	// get api server hostname
	m.log.Debug("waiting for ip-apiserver server hostname")
	ip, err := m.pipcli.Get(ctx, resourceGroup, "ip-apiserver", "")
	if err != nil {
		return nil, err
	}
	hostnames = append(hostnames, fmt.Sprintf("%s.%s.cloudapp.azure.com", *ip.DNSSettings.DomainNameLabel, *ip.Location))

	return hostnames, nil
}

func (m *monitor) run(ctx context.Context) error {
	m.log.Info("fetching URLs")
	hostnames, err := m.listResourceGroupMonitoringHostnames(ctx, m.subscriptionID, m.resourceGroup)
	if err != nil {
		return err
	}

	for _, hostname := range hostnames {
		m.log.Debugf("initiate blackbox monitor %s", hostname)
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://%s/healthz", hostname), nil)
		if err != nil {
			return err
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

		b := &blackbox.Config{
			Cli: &http.Client{
				Transport: &http.Transport{
					// see net/http/transport.go: all Dialer values are default
					// except for LocalAddr.
					DialContext: (&net.Dialer{
						Timeout:   30 * time.Second,
						KeepAlive: 30 * time.Second,
						DualStack: true,
						LocalAddr: &net.TCPAddr{},
					}).DialContext,
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
					DisableKeepAlives: true,
				},
				Timeout: 5 * time.Second,
			},
			ICli:             m.icli,
			Req:              req,
			Interval:         *interval,
			LogInitialErrors: *logerrors,
		}

		m.instances = append(m.instances, struct {
			hostname string
			b        *blackbox.Config
		}{
			hostname: hostname,
			b:        b,
		})
	}

	for _, mon := range m.instances {
		mon.b.Start(ctx)
	}

	m.log.Info("collecting metrics... CTRL+C to stop")
	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch
	// persist blackbox monitors
	return m.persist(m.instances)

}

func (m *monitor) persist(instances []instance) error {
	for _, mon := range instances {
		path := path.Join(*outputdir, fmt.Sprintf("%s.log", mon.hostname))
		m.log.Infof("writing %s", path)
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

	m := new(monitor)
	ctx := context.Background()

	if err := m.init(ctx, log); err != nil {
		log.Fatalf("Cannot initialize monitor: %v", err)
	}

	if err := m.run(ctx); err != nil {
		panic(err)
	}
}
