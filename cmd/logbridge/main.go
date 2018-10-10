package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"strconv"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/preview/operationalinsights/mgmt/2015-11-01-preview/operationalinsights"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/coreos/go-systemd/sdjournal"
	"github.com/ghodss/yaml"
)

var (
	cursorSyncInterval = flag.Duration("cursorSyncInterval", 10*time.Second, "interval after which the current journald cursor is synced to disk")
)

func getWorkspaceInfo() (string, []byte, error) {
	b, err := ioutil.ReadFile("cloudprovider/azure.conf")
	if err != nil {
		return "", nil, err
	}

	var m map[string]string
	if err := yaml.Unmarshal(b, &m); err != nil {
		return "", nil, err
	}

	config := auth.NewClientCredentialsConfig(m["aadClientId"], m["aadClientSecret"], m["tenantId"])
	authorizer, err := config.Authorizer()
	if err != nil {
		return "", nil, err
	}

	wcli := operationalinsights.NewWorkspacesClient(m["subscriptionId"])
	wcli.Authorizer = authorizer

	ws, err := wcli.ListByResourceGroup(context.Background(), m["resourceGroup"])
	if err != nil {
		return "", nil, err
	}

	if len(*ws.Value) != 1 {
		return "", nil, fmt.Errorf("error: found %d workspaces, expected 1", len(*ws.Value))
	}

	keys, err := wcli.GetSharedKeys(context.Background(), m["resourceGroup"], *(*ws.Value)[0].Name)
	if err != nil {
		return "", nil, err
	}

	key, err := base64.StdEncoding.DecodeString(*keys.PrimarySharedKey)
	if err != nil {
		return "", nil, err
	}

	return *(*ws.Value)[0].CustomerID, key, nil
}

func readCursor() (string, error) {
	b, err := ioutil.ReadFile("state/cursor")
	return string(b), err
}

func writeCursor(cursor string) error {
	f, err := os.Create("state/cursor.new")
	if err != nil {
		return err
	}

	_, err = f.WriteString(cursor)
	if err != nil {
		f.Close()
		return err
	}

	err = f.Sync()
	if err != nil {
		f.Close()
		return err
	}

	err = f.Close()
	if err != nil {
		return err
	}

	err = os.Rename("state/cursor.new", "state/cursor")
	if err != nil {
		return err
	}

	d, err := os.Open("state")
	if err != nil {
		return err
	}
	defer d.Close()

	return d.Sync()
}

func startReader(ch chan<- []map[string]interface{}) error {
	j, err := sdjournal.NewJournal()
	if err != nil {
		return err
	}
	var cursor string
	cursor, err = readCursor()
	if err != nil {
		return err
	}
	// check if cursor file is valid
	// https://godoc.org/github.com/coreos/go-systemd/sdjournal#Journal.GetCursor
	if len(cursor) == 0 {
		_, err := j.Next()
		if err != nil {
			return err
		}
		cursor, err = j.GetCursor()
		if err != nil {
			return err
		}
	}
	if err == nil {
		err = j.SeekCursor(cursor)
		if err != nil {
			return err
		}

		// normally we should hit the last entry we logged, in which case
		// advance the cursor by one.  If that entry no longer exists, we may
		// have lost logs.  In this case, SeekCursor should put us on the very
		// next entry it finds, in which case don't further advance the cursor
		if j.TestCursor(cursor) == nil {
			j.Next()
		}
	}

	go func() {
		var entries []map[string]interface{}

		for {
			if entries == nil {
				entries = make([]map[string]interface{}, 0, 100)
			}

			n, err := j.Next()
			if err != nil {
				panic(err)
			}

			if n == 0 {
				if len(entries) > 0 {
					ch <- entries
					entries = nil
				}

				j.Wait(sdjournal.IndefiniteWait)
				continue
			}

			entry, err := j.GetEntry()
			if err != nil {
				panic(err)
			}

			e := make(map[string]interface{}, len(entry.Fields))
			for k, v := range entry.Fields {
				e[k] = v
			}

			// https://www.freedesktop.org/software/systemd/man/systemd.journal-fields.html
			e["__CURSOR"] = entry.Cursor
			e["__REALTIME_TIMESTAMP"] = entry.RealtimeTimestamp
			e["__MONOTONIC_TIMESTAMP"] = entry.MonotonicTimestamp
			e["__TIME_GENERATED"] = time.Unix(0, int64(entry.RealtimeTimestamp)*int64(time.Microsecond)).UTC().Format("2006-01-02T15:04:05.000Z")

			entries = append(entries, e)

			if len(entries) == cap(entries) {
				ch <- entries
				entries = nil
			}
		}
	}()

	return nil
}

func run() error {
	customerID, key, err := getWorkspaceInfo()
	if err != nil {
		return err
	}
	hm := hmac.New(sha256.New, key)

	ch := make(chan []map[string]interface{})

	err = startReader(ch)
	if err != nil {
		return err
	}

	lastSync := time.Now()
	for entries := range ch {
		b, err := json.Marshal(entries)
		if err != nil {
			return err
		}

		// https://docs.microsoft.com/en-us/azure/log-analytics/log-analytics-data-collector-api
		req, err := http.NewRequest(http.MethodPost, "https://"+customerID+".ods.opinsights.azure.com/api/logs?api-version=2016-04-01", bytes.NewReader(b))
		if err != nil {
			return err
		}

		req.Header.Add("Log-Type", "osa")
		req.Header.Add("x-ms-date", time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")) // time.RFC1123 but ending in GMT, not UTC
		req.Header.Add("time-generated-field", "__TIME_GENERATED")
		req.Header.Add("Content-Type", "application/json")

		sig := req.Method + "\n" +
			strconv.FormatInt(int64(len(b)), 10) + "\n" +
			req.Header.Get("Content-Type") + "\n" +
			"x-ms-date:" + req.Header.Get("x-ms-date") + "\n" +
			req.URL.Path

		hm.Reset()
		hm.Write([]byte(sig))
		req.Header.Add("Authorization", "SharedKey "+customerID+":"+base64.StdEncoding.EncodeToString(hm.Sum(nil)))

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}

		if resp.StatusCode/100 != 2 {
			b, _ = httputil.DumpResponse(resp, true)
			log.Println(string(b))
			return fmt.Errorf("error: unexpected status code %d, expected 2xx", resp.StatusCode)
		}

		if time.Now().Sub(lastSync) > *cursorSyncInterval {
			lastSync = time.Now()

			err = writeCursor(entries[len(entries)-1]["__CURSOR"].(string))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}
