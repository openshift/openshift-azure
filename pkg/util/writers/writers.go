package writers

import (
	"archive/tar"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

//go:generate go get github.com/golang/mock/mockgen
//go:generate mockgen -destination=../mocks/mock_$GOPACKAGE/writers.go -package=mock_$GOPACKAGE -source writers.go
//go:generate go get golang.org/x/tools/cmd/goimports
//go:generate goimports -local=github.com/openshift/openshift-azure -e -w ../../util/mocks/mock_$GOPACKAGE/writers.go

type Writer interface {
	Close() error
	MkdirAll(path string, perm os.FileMode) error
	WriteFile(path string, b []byte, perm os.FileMode) error
}

type filesystemWriter struct{}

func NewFilesystemWriter() Writer {
	return &filesystemWriter{}
}

func (*filesystemWriter) MkdirAll(path string, perm os.FileMode) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("path %q must be absolute", path)
	}

	return os.MkdirAll(path, perm)
}

func (*filesystemWriter) WriteFile(path string, b []byte, perm os.FileMode) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("path %q must be absolute", path)
	}

	return ioutil.WriteFile(path, b, perm)
}

func (*filesystemWriter) Close() error {
	return nil
}

type pkgtarWriter interface {
	io.WriteCloser
	WriteHeader(*tar.Header) error
}

type tarWriter struct {
	dirs map[string]struct{}
	pkgtarWriter
}

func NewTarWriter(w io.Writer) Writer {
	return &tarWriter{
		dirs: map[string]struct{}{
			"/": {},
		},
		pkgtarWriter: tar.NewWriter(w),
	}
}

func (tw *tarWriter) MkdirAll(path string, perm os.FileMode) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("path %q must be absolute", path)
	}

	parts, err := PathAndParents(path)
	if err != nil {
		return err
	}

	for _, path = range parts {
		if _, found := tw.dirs[path]; found {
			continue
		}

		err := tw.WriteHeader(&tar.Header{
			Name:    path[1:] + "/",
			Mode:    int64(perm.Perm()),
			Uname:   "root",
			Gname:   "root",
			ModTime: time.Unix(1546300800, 0),
		})
		if err != nil {
			return err
		}

		tw.dirs[path] = struct{}{}
	}

	return nil
}

func (tw *tarWriter) WriteFile(path string, b []byte, perm os.FileMode) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("path %q must be absolute", path)
	}

	err := tw.WriteHeader(&tar.Header{
		Name:    path[1:],
		Size:    int64(len(b)),
		Mode:    int64(perm.Perm()),
		Uname:   "root",
		Gname:   "root",
		ModTime: time.Unix(1546300800, 0),
	})
	if err != nil {
		return err
	}

	_, err = tw.Write(b)
	return err
}

// PathAndParents takes an absolute path, e.g. /a/b/c and returns it along with
// its parents, ordered from root to path, e.g. /, /a, /a/b, /a/b/c
func PathAndParents(path string) ([]string, error) {
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("path must be absolute")
	}
	if path == "/" {
		path = ""
	}

	pieces := strings.Split(path, "/")

	parts := []string{"/"}
	for i := 2; i <= len(pieces); i++ {
		parts = append(parts, strings.Join(pieces[:i], "/"))
	}

	return parts, nil
}
