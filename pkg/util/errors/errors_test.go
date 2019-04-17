package errors

import (
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"syscall"
	"testing"
)

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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsMatchingSyscallError(tt.err, tt.match...)
			if tt.want != got {
				t.Errorf("IsMatchingSyscallError(%s) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
