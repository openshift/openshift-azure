package errors

import (
	"net"
	"net/url"
	"os"
	"syscall"
)

// IsMatchingSyscallError returns true when the error is one of the Errno's in match
// it deals with the different ways of wrapping the syscalls.
func IsMatchingSyscallError(err error, match ...syscall.Errno) bool {
	for {
		switch t := err.(type) {
		case *url.Error:
			err = t.Err
		case *net.OpError:
			err = t.Err
		case *os.SyscallError:
			err = t.Err
		case syscall.Errno:
			for _, sc := range match {
				if t == sc {
					return true
				}
			}
			return false
		default:
			return false
		}
	}
}
