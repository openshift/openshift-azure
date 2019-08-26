package errors

import (
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"syscall"
	"testing"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
)

type fakeAddr struct {
	network string
	address string
}

func (a *fakeAddr) Network() string {
	return a.network
}
func (a *fakeAddr) String() string {
	return a.address
}

func TestIsMatchingSyscallError(t *testing.T) {
	urltocheck := "http://localhost:12345/nowhere"

	tests := []struct {
		name  string
		err   error
		match []syscall.Errno
		want  bool
	}{
		{
			name: "nil",
			want: false,
		},
		{
			name:  "unknown",
			want:  false,
			match: []syscall.Errno{syscall.ECONNREFUSED},
			err:   fmt.Errorf("not this"),
		},
		{
			name:  "unreachable",
			want:  true,
			match: []syscall.Errno{syscall.ENETUNREACH},
			err: &url.Error{
				URL: urltocheck,
				Err: &net.OpError{
					Err: os.NewSyscallError("socket", syscall.ENETUNREACH),
				},
			},
		},
		{
			name:  "connection refused - nested",
			want:  true,
			match: []syscall.Errno{syscall.ECONNREFUSED},
			err: &url.Error{
				URL: urltocheck,
				Err: &net.OpError{
					Err: os.NewSyscallError("socket", syscall.ECONNREFUSED),
				},
			},
		},
		{
			name:  "connection refused",
			want:  true,
			match: []syscall.Errno{syscall.ECONNREFUSED},
			err: &net.OpError{
				Err: os.NewSyscallError("socket", syscall.ECONNREFUSED),
			},
		},
		{
			name:  "net/http: HTTP/1.x transport connection broken: connection refused",
			want:  true,
			match: []syscall.Errno{syscall.ECONNREFUSED},
			err:   fmt.Errorf("net/http: HTTP/1.x transport connection broken: %v", os.NewSyscallError("write", syscall.ECONNREFUSED)),
		},
		{
			name:  "net/http: HTTP/1.x transport connection broken: something else",
			want:  false,
			match: []syscall.Errno{syscall.ECONNREFUSED},
			err:   fmt.Errorf("net/http: HTTP/1.x transport connection broken: %v", os.NewSyscallError("open", syscall.EADDRNOTAVAIL)),
		},
		{
			name:  "net/http: HTTP/1.x transport connection broken (structured)",
			want:  true,
			match: []syscall.Errno{syscall.ECONNRESET},
			err: &url.Error{
				URL: urltocheck,
				Op:  "Put",
				Err: fmt.Errorf("net/http: HTTP/1.x transport connection broken: %v", &net.OpError{
					Addr:   &fakeAddr{network: "tcp", address: "40.71.240.16:443"},
					Source: &fakeAddr{network: "tcp", address: "172.16.108.55:36980"},
					Op:     "write",
					Net:    "tcp",
					Err:    os.NewSyscallError("write", syscall.ECONNRESET),
				}),
			},
		},
		{
			name:  "net/http: HTTP/1.x transport connection broken (formatted)",
			want:  true,
			match: []syscall.Errno{syscall.ECONNRESET},
			err: fmt.Errorf("%v", &url.Error{
				URL: urltocheck,
				Op:  "Put",
				Err: fmt.Errorf("net/http: HTTP/1.x transport connection broken: %v", &net.OpError{
					Addr:   &fakeAddr{network: "tcp", address: "40.71.240.16:443"},
					Source: &fakeAddr{network: "tcp", address: "172.16.108.55:36980"},
					Op:     "write",
					Net:    "tcp",
					Err:    os.NewSyscallError("write", syscall.ECONNRESET),
				}),
			}),
		},
		{
			name:  "no match",
			want:  false,
			match: []syscall.Errno{syscall.ENETUNREACH},
			err: &url.Error{
				URL: urltocheck,
				Err: &net.OpError{
					Err: os.NewSyscallError("socket", syscall.ECONNREFUSED),
				},
			},
		},
		{
			name: "url io.EOF",
			want: false,
			err:  &url.Error{Err: io.EOF},
		},
		{
			name:  "azure request error : connection reset",
			want:  true,
			match: []syscall.Errno{syscall.ECONNRESET},
			err: &azure.RequestError{
				DetailedError: autorest.DetailedError{
					Original:    os.NewSyscallError("io.read", syscall.ECONNRESET),
					PackageType: "packageType",
					Method:      "GET",
					StatusCode:  "500",
					Message:     "testing",
				},
			},
		},
		{
			name:  "autorest detailed error : connection reset",
			want:  true,
			match: []syscall.Errno{syscall.ECONNRESET},
			err: &autorest.DetailedError{
				Original: os.NewSyscallError("io.read", syscall.ECONNRESET),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsMatchingSyscallError(tt.err, tt.match...)
			if tt.want != got {
				if tt.err != nil {
					t.Errorf("%v", tt.err.Error())
				}
				t.Errorf("IsMatchingSyscallError(%s) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
