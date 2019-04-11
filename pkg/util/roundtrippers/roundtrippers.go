package roundtrippers

import (
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// AuthorizingRoundTripper acts as a normal RoundTripper, but with a bearer token
type AuthorizingRoundTripper struct {
	http.RoundTripper
	Token string
}

func (rt AuthorizingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+rt.Token)
	return rt.RoundTripper.RoundTrip(req)
}

// The retryingRoundTripper implementation is customised to help with multiple
// network connection-related issues seen in CI which we haven't necessarily
// been able to fully explain yet.  Importantly, it is not yet clear whether any
// of these issues could impact cluster end-users or whether they are more
// symptomatic of a CI-related issue.
//
// 1. We think the following flow may be possible:
//    * client has an open TCP connection to master-000000:443 via LB.
//    * master-000000 is deallocated as part of a rotation.
//    * deallocation takes place too quickly for TCP connection to be closed
//      properly.
//    * next client request errors.
//    We're trying to solve this via the disableKeepAlives flag: in such
//    circumstances the caller can set this and a new TCP connection will be
//    opened for each request.
//
// 2. Notwithstanding the fix in 1, we are still seeing read: connection timed
//    out errors, such as "WaitForInfraDaemonSets: Get
//    https://hostname.eastus.cloudapp.azure.com/apis/apps/v1/namespaces/default/daemonsets/router:
//    read tcp 172.16.217.87:48274->52.188.220.9:443: read: connection timed
//    out".  At least most of these errors are on GETs, and it appears that the
//    timeout in question is long (around 15-16 minutes).  Current best guess is
//    we're hitting the kernel tcp_retries2 limit; it looks like the client
//    never receives acknowledgement that the server has received the outgoing
//    request; no packets from the server arrive; eventually the subsequent
//    client read times out.
//
//    retryingRoundTripper aims to help the above by setting a 30 second timeout
//    on client GETs and retrying if the timeout is reached.  This is done only
//    on GETs since other actions are not idempotent.
//
//    The default Dial timeout of 30 seconds is reduced to 10 seconds to give
//    confidence that requests are normally likely to complete within the
//    timeout.
//
// 3. Even with the default 30 second Dial timeout, sometimes we see unexplained
//    Dial timeout failures.  retryingRoundTripper retries in these cases.
//
// 4. The default TLS handshake timeout is 10 seconds.  Sometimes we see this
//    timeout triggered.  retryingRoundTripper also retries in these cases.

var timerExpired = errors.New("retryingRoundTripper timer expired")

type RetryingRoundTripper struct {
	Log *logrus.Entry
	http.RoundTripper
	Retries    int
	GetTimeout time.Duration
}

func (rt *RetryingRoundTripper) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	var retry int
	for {
		retry++

		if req.Method == http.MethodGet {
			done := make(chan struct{})

			cancel := make(chan struct{})
			req.Cancel = cancel

			t := time.NewTimer(rt.GetTimeout)

			go func() {
				select {
				case <-done:
				case <-t.C:
					close(cancel)
				}
			}()

			resp, err = rt.RoundTripper.RoundTrip(req)

			if !t.Stop() {
				err = timerExpired
			}

			close(done)

		} else {
			resp, err = rt.RoundTripper.RoundTrip(req)
		}

		if err, ok := err.(*net.OpError); retry <= rt.Retries && ok {
			if err.Op == "dial" && err.Err.Error() == "i/o timeout" {
				rt.Log.Warnf("%s: retry %d", err, retry)
				continue
			}
		}

		if retry <= rt.Retries && err != nil && err.Error() == "net/http: TLS handshake timeout" {
			rt.Log.Warnf("%s: retry %d", err, retry)
			continue
		}

		if retry <= rt.Retries && err == timerExpired {
			rt.Log.Warnf("%s: retry %d", err, retry)
			continue
		}

		if err != nil {
			rt.Log.Warnf("%#v: not retrying", err)
		}

		return
	}
}
