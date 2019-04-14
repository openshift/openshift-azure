package reporters

import (
	"bufio"
	"io"
	"os"
	"syscall"

	"github.com/Microsoft/ApplicationInsights-Go/appinsights"
	"github.com/Microsoft/ApplicationInsights-Go/appinsights/contracts"
)

type Capture struct {
	done    chan struct{}
	fd      int
	oldfile *os.File
	io.Reader
}

func (c *Capture) Close() error {
	// fd -> closefd
	closefd, err := syscall.Dup(c.fd)
	if err != nil {
		return err
	}

	// oldfile.fd -> fd
	err = syscall.Dup2(int(c.oldfile.Fd()), c.fd)
	if err != nil {
		return err
	}

	err = syscall.Close(closefd)
	if err != nil {
		return err
	}

	<-c.done

	return nil
}

func NewCapture(fd int) (*Capture, error) {
	c := &Capture{
		done: make(chan struct{}),
		fd:   fd,
	}

	// fd -> oldfd
	oldfd, err := syscall.Dup(c.fd)
	if err != nil {
		return nil, err
	}

	c.oldfile = os.NewFile(uintptr(oldfd), "")

	ospr, ospw, err := os.Pipe()
	if err != nil {
		c.oldfile.Close()
		return nil, err
	}
	defer ospw.Close()

	// ospw -> fd
	err = syscall.Dup2(int(ospw.Fd()), c.fd)
	if err != nil {
		c.oldfile.Close()
		ospr.Close()
		return nil, err
	}

	iopr, iopw := io.Pipe()

	go func() {
		// copy from ospr -> c.oldfile and iopw (read by c.Reader)
		_, err = io.Copy(io.MultiWriter(c.oldfile, iopw), ospr)
		iopw.CloseWithError(err)
		close(c.done)
	}()

	c.Reader = iopr

	return c, nil
}

func StartCapture(fd int, c appinsights.TelemetryClient, done chan struct{}) (*Capture, error) {
	capture, err := NewCapture(fd)
	if err != nil {
		return nil, err
	}

	go func() {
		br := bufio.NewReader(capture)
		for {
			s, err := br.ReadString('\n')
			if err != nil {
				break
			}
			c.TrackTrace(s, contracts.Information)
		}
		close(done)
	}()
	return capture, nil
}
