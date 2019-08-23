package errors

import (
	"net"
	"net/url"
	"os"
	"strings"
	"syscall"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
)

const (
	netHTTPErrorPrefix = "net/http: HTTP/1.x transport connection broken"
)

var (
	netHTTPErrorPrefixLen = len(netHTTPErrorPrefix)
	str2Syscall           = map[string]syscall.Errno{
		"connection reset by peer":  syscall.ECONNRESET,
		"connection refused":        syscall.ECONNREFUSED,
		"network is unreachable":    syscall.ENETUNREACH,
		"no such file or directory": syscall.ENOENT,
	}
)

func unwrapNetHTTPError(err error) (bool, error) {
	if !strings.Contains(err.Error(), netHTTPErrorPrefix) {
		return false, err
	}
	for sysMsg, errno := range str2Syscall {
		if strings.Contains(err.Error(), sysMsg) {
			return true, errno
		}
	}
	return false, err
}

// IsMatchingSyscallError returns true when the error is one of the Errno's in match
// it deals with the different ways of wrapping the syscalls.
func IsMatchingSyscallError(err error, match ...syscall.Errno) bool {
	for {
		switch t := err.(type) {
		case nil:
			return false
		case *azure.RequestError:
			err = &t.DetailedError
		case *autorest.DetailedError:
			err = t.Original
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
			// hack alert. This is a horrible string check because there is a
			// fmt.Errorf() wrapper in net/http
			// see: https://groups.google.com/forum/#!topic/golang-nuts/AkPSBPfZt0o
			var found bool
			found, err = unwrapNetHTTPError(err)
			if !found {
				return false
			}
		}
	}
}
