package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/blackbox"
	"github.com/openshift/openshift-azure/pkg/util/log"
)

var (
	gitCommit = "unknown"
	logLevel  = flag.String("loglevel", "Debug", "valid values are Debug, Info, Warning, Error")
	interval  = flag.Duration("interval", 100*time.Millisecond, "check interval with dimension. Example: 1000ms ")
	logerrors = flag.Bool("logerrors", false, "log initial errors")
	outputdir = flag.String("outputdir", "./", "output directory")
)

type monitor struct {
	log    *logrus.Entry
	pipcli azureclient.PublicIPAddressesClient

	resourceGroup  string
	subscriptionID string

	instances []instance
}

type instance struct {
	hostname string
	b        *blackbox.Config
}

func (m *monitor) init(ctx context.Context, log *logrus.Entry) error {
	authorizer, err := azureclient.NewAuthorizerFromEnvironment("")
	if err != nil {
		return err
	}
	if os.Getenv("RESOURCEGROUP") == "" {
		return fmt.Errorf("RESOURCEGROUP environment variable must be set")
	}
	m.resourceGroup = os.Getenv("RESOURCEGROUP")
	m.subscriptionID = os.Getenv("AZURE_SUBSCRIPTION_ID")
	m.pipcli = azureclient.NewPublicIPAddressesClient(ctx, m.subscriptionID, authorizer)
	m.log = log

	return nil
}

func (m *monitor) listResourceGroupMonitoringHostnames(ctx context.Context, subscriptionID, resourceGroup string) (hostnames []string, err error) {
	// get all external IP's used by VMSS
	for {
		hostnames = []string{}
		for iter, err := m.pipcli.ListVirtualMachineScaleSetPublicIPAddressesComplete(ctx, resourceGroup, "ss-master"); iter.NotDone(); err = iter.Next() {
			if err != nil {
				m.log.Debug("waiting for url")
				time.Sleep(5 * time.Second)
			} else if iter.Value().IPAddress != nil {
				hostnames = append(hostnames, *iter.Value().IPAddress)
			}
		}
		if err == nil && len(hostnames) == 3 {
			break
		}
	}
	// get api server hostname
	for {
		ip, err := m.pipcli.Get(ctx, resourceGroup, "ip-apiserver", "")
		if err != nil {
			m.log.Debug("waiting for url")
			time.Sleep(5 * time.Second)
		} else if err == nil && ip.Location != nil {
			hostnames = append(hostnames, fmt.Sprintf("%s.%s.cloudapp.azure.com", *ip.DNSSettings.DomainNameLabel, *ip.Location))
			break
		}
	}
	return hostnames, nil
}

func (m *monitor) run(ctx context.Context) error {
	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)

	// we will close this guard when we load all monitors and clean-up is needed
	bootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ch:
				os.Exit(0)
			}
		}
	}(bootCtx)

	m.log.Info("fetching URLs\n")
	hostnames, err := m.listResourceGroupMonitoringHostnames(ctx, m.subscriptionID, m.resourceGroup)
	if err != nil {
		return err
	}

	for _, hostname := range hostnames {
		m.log.Debugf("initiate blackbox monitor %s \n", hostname)
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://%s/healthz", hostname), nil)
		if err != nil {
			return err
		}
		b := &blackbox.Config{
			Cli: &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
					DisableKeepAlives: true,
				},
				Timeout: time.Second,
			},
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

	m.log.Info("collecting metrics... CTRL+C to stop\n")
	cancel()
	for {
		select {
		case <-ch:
			return m.persist(m.instances)
		}
	}
}

func (m *monitor) persist(instances []instance) error {
	for _, mon := range instances {
		path := path.Join(*outputdir, fmt.Sprintf("%s.log", mon.hostname))
		m.log.Infof("writing %s\n", path)
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		mon.b.Stop(f)
		f.Close()
	}
	return nil
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
