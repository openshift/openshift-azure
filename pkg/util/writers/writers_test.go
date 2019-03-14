package writers

import (
	"archive/tar"
	"fmt"
	"io/ioutil"
	"reflect"
	"testing"
	"time"

	"github.com/golang/mock/gomock"

	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_writers"
)

func TestPathAndParents(t *testing.T) {
	for _, tt := range []struct {
		path          string
		expectedParts []string
		expectedErr   error
	}{
		{
			path:        "",
			expectedErr: fmt.Errorf("path must be absolute"),
		},
		{
			path:        "test",
			expectedErr: fmt.Errorf("path must be absolute"),
		},
		{
			path:          "/",
			expectedParts: []string{"/"},
		},
		{
			path:          "/a/b/c",
			expectedParts: []string{"/", "/a", "/a/b", "/a/b/c"},
		},
		{
			path:          "/a/b/x/.././/c/",
			expectedParts: []string{"/", "/a", "/a/b", "/a/b/c"},
		},
	} {
		parts, err := PathAndParents(tt.path)
		if !reflect.DeepEqual(tt.expectedParts, parts) {
			t.Errorf("%s: expected %#v, got %#v", tt.path, tt.expectedParts, parts)
		}
		if !reflect.DeepEqual(tt.expectedErr, err) {
			t.Errorf("%s: expected %s, got %s", tt.path, tt.expectedErr, err)
		}
	}
}

func TestTarWriter(t *testing.T) {
	gmc := gomock.NewController(t)
	defer gmc.Finish()

	pkgtarWriter := mock_writers.NewMockpkgtarWriter(gmc)

	c := pkgtarWriter.EXPECT().WriteHeader(&tar.Header{
		Name:    "a/",
		Mode:    0755,
		Uname:   "root",
		Gname:   "root",
		ModTime: time.Unix(1546300800, 0),
	}).Return(nil)

	c = pkgtarWriter.EXPECT().WriteHeader(&tar.Header{
		Name:    "a/b/",
		Mode:    0755,
		Uname:   "root",
		Gname:   "root",
		ModTime: time.Unix(1546300800, 0),
	}).Return(nil).After(c)

	c = pkgtarWriter.EXPECT().WriteHeader(&tar.Header{
		Name:    "a/file",
		Size:    5,
		Mode:    0644,
		Uname:   "root",
		Gname:   "root",
		ModTime: time.Unix(1546300800, 0),
	}).Return(nil).After(c)

	c = pkgtarWriter.EXPECT().Write([]byte("hello")).Return(5, nil).After(c)

	c = pkgtarWriter.EXPECT().WriteHeader(&tar.Header{
		Name:    "a/b/c/",
		Mode:    0755,
		Uname:   "root",
		Gname:   "root",
		ModTime: time.Unix(1546300800, 0),
	}).Return(nil).After(c)

	w := NewTarWriter(ioutil.Discard)
	w.(*tarWriter).pkgtarWriter = pkgtarWriter

	err := w.MkdirAll("/", 0755)
	if err != nil {
		t.Error(err)
	}

	err = w.MkdirAll("/a/b", 0755)
	if err != nil {
		t.Error(err)
	}

	err = w.WriteFile("/a/file", []byte("hello"), 0644)
	if err != nil {
		t.Error(err)
	}

	err = w.MkdirAll("/a/b/c", 0755)
	if err != nil {
		t.Error(err)
	}
}
