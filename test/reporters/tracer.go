package reporters

import (
	"bufio"
	"io"
	"os"
	"syscall"

	"github.com/Microsoft/ApplicationInsights-Go/appinsights"
	"github.com/Microsoft/ApplicationInsights-Go/appinsights/contracts"
)

// skip log lines which are just artifacts of ginko
var exclude = []string{"", "S", "========================", "------------------------------", "•"}

type TraceWriter struct {
	c appinsights.TelemetryClient

	w *io.PipeWriter
	r *io.PipeReader

	br *bufio.Reader
}

func (tw *TraceWriter) reader() {
	for {
		s, err := tw.br.ReadString('\n')
		if !stringInSlice(s, exclude) {
			tw.c.TrackTrace(s, contracts.Information)
		}
		if err != nil {
			break
		}
	}
}

func (tw *TraceWriter) Close() error {
	return tw.w.Close()
}

func NewTraceWriter(c appinsights.TelemetryClient, rd io.Reader) *TraceWriter {
	r, w := io.Pipe()

	tw := &TraceWriter{
		c:  c,
		w:  w,
		r:  r,
		br: bufio.NewReader(r),
	}

	go func() {
		_, err := io.Copy(tw.w, rd)
		tw.r.CloseWithError(err)
	}()

	go tw.reader()

	return tw
}

type Capture struct {
	fd int
	f  *os.File
	r  *os.File
	io.Reader
}

func (c *Capture) Close() error {
	err := syscall.Dup2(int(c.f.Fd()), c.fd)
	if err != nil {
		return err
	}

	return nil
}

// FD taps a file descriptor opened for write (e.g. stdout or stderr) and
// returns an io.Reader which streams the bytes written to the file descriptor.
func NewCapture(fd int) (*Capture, error) {
	c := &Capture{fd: fd}

	newfd, err := syscall.Dup(c.fd)
	if err != nil {
		return nil, err
	}

	c.f = os.NewFile(uintptr(newfd), "")

	var w *os.File
	c.r, w, err = os.Pipe()
	if err != nil {
		c.f.Close()
		return nil, err
	}
	defer w.Close()

	err = syscall.Dup2(int(w.Fd()), c.fd)
	if err != nil {
		c.f.Close()
		c.r.Close()
		return nil, err
	}

	var w2 *io.PipeWriter
	c.Reader, w2 = io.Pipe()

	go io.Copy(io.MultiWriter(c.f, w2), c.r)

	return c, nil
}

func stringInSlice(str string, list []string) bool {
	for _, v := range list {
		if v == str {
			return true
		}
	}
	return false
}
