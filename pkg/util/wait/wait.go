package wait

//go:generate go get github.com/golang/mock/gomock
//go:generate go install github.com/golang/mock/mockgen
//go:generate mockgen -destination=../mocks/mock_$GOPACKAGE/wait.go -package=mock_$GOPACKAGE -source wait.go
//go:generate gofmt -s -l -w ../mocks/mock_$GOPACKAGE/wait.go
//go:generate goimports -e -w ../mocks/mock_$GOPACKAGE/wait.go

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"syscall"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

func PollImmediateUntil(interval time.Duration, condition wait.ConditionFunc, stopCh <-chan struct{}) error {
	done, err := condition()
	if err != nil {
		return err
	}
	if done {
		return nil
	}
	return wait.PollUntil(interval, condition, stopCh)
}

// SimpleHttpClient to aid in mocking
type SimpleHttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// ForHTTPStatusOk poll until URL returns 200
func ForHTTPStatusOk(ctx context.Context, transport http.RoundTripper, urltocheck string) error {
	cli := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}
	return forHTTPStatusOkWithTimeout(ctx, cli, urltocheck)
}

func forHTTPStatusOkWithTimeout(ctx context.Context, cli SimpleHttpClient, urltocheck string) error {
	req, err := http.NewRequest("GET", urltocheck, nil)
	if err != nil {
		return err
	}
	return PollImmediateUntil(time.Second, func() (bool, error) {
		resp, err := cli.Do(req)
		if err, ok := err.(*url.Error); ok {
			if err, ok := err.Err.(*net.OpError); ok {
				if err, ok := err.Err.(*os.SyscallError); ok {
					if err.Err == syscall.ENETUNREACH {
						return false, nil
					}
				}
			}
			if err.Timeout() || err.Err == io.EOF || err.Err == io.ErrUnexpectedEOF {
				return false, nil
			}
		}
		if err == io.EOF {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		return resp != nil && resp.StatusCode == http.StatusOK, nil
	}, ctx.Done())
}
