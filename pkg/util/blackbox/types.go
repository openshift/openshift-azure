package blackbox

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/Microsoft/ApplicationInsights-Go/appinsights"
)

// Blackbox is interface for blackbox monitoring
type Blackbox interface {
	// Start blackbox monitoring instance
	Start(ctx context.Context)

	// Stop blackbox monitoring instance
	Stop(w io.Writer) error
}

// Config defines blackbox instance configuration
type Config struct {
	Cli              *http.Client
	Icli             appinsights.TelemetryClient
	Req              *http.Request
	Interval         time.Duration
	LogInitialErrors bool

	// samples for individual monitor
	samples []*sample

	done      chan struct{}
	collected chan struct{}

	// cluster creation e2e time
	startTime time.Time
	stopTime  time.Time
	duration  time.Time

	logfile string
}

type sample struct {
	time    time.Time
	latency time.Duration
	err     error
}

// Start runs black box monitor and samples data into buffer until stopped
func (c *Config) Start(ctx context.Context) {
	c.done = make(chan struct{})
	c.collected = make(chan struct{})
	c.startTime = time.Now().UTC()

	t := time.NewTicker(c.Interval)

	go func(ctx context.Context) {
		var seenGoodSample bool
		for {
			select {
			case <-t.C:
			case <-c.done:
				t.Stop()
				close(c.collected)
				return
			}

			start := time.Now()
			resp, err := c.Cli.Do(c.Req)
			end := time.Now()

			if c.Icli != nil {
				if resp != nil {
					request := appinsights.NewRequestTelemetry(c.Req.Method, c.Req.URL.String(), time.Second, resp.Status)
					request.Id = os.Getenv("RESOURCEGROUP")
					request.MarkTime(start, end)
					c.Icli.Track(request)
				} else {
					request := appinsights.NewRequestTelemetry(c.Req.Method, c.Req.URL.String(), time.Second, "error")
					request.Id = os.Getenv("RESOURCEGROUP")
					request.MarkTime(start, end)
					c.Icli.Track(request)
				}
			}

			if err == nil && resp.StatusCode != http.StatusOK {
				err = fmt.Errorf("invalid status %d", resp.StatusCode)
			}

			if !c.LogInitialErrors && !seenGoodSample && err != nil {
				continue
			}
			if !seenGoodSample && err == nil {
				seenGoodSample = true
			}

			c.samples = append(c.samples, &sample{
				time:    start,
				latency: end.Sub(start),
				err:     err,
			})
		}
	}(ctx)
}

// Stop stops blackbox instance and persists data into writer
func (c *Config) Stop(w io.Writer) {
	close(c.done)
	<-c.collected

	c.stopTime = time.Now().UTC()

	if len(c.samples) == 0 {
		return
	}

	var sum time.Duration
	for _, sample := range c.samples {
		sum += sample.latency

		fmt.Fprintf(w, "%s", sample.time.UTC().Format("2006-01-02T15:04:05.000"))

		ct := int(sample.latency / (100 * time.Millisecond))
		fmt.Fprintf(w, " %-10s", strings.Repeat("*", min(ct, 10)))
		if ct > 10 {
			fmt.Fprintf(w, "+")
		} else {
			fmt.Fprintf(w, " ")
		}

		fmt.Fprintf(w, " %-4dms", sample.latency/time.Millisecond)

		if sample.err != nil {
			fmt.Fprintf(w, " %s", sample.err)
		}

		fmt.Fprintln(w)
	}

	fmt.Fprintln(w)

	fmt.Fprintf(w, "mean   latency: %-4dms\n", sum/time.Duration(len(c.samples))/time.Millisecond)

	sort.Slice(c.samples, func(i, j int) bool { return c.samples[i].latency < c.samples[j].latency })

	fmt.Fprintf(w, "median latency: %-4dms\n", c.samples[len(c.samples)/2].latency/time.Millisecond)

	fmt.Fprintf(w, "95%%ile latency: %-4dms\n", c.samples[len(c.samples)*95/100].latency/time.Millisecond)

	fmt.Fprintf(w, "99%%ile latency: %-4dms\n", c.samples[len(c.samples)*99/100].latency/time.Millisecond)
	fmt.Fprintf(w, "start time: %v . end time: %v\n", c.startTime, c.stopTime)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
