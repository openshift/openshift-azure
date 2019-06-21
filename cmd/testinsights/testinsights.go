package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/microsoft/ApplicationInsights-Go/appinsights"
)

//UnitTestEntry represents a log entry in the JSON output of go test
type UnitTestEntry struct {
	Time    time.Time
	Action  string
	Package string
	Output  string
}

type metricName struct {
	ComponentTexts string
	RunTime        string
	FailureMessage string
	Failed         bool
	Passed         bool
	Skipped        bool
}

var startTime map[string]time.Time

func generateEntry(e UnitTestEntry) (*metricName, float64) {
	switch e.Action {
	case "skip":
		return &metricName{ComponentTexts: e.Package, RunTime: "0s", FailureMessage: e.Output, Failed: false, Passed: false, Skipped: true}, 0
	case "pass":
		return &metricName{ComponentTexts: e.Package, RunTime: e.Time.Sub(startTime[e.Package]).String(), FailureMessage: e.Output, Failed: false, Passed: true, Skipped: false}, 0
	case "fail":
		return &metricName{ComponentTexts: e.Package, RunTime: e.Time.Sub(startTime[e.Package]).String(), FailureMessage: e.Output, Failed: true, Passed: false, Skipped: false}, 1
	case "run":
		startTime[e.Package] = e.Time
	}
	return nil, 0
}

func main() {
	offline := false
	failed := false
	var c appinsights.TelemetryClient
	startTime = make(map[string]time.Time)

	if os.Getenv("AZURE_APP_INSIGHTS_KEY") == "" {
		fmt.Println("Env variable AZURE_APP_INSIGHTS_KEY not set, ignoring app insights")
		offline = true
	} else {
		c = appinsights.NewTelemetryClient(os.Getenv("AZURE_APP_INSIGHTS_KEY"))
		c.Context().CommonProperties["type"] = "unit"
		c.Context().CommonProperties["resourcegroup"] = os.Getenv("RESOURCEGROUP")
		// fields below are populated by PROW env variables
		// see https://github.com/kubernetes/test-infra/blob/master/prow/jobs.md#job-environment-variables
		if os.Getenv("JOB_NAME") == "" {
			// local make unit
			c.Context().CommonProperties["prowjobname"] = "local-run"
			c.Context().CommonProperties["prowjobtype"] = ""
			c.Context().CommonProperties["prowjobbuild"] = ""
			c.Context().CommonProperties["prowprnumber"] = ""
		} else {
			// prow run
			c.Context().CommonProperties["prowjobname"] = os.Getenv("JOB_NAME")
			c.Context().CommonProperties["prowjobtype"] = os.Getenv("JOB_TYPE")
			c.Context().CommonProperties["prowjobbuild"] = os.Getenv("BUILD_ID")
			c.Context().CommonProperties["prowprnumber"] = os.Getenv("PULL_NUMBER")
		}
	}
	var entry UnitTestEntry
	dec := json.NewDecoder(os.Stdin)
	for {
		err := dec.Decode(&entry)
		fmt.Printf("%s %s", entry.Package, entry.Output)
		n, v := generateEntry(entry)
		if n != nil {
			if v > 0 {
				failed = true
			}
			nameJSON, err := json.Marshal(n)
			if err != nil {
				log.Fatal(err)
			}
			if !offline {
				c.TrackMetric(string(nameJSON), v)
			}
		}
		if err == io.EOF {
			// reached end of input
			break
		}
		if err != nil {
			log.Fatal(err)
		}
	}
	if !offline {
		<-c.Channel().Close(30 * time.Second)
	}
	if failed {
		os.Exit(1)
	}
}
