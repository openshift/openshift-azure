// Code generated by go-bindata. (@generated) DO NOT EDIT.

// Package arm generated by go-bindata.// sources:
// data/master-startup.sh
// data/node-startup.sh
package arm

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func bindataRead(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	clErr := gz.Close()

	if err != nil {
		return nil, fmt.Errorf("read %q: %v", name, err)
	}
	if clErr != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type asset struct {
	bytes []byte
	info  os.FileInfo
}

type bindataFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}

// Name return file name
func (fi bindataFileInfo) Name() string {
	return fi.name
}

// Size return file size
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}

// Mode return file mode
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}

// ModTime return file modify time
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}

// IsDir return file whether a directory
func (fi bindataFileInfo) IsDir() bool {
	return fi.mode&os.ModeDir != 0
}

// Sys return file is sys mode
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var _masterStartupSh = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xbc\x59\x5f\x77\x1b\xb5\x12\x7f\x46\x9f\x62\x58\x07\x4a\x4b\xe4\x4d\x0a\xdc\x0b\x86\xf4\x9c\x34\x49\x39\xbd\x94\x24\x37\x69\x2f\x0f\xa5\xa7\x47\x5e\xcd\xda\xaa\xb5\xd2\x22\x69\x9d\x98\xd4\xdf\xfd\x9e\x91\x76\x1d\x3b\x76\xd2\x96\x16\xfa\x90\xda\xd2\x68\x66\x34\xf3\x9b\x7f\x72\xef\xf3\x7c\xa8\x4c\x3e\x14\x7e\x0c\x1c\x2f\x19\xeb\xc1\x13\xeb\x20\xa0\x0f\xca\x8c\x06\xa0\xed\x08\x84\x91\x20\x9d\xad\x41\x68\x0d\xc1\x89\xb2\x54\x05\x84\xb1\x08\x70\x61\x1b\x2d\xc1\xd9\x26\x20\x4c\x95\x80\x30\x46\xa8\x84\x0f\xe8\xe0\xe8\xd9\x63\xd6\x83\xb3\xa3\xf3\x93\x17\x67\x07\x47\x3f\x9f\x9d\xbc\x38\xdd\xcb\x66\xb6\x71\xdc\xa1\xb7\x8d\x2b\x90\x8f\x9c\x6d\xea\x8c\xf5\xe0\xe4\xfc\xf5\x93\xff\x1e\x1e\xef\x65\xb6\x46\xe3\xc7\xaa\x0c\xfd\xad\x95\x93\x7d\xeb\x85\xc4\x69\xbf\xd0\xb6\x91\x19\xeb\xb1\x1e\xa8\x3a\x88\xa1\x46\x0f\xfc\x29\x3c\x3d\x3e\x7d\xf1\x1c\xb8\x87\xad\xaf\xa4\x1a\xc1\xd7\x7e\x6c\x5d\x80\x6c\xab\xe5\x9b\xc1\x5b\x08\x42\x69\xe0\xbb\xf7\x81\xbf\x81\x67\x27\x3f\x03\xe7\xda\x8e\x78\xed\xb0\x54\x97\x90\xfd\xf2\xe2\xf1\x11\x10\x29\x1c\x9e\x9d\x9c\x0e\xb2\x8f\xe3\x4f\x3c\x18\xbb\xba\x02\x55\x42\xff\xc0\x9a\x52\x8d\xfa\xe7\x58\x34\x4e\x85\xd9\xa9\x08\xc5\xf8\x54\x14\x13\x31\x42\x0f\xf3\x39\xd3\x76\x34\x42\x07\x3c\xb4\x86\xe3\x3e\x08\x17\x9a\xba\xef\xc7\x90\x29\xe3\x83\xd0\x5a\x99\x11\x38\x94\x40\x26\x2f\xa4\x81\x22\xf2\x6c\x9c\x08\xca\x1a\xb0\x06\xb6\xbe\x1a\x5b\x1f\x8c\xa8\xf0\x7e\xc6\x0a\x11\xe0\x51\x3e\x15\x2e\xd7\x6a\x98\xcf\x9a\x2a\x2f\xb4\x42\x13\x78\x81\x2e\xf4\x6b\xac\xe0\xa7\x9f\xee\x1d\x9d\x3c\xb9\x47\x2a\x1e\xa0\x0b\xfb\xfe\xf1\x2c\xa0\x5f\xe8\x4a\x6b\xaa\x54\x85\x08\xe8\xfb\xad\xae\x67\x58\x5b\xaf\x82\x75\xb3\xb8\x0d\x6f\xe1\x3c\x38\xd2\x6b\x3e\x67\x47\x27\x4f\x6e\x17\x3a\xc1\xd9\x4d\x99\xa7\x4e\x4d\x45\xc0\x5f\x70\xf6\x81\x92\x7f\xc1\xd9\x9a\xe0\xf7\x36\xe0\xfe\xd9\x09\xf8\xd6\x0b\xd0\xd4\x92\x64\xc0\xcb\xab\xab\x96\x9f\xff\x8f\x55\xe6\x1d\xee\xca\xb6\x21\x83\xf9\xfc\xd5\x9a\xc9\x4b\xeb\x40\x84\x80\x55\x1d\x40\x19\xb8\xda\xed\xf7\xbf\x9b\xff\x08\xd2\x32\x80\x59\x53\x41\xab\x06\xf0\x19\xf0\x3f\xe0\xc3\x64\x46\x91\xf0\xe5\x97\x30\x74\x28\x26\x0c\xe0\xce\x0b\xbf\xec\xd4\xd8\xba\x6a\x3f\xcd\x5f\x6d\xbe\x7a\xab\x53\xc2\x50\x29\x94\x46\x99\x31\x20\xcc\xbe\x7c\xb9\x74\x1a\xb8\x0e\xf0\x1d\xbc\x7a\xf5\x23\x45\xb7\x01\xaf\x11\x6b\xd8\xfd\x11\x50\x7b\x04\xbc\x54\x81\xbe\x94\x8a\x49\x6b\xf0\x1d\xde\x70\x58\xd9\xe9\x87\x81\x99\xac\x57\x68\x14\x86\x92\x0f\x73\x15\x70\x57\xc2\x9d\xe0\xbe\x03\x84\xec\xea\x0a\x8d\x9c\xcf\x29\xcb\x15\x0e\x45\x40\x92\x1e\x84\x32\xe8\xa0\x6e\xb4\x26\x2b\x39\x0c\xac\x9a\x48\xe5\x80\xd7\xd7\xcc\xac\x53\x23\x65\xf2\xbe\xb4\xc5\x04\xdd\x0d\xb8\xaf\x6e\xe6\xe9\x46\xfd\x37\xde\x9a\x65\xd8\xf7\x0f\xd1\xa9\x29\xca\xfe\x81\xad\x86\xca\xa0\x7c\x5a\x89\x11\x9e\x36\x5a\x9f\x47\xa9\x1d\x10\xd6\x20\xae\x0d\xe5\x9e\x5b\xa4\x41\xee\xac\x0d\x39\x5d\xe9\xf9\xc9\xe1\xc9\x00\x24\x6a\x0c\x18\x53\x71\x69\xb5\xb6\x17\xc4\x29\xa6\xda\x74\x67\xb2\xb2\x28\x29\x45\xab\x00\xca\xc3\x50\x4c\x50\x82\x32\xc1\x82\x6d\x1c\xfc\xef\x57\x50\xa4\x97\x67\xf1\x8c\x90\x12\x78\x09\xed\xb5\x99\x2a\xe1\x73\x18\x39\x5c\xb2\x4c\xa7\x06\x86\x22\x2f\x7d\x10\xc3\x04\x14\x06\xe0\x67\x3e\x60\x55\x04\x0d\x3e\xd8\xba\xe5\xc1\xa3\x37\x9b\xba\x1f\x54\x85\xee\x9d\x54\x1e\xdd\x54\x15\x78\x1b\xdd\xd2\x7e\x35\x29\x7d\xff\xb2\xf4\xa4\x6e\x2e\x71\x9a\x4b\xe5\x27\xb9\xf8\xb3\x71\x98\x2f\x4a\x4e\x2d\x5c\xd8\x65\x00\x58\x8c\x2d\xdc\xbb\x9b\x0c\xd6\xee\x08\xc4\x1e\x46\xae\xfe\xa3\xb1\x41\x00\xec\xc0\xce\x3d\x78\xf4\xe8\xfa\xea\xa4\x86\x6d\x4c\xb8\x79\x92\x01\x38\xf4\xc1\x3a\x2c\xac\x01\x7e\xb6\x61\x3f\x21\x8a\x38\xb5\x28\x92\x02\x2b\x6b\x6e\xa0\x88\x01\x64\x54\xb8\x24\x21\xc9\x65\x03\xc8\xde\xd8\xc6\x19\xa1\x65\xb6\x4d\x7b\x52\x79\xaa\x5a\x5c\xe3\x48\x14\x33\xee\x70\xa4\x7c\x70\xb3\x6c\x00\xc1\x35\xc8\x12\x9e\x56\x6d\x29\x5c\x58\x37\xe6\x66\x82\x1b\xbe\x2b\x15\x63\xad\x65\x62\xf0\x10\xc6\xdb\x5c\x16\xa1\xed\xfb\xc7\x56\x62\xcc\x5e\x8f\xa2\xa9\x0d\x51\x7d\xb9\x11\x45\x18\x0a\xb9\x09\x43\x0b\xaf\xde\xf4\x95\x2f\xbc\xda\xcd\x75\x63\x76\xe0\xed\xdb\x74\xbb\xdb\xdc\xba\x44\x7a\x43\x60\x72\xa8\xc4\x52\x34\x3a\xf8\xf7\x72\x28\x9d\xbb\xdd\x9d\x71\x97\xec\xd2\x03\x51\x14\x58\x53\x13\x05\xdf\x7f\xfb\xed\x37\x40\x25\x82\x62\x52\xc8\x4a\x79\x4f\x41\x48\xa9\xc7\x59\xad\xc9\x92\xd6\x81\xf4\xb1\x76\x84\xa2\xde\x8e\x07\xda\x0f\xdf\xc6\x32\xf2\x59\xed\x6c\xb0\x7b\x5b\x57\xd2\x87\x2f\xbe\xd8\x7e\x30\x67\x9f\xd5\xd6\x85\xb4\xd0\xeb\x3d\xd8\x9e\xb3\xcf\xae\x3b\x96\xfd\xd8\x51\x3d\x3d\x3b\xfa\x6d\xff\xd9\xb3\xd7\xfb\xcf\x9e\x9d\xfc\x46\xc9\x6c\x2b\x32\x01\x5e\x91\x53\x03\x02\xe7\xe9\xff\xe3\xa3\xdf\x68\xb1\xdb\xe6\x92\x58\xc3\x56\xfc\xcb\xdf\xc0\xfe\xc1\xc1\xd1\xe9\x73\xe0\x17\x6d\x8a\xef\xe4\x70\x2f\xa6\xd8\x62\xd6\xcf\x7c\xca\x7a\x79\xb7\xfb\x8e\x52\x40\x80\x21\xdb\xac\x63\xe6\x3c\x51\xc1\x7c\x7e\x77\x5d\xbd\x1b\x79\xd7\x5c\x3e\xae\x74\xbe\xb7\x94\x0f\xad\xa0\xff\xda\xb9\xad\x84\x52\x77\x7b\x7c\xf2\xfc\x68\x00\x4f\x0d\x94\x4d\x68\x1c\x6e\x43\x65\xa7\x98\x7a\x6e\x65\x4a\xeb\xaa\xb6\x5a\x36\xc1\x2b\x89\x60\x4b\x40\x33\x55\xce\x9a\x0a\x4d\x80\xa9\x70\x2a\x39\xa1\xc7\x3c\x06\xf8\xfa\x92\xe1\x65\x74\xe7\xf9\xfe\xf9\x8b\xb3\xa7\x7b\xf7\x96\xae\xf2\x6b\xb4\x44\x7b\x93\xb4\x0f\xf3\xf9\xbd\x78\x90\xc7\x81\xc0\x35\x26\x42\xb7\x35\x16\x70\xae\x8c\x0a\x10\x2c\x0c\xad\x0d\x3e\x38\x51\xc3\xe1\xf1\x39\x78\x0c\x4d\xdd\x65\x04\x3a\xc4\x79\xed\xd4\x54\x69\x1c\xa1\x04\xce\xa9\x7a\x73\x83\xe1\xc2\xba\x09\x50\x8d\x07\x3e\x85\x7c\x90\xa7\x8f\xd8\x2a\x77\xb7\x99\x57\x75\xe8\xb8\x45\x2d\x51\xd3\x9d\xa1\x54\x14\x03\xc1\x46\x95\x9d\x1a\x8d\x43\x8c\x34\xbc\x0c\xec\x46\xd0\x62\x28\x52\xf5\x4c\xc7\x53\xa6\xeb\xf4\xab\x84\x11\x84\x95\x60\xa1\x56\xc5\x04\x9a\x1a\x0c\x5e\x24\xb5\x3d\x06\x0a\x6c\x1f\xe3\x75\x8c\x20\xc7\xa9\xcf\x60\x71\x96\xba\xce\x9d\x1d\xcf\xe3\xc4\xf3\xd7\xc4\x92\xf5\x88\x41\x3c\x18\x63\x87\xaa\x8e\x9e\xf6\x29\x7e\x58\x5c\x58\x25\xa7\x4c\x56\x53\x4b\x88\xae\x2f\xf3\x1f\x7e\xe0\xa9\xfe\x73\x69\x7c\xdf\x8f\x97\x34\x97\xc6\x57\xc2\xff\x41\x1a\x8f\x5a\xfe\xa4\xf1\xb2\xb2\xa9\xbe\x24\xba\x56\xe0\x66\x8d\x3b\x9a\xae\x22\xb0\x1e\x5c\x44\xec\xd2\x2e\xc5\xad\xa1\xcc\x7e\x21\xc4\x88\xf0\x46\x03\x62\x67\xb7\x26\x28\xad\x82\x42\x0f\x23\x1b\xbb\xcc\x60\xc1\x89\x22\x76\x5a\x52\x11\x6a\xfb\x34\x5d\x95\x8b\xc3\xae\x31\x1e\x86\x58\x5a\x87\x24\x96\x5a\x92\x89\xb1\x17\xa6\x73\x61\x92\x84\x80\x46\x92\x13\x2e\x54\x18\x03\xc5\xd5\x0c\x7c\xec\x92\xd8\xc5\x58\x69\x8c\x21\xb7\x68\x1c\x81\xcb\xfb\xb0\xb7\x07\x59\x16\xc3\x4e\xda\xeb\xb6\x35\xc5\xd8\x3f\x02\xd2\x8f\x46\x65\x63\x08\x28\x49\x22\x63\xa9\x71\xe7\x85\xe0\xc1\x35\x3e\x82\xb6\xf5\x2d\xd9\x65\x84\x06\xa7\x22\x66\x38\x5a\xf1\x41\x14\x13\x10\x1e\xbc\xa5\x7e\xcf\x47\xb9\xab\xad\xb6\xf2\xa0\x85\x92\x94\x44\x60\x38\x63\xbd\x95\x18\x5f\xf4\xc5\xdb\xe9\xa4\xb6\x9e\x82\x61\xac\xa2\x83\xda\x7b\xdc\x42\x5c\x59\x87\xac\x47\xaa\x78\x28\x9d\xad\x56\x68\x6b\x67\x0b\xf4\x9e\x3c\x7a\xa1\xa8\xe3\x1e\xab\x3a\x41\x96\xf4\x67\x49\x0d\x8f\xe0\xc7\xe9\x6d\xa1\xa1\x99\xa0\x40\x10\x20\xc5\x0c\xac\xd1\x33\xba\x4d\x8d\xa9\x9a\x4a\x5b\x78\x96\x37\xde\xe5\xda\x16\x42\x47\x34\x8b\x3f\x3d\x16\xb2\xbd\x2c\x75\xce\x43\xe1\x51\x2b\x43\xa8\x80\xd3\xdd\xc3\x77\xd2\x7b\x5b\x86\x0b\xe1\xde\x9b\xbe\xd0\xa2\x12\xd3\x8e\x9a\xf5\x00\x4d\xf4\x39\x85\x45\x0a\xa7\x55\xaf\xb4\x41\xe5\xd9\x75\xdc\x35\xa6\x12\x7e\x02\x95\xf4\xb2\x8b\x39\x48\x72\x56\xbf\x56\xd6\x5c\xaf\x94\xba\x41\x13\x16\xdf\x97\xd8\xb5\x0a\x7c\x2a\x76\xe9\x12\x1f\xc7\x8d\xf5\xe0\x54\x19\x98\x34\x43\x4c\x96\x8b\x28\x6a\x3c\x42\xb4\x2c\x88\x5a\x71\xa2\x45\xc7\x3c\x05\xa2\x02\xee\x10\x32\xdf\xfb\x0a\x1e\xa4\xf5\x01\xdc\xef\x3f\xe8\xfd\xbe\x3b\x0e\xa1\xf6\x83\x3c\x5f\x9a\x13\x7b\x59\x4a\x6d\xed\x68\x94\x8a\x7b\x4e\xad\x96\xe9\x5f\x4b\xfc\x64\x8c\x17\xcf\x54\x3c\x2d\x7c\x52\x19\x94\xef\xe2\x9f\x4f\xcf\xd5\xcb\x4f\x60\x8e\x38\x6e\x46\x36\xed\xa0\xcb\xd8\xd5\x15\xa7\x8c\x6e\x10\xb6\xfa\x8f\x45\x31\x69\xea\xc7\xda\x0e\x8f\x29\x11\x67\xd9\x3b\x1f\xb9\x16\x35\x85\x72\xe0\x14\xdd\x6c\xed\x11\x80\x32\x5d\xa0\xf4\xbd\x28\x6c\xc3\x28\x25\xbe\x07\x9c\x95\xab\xed\x77\xfe\x80\x51\xdf\x45\x7a\x1c\x2a\xb7\xb7\xba\xd7\x9e\x4b\xf3\xfd\xd6\x12\xdd\x5f\x6e\x57\x8f\x42\x21\xd3\x9d\x3f\xb2\x63\x5d\x61\xf4\x77\x36\xad\xab\x82\x3e\x5d\xdf\x7a\xa7\x9e\xd2\x5e\x18\x6d\x85\x24\x23\x26\x27\x64\xab\x45\x78\xbd\xee\xfe\xce\x20\xd6\xde\xb5\xf8\x1b\xac\x2f\x6d\x22\x8e\x8f\xc5\xb5\xb3\x53\x25\xd1\xe5\x83\xfc\xb5\x14\x41\xe4\xaf\xa9\xdc\xb5\xd4\xcb\x00\x18\xe4\xb6\x09\x83\x3f\xe3\xd6\xbb\x6c\x46\x50\x4a\x97\x48\x9c\xf8\xb0\x85\xfb\x1e\x9d\xbc\x11\x01\xf3\x79\x4b\x24\xe3\x9b\x7a\xac\xbd\x7b\x24\xac\x05\x63\x5f\x0e\x5b\x02\x51\xc4\xbd\xce\x54\x77\x1b\xb4\x95\xdf\x11\x93\x0b\xbb\x30\x79\xd8\x4d\xa7\x7f\x15\xd3\x69\x24\xa0\x3b\x7f\x24\xa6\x57\x18\xfd\x9d\x98\x5e\x15\xf4\x0f\x61\x3a\x59\x39\xd6\x75\x23\x6a\x3f\xb6\xe1\x83\x30\x4d\x28\x1a\x2c\x3e\x2d\xb6\x96\xf3\xd5\x60\xf5\x5b\x42\x27\x47\x38\x7a\x7e\x70\x78\xf0\xfc\xd9\xeb\xfd\xd3\xa7\x7b\xd9\x37\xd9\x2d\xa0\x5d\x35\x0a\xd1\x10\x97\x58\xd0\x5b\x7d\x3b\xa0\xac\x44\xc2\x1a\x2e\x29\x6e\x38\x25\xcc\xd5\x5c\x4a\xc3\x45\x22\x88\x2d\xf7\x52\xc6\x6e\x97\x69\x42\x53\x42\xf3\x42\x37\x31\x46\xb3\xd6\x86\x3b\xf1\xdf\x5e\x57\x5f\x56\x56\x07\x0f\xbf\xf9\x7e\x67\x7b\x79\x69\x77\x23\xe1\xee\x3a\xe1\xc3\x8d\x84\x0f\x23\x61\xb6\x59\x25\x1e\xec\x04\x4d\x34\x0b\x2f\xad\xe3\xf1\xbd\xe8\x06\xa9\x90\x53\x74\x41\x79\xe4\x35\xa2\xe3\x8d\xd3\x1e\x36\x94\xc6\x28\x86\xb1\x6a\xba\x6e\xa5\xfc\xc1\x8d\xb5\xb5\x77\xec\x85\x3d\x57\x4a\xd2\xca\x60\x70\x83\xef\xfb\x20\x13\xe3\xb0\x93\xc5\xf2\x4c\xd3\xd3\x7c\xce\x58\x68\x0c\x4a\x2e\x64\x45\x8d\x38\x0d\x25\x70\xdd\xcc\xb4\x0f\x52\xbc\xd6\xc2\xa4\x91\x0f\x41\x68\x6f\xc1\x20\xca\x6b\xba\x7e\x6c\xd8\xfa\x53\xab\x9b\x0a\x3d\x10\x30\xd2\x63\xba\xec\xc6\xb8\xcb\xd2\x43\x7a\x22\x2d\x68\x78\xa3\x09\xaf\x7b\x52\xaf\x60\xe7\xdf\xdf\xed\x6c\x7a\x5a\xbf\x85\x3f\xe9\x91\x5e\x35\x63\x8b\xe0\x67\x5e\xdb\x11\x78\x45\x33\xc1\x05\xb6\x63\x3a\x20\xf5\x0d\x61\x4c\x24\x61\xec\x6c\x33\x1a\x43\xf7\x30\xba\xd4\xc7\xb6\xaf\xa3\x1d\x97\x8d\x9d\xae\xad\xd7\xb6\x59\x0f\x8c\x0d\x38\x00\x11\x6c\xa5\x0a\x7e\x6d\xb1\x38\x9b\x16\x4e\xf8\x31\x68\x6b\x6b\x0f\x8d\x09\x4a\x77\x3f\x81\x2a\x0f\x4d\xbd\xde\x95\x6f\xe4\xb2\x10\xf6\x29\x7e\x36\xf4\xc5\x18\x65\x13\x0d\xb6\x1c\x95\x0e\x87\xd6\xc6\x77\x9b\xc2\x56\x75\xfc\x95\x60\xd3\x2f\x43\x19\xf3\xe3\x26\x50\x61\xa1\x14\x96\xce\x7c\xfd\x90\x5d\x5d\x51\x8a\x9c\xcf\xd7\xe6\x82\x3b\xef\xb3\x78\x9c\xed\x7e\x7a\xf9\x7f\x00\x00\x00\xff\xff\x64\x5c\xc3\xc8\x71\x1e\x00\x00")

func masterStartupShBytes() ([]byte, error) {
	return bindataRead(
		_masterStartupSh,
		"master-startup.sh",
	)
}

func masterStartupSh() (*asset, error) {
	bytes, err := masterStartupShBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "master-startup.sh", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _nodeStartupSh = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xbc\x58\x51\x6f\xdb\x38\x12\x7e\xe7\xaf\x98\xb5\x17\xcd\x2d\xb6\x92\xd2\x03\x7a\x87\x4d\xb7\x05\x7a\x6d\x17\xe8\x2d\xb6\x09\x92\xbd\xbb\x87\xa2\x0f\xb4\x38\x92\xb8\xa6\x48\x95\x33\xb4\xe3\x75\xfd\xdf\x0f\x23\xc9\x89\x1d\x27\x6e\x8b\x1e\xee\x2d\x16\xc9\x99\x6f\x66\xbe\xf9\x38\xcc\xf4\xbb\x62\x66\x7d\x31\xd3\xd4\x40\x86\xd7\x4a\xad\xd7\x60\x2b\xc8\x5f\x05\x5f\xd9\x3a\xbf\xc2\x32\x45\xcb\xab\x0b\xcd\x65\x73\xa1\xcb\xb9\xae\x91\x60\xb3\x51\x2e\xd4\x35\x46\xc8\x18\x7c\x30\x98\x11\xeb\xc8\xa9\xcb\xa9\x81\x89\xf5\xc4\xda\x39\xeb\x6b\x88\x68\xa0\xd1\x0c\xa5\xf1\x50\xf6\x16\x53\xd4\x6c\x83\x87\xe0\xe1\xfb\xbf\x34\x81\xd8\xeb\x16\x7f\x98\xa8\x52\x33\xbc\x28\x16\x3a\x16\xce\xce\x8a\x55\x6a\x8b\xd2\x59\xf4\x9c\x95\x18\x39\xef\xb0\x85\x9f\x7f\x3e\x79\x73\xfe\xcb\x89\x00\x7c\x85\x91\x5f\xd2\x3f\x56\x8c\x74\x83\x54\xbe\xd9\xca\x96\x9a\x91\xf2\x11\xe9\x25\x76\x81\x2c\x87\xb8\xea\x97\xe1\x13\x5c\x71\x14\x5c\x9b\x8d\x7a\x73\xfe\xcb\xc3\x4e\xe7\xb8\xba\xeb\xf3\x22\xda\x85\x66\xfc\x15\x57\x5f\xe9\xf9\x57\x5c\x1d\x38\xfe\xc2\xf4\xbd\xbc\x3c\x07\x1a\x2b\x00\xa9\x33\xe2\x01\xde\xaf\xd7\xa3\x35\xfa\x67\xb0\xfe\x33\xa5\x9a\x3c\x86\x09\x6c\x36\x1f\x0e\x12\x5e\x85\x08\x9a\x19\xdb\x8e\xc1\x7a\x58\x3f\xc9\xf3\xa7\x9b\x67\x60\x82\x02\x58\xa5\x16\x46\x18\x90\xad\x20\xfb\x08\x5f\xe7\xb3\x77\x09\x8f\x1e\xc1\x2c\xa2\x9e\x2b\x80\x23\xe1\xbe\xdf\x82\xf8\x7e\x3d\xfe\xb5\xf9\x70\x7f\xe0\x23\xa2\x81\x3f\x95\xb6\x0e\xcd\x44\x81\xb0\xf5\xfd\xfb\x9d\xd3\x90\x39\x86\xa7\xf0\xe1\xc3\x33\xe0\x06\x3d\x90\x43\xec\xe0\xc9\x33\x40\x47\x08\x78\x6d\x59\x7e\x54\x56\x99\xe0\xf1\x68\x25\x22\xb6\x61\xf1\x75\x34\x96\xcc\x95\x0e\xb5\x07\xed\x9c\x8a\x2d\x64\xb1\x82\xa3\xb4\x3e\x42\x3f\xb5\x5e\xa3\x37\x9b\x8d\x52\x53\x28\x23\x6a\x46\xf1\xce\xda\x7a\x8c\xd0\x25\xe7\x24\x47\x11\x59\xb5\x73\x63\x23\x64\xdd\xad\xb1\x10\x6d\x6d\x7d\x91\x9b\x50\xce\x31\xde\x21\xfa\xfe\x62\x31\x44\x94\xff\x41\xc1\xef\x12\x3e\x7f\x8d\xd1\x2e\xd0\xe4\xaf\x42\x3b\xb3\x1e\xcd\xdb\x56\xd7\x78\x91\x9c\xbb\xea\xbd\x6e\x49\x70\x40\x6e\xe7\x21\xa3\x87\xa0\x40\x11\x43\xe0\x42\x42\xfa\xfd\xfc\xf5\xf9\x19\x18\x74\xc8\x28\xa5\x82\x2a\x38\x17\x96\x62\xa9\x8e\x21\x75\x43\xcc\x92\x65\x5d\x31\x46\xb0\x0c\x96\x60\xa6\xe7\x68\xc0\x7a\x0e\x10\x52\x84\x7f\xff\x06\x56\x70\x91\xea\xcf\x68\x63\x20\xab\x60\x0c\x5b\xd9\x0a\xbe\x83\x3a\xe2\x4e\x66\xb6\x30\x90\xcb\xa2\x22\xd6\xb3\x81\x26\x0a\x80\x56\xc4\xd8\x96\xec\x80\x38\x74\xa3\x8d\xac\xaf\x66\xea\x72\xb6\x2d\xc6\xcf\xee\x22\x8c\x0b\x5b\xe2\x43\xfb\x76\xd6\xdb\x79\x45\xf9\x75\x45\x02\xb7\x30\xb8\x28\x8c\xa5\x79\xa1\xff\x4c\x11\x8b\x88\x14\x52\x2c\x31\xeb\x74\xe4\x27\x0a\x00\xcb\x26\xc0\xc9\xf1\x6d\x70\x10\x23\x88\x79\xa8\x63\xf7\x31\x05\xd6\x00\xa7\x70\x7a\x02\x2f\x5e\xdc\x86\x2e\x30\x42\xf2\x7c\xf7\xa4\x02\x88\x48\x1c\x22\x96\xc1\x43\x76\x79\xb0\xbe\x5e\x67\xd2\x77\xf8\x11\xf2\xcb\xe0\x50\x44\xab\x8a\x5a\xba\x5e\x01\x0c\x64\x13\x27\x23\xc1\x8c\xc6\x36\xf8\x3b\x04\x53\x00\x13\x17\xea\xcc\x08\xc9\xe2\xe4\x0c\x26\x7f\x84\x14\xbd\x76\x66\xf2\x58\xd6\x8c\x25\x3d\x73\x98\x39\xac\x75\xb9\xca\x22\xd6\x96\x38\xae\x26\x67\xc0\x31\xa1\x1a\xa8\x26\x38\xd0\x9b\xc1\xef\x6e\xc6\x75\xe4\xc3\x94\xdf\xbf\xe1\x4e\x85\x2b\xab\xd4\x98\xbf\xbe\xc5\xa4\x13\x46\xb5\xeb\x1b\x80\xf2\x77\xc1\x60\xaf\x6f\x2f\xfa\x82\x78\xd9\xf5\xe8\xa8\x90\x88\x21\xa1\xf5\xa1\xad\xab\x61\x17\x6c\x36\xc7\x15\xf9\x38\xa2\x5b\x2b\xdf\x22\xba\x5f\xec\xe3\x6b\xb5\xf7\x6f\xa7\x0f\x89\xef\x54\x4d\xe1\xdd\xf9\xef\x6f\xce\xe0\xad\x87\x2a\x71\x8a\xf8\x18\xda\xb0\x10\x3d\xd0\x92\x85\x2a\xc4\x76\x54\xda\xc4\x64\x0d\x42\xa8\x00\xfd\xc2\xc6\xe0\x5b\xf4\x0c\x0b\x1d\xad\xf0\x84\xd4\x54\x11\x32\xfc\x78\xad\xf0\xba\x0b\x91\xe1\xea\xe5\xd5\xbf\x2e\xdf\x3e\x3f\xd9\x09\xe5\x3f\x21\xce\x31\x8e\x91\x0c\xeb\xb0\xd9\x9c\xf4\x07\xb3\x6b\xd1\xa4\x98\x7c\x2f\x45\x63\xb2\x20\xcb\xac\xb7\x0c\x1c\x60\x16\x02\x13\x47\xdd\xc1\xeb\x77\x57\x40\xc8\xa9\xdb\xf2\x44\x0e\x65\x59\x17\xed\xc2\x3a\xac\xd1\x40\x96\x89\xf2\x67\x1e\x79\x19\xe2\x1c\xe4\x7e\x80\x6c\x01\xc5\x59\x31\xfc\x89\x23\xb8\xe3\x69\xde\xc7\xb0\xb5\xd6\xa3\x44\x27\x31\x43\x65\x1d\x92\x80\x13\xc8\xd1\xd6\x0d\xf7\x17\x04\x5e\xb3\xba\xd3\xbf\xc8\xe5\xa0\xbc\xc3\xf1\x81\xff\x5b\x7c\xad\xf6\x5a\x98\xc2\x01\x3a\x5b\xce\x21\x75\xe0\x71\x39\xc0\x26\x64\x96\x6b\x5f\x18\x29\x5e\x4c\x33\xdc\x51\xaa\x1f\x1b\x6f\x3b\x6a\x6b\xf3\xdd\x60\xf3\xb7\xc1\xa4\x9a\x8a\x81\xfe\x60\x2f\x08\xa2\x58\x6e\x91\xcb\x8d\xa3\xfa\x0f\xfb\xdb\x45\xdc\x3a\x19\x25\x30\xe6\xa6\xf8\xe9\xa7\x6c\xb8\x3b\x32\xe3\x29\xa7\x66\x07\xb9\xf1\xd4\x6a\xfa\x28\x88\xeb\xd1\xbe\x20\xde\x05\x3b\x08\xd0\xb0\x6f\x74\x78\x3f\xe2\xed\x9e\xad\x4e\xa8\x29\x2c\x7b\xee\xca\xaa\x74\xad\x74\x0f\x2c\xb5\xae\x85\x6f\xda\x9b\x9b\xbc\x25\xb6\xce\xb2\x45\x82\x3a\xf4\xf3\x09\x07\x88\xba\xec\x6f\x69\x63\x85\xb5\xb9\x9a\x4a\x8f\x6c\x0f\xc7\xe4\x09\x66\x58\x85\x88\xe2\x56\xae\xb3\xb9\x0f\x4b\xbf\x2d\xe1\xe0\x09\x7b\x49\x4b\x1d\x2c\x2d\x37\x20\x7d\xb5\x02\xea\x6f\x58\xb5\x6c\xac\xc3\xbe\xe5\x6e\x86\x0e\xc8\xcc\x0f\xf0\xfc\x39\x4c\x26\x7d\xdb\x99\x70\x3b\xf0\x0c\x3d\xf6\x7f\x21\xe9\x37\xb3\x32\x79\x21\xca\xe0\x51\xa9\x61\xe4\xcb\x4a\x9d\x71\x4c\xd4\x93\x76\xac\xad\xe4\xa5\x46\x8f\x0b\xdd\xeb\x9b\x7c\x21\xd6\xe5\x1c\x34\x01\x05\x99\x15\xa8\xf7\xbb\x3f\xa6\x59\x02\xa7\xad\x11\x11\x81\xd9\x4a\x4d\xf7\x7a\xfc\x66\xa6\x7a\x3c\x9c\x74\x81\xa4\x19\x1a\xdb\x17\x68\x8c\xe3\x81\xcd\x6d\x88\xa8\xa6\x02\x85\xa0\x8a\xa1\xdd\xdb\xdb\xc5\x50\x22\x91\x54\x74\x69\x65\x5a\x6b\x6c\x37\x50\x56\xf0\xab\x01\x06\x21\x50\x13\x92\x33\x7d\x85\x82\x2f\x11\x34\x18\xbd\x82\xe0\xdd\x4a\xa2\xe9\x7a\x30\x28\xf2\x4c\xaa\x48\x14\x0b\x17\x4a\xed\x7a\x36\xeb\x3f\x09\x4b\x33\x06\x2b\x53\xd7\x4c\x13\x3a\xeb\x85\x15\x70\xf1\xe4\xf5\x67\xf7\x53\xa8\x78\xa9\xe3\x17\xef\x2f\x9d\x6e\xf5\x62\xbb\x5b\x4d\x01\x7d\x5f\x73\x69\x8b\xa1\x9d\xf6\xab\x32\x36\x15\xa9\xdb\xbe\x4b\xbe\xd5\x34\x87\xd6\x90\xd9\xf6\x1c\x0c\x7e\xf6\x7f\xb6\xc1\xdf\x7e\xa9\x5c\x42\xcf\x37\xbf\x77\xcc\x8d\x00\xfe\x57\xe6\x86\x20\xbe\xcd\x9a\x3a\x36\x1f\x71\xf2\x68\x32\x6d\x5a\xa1\x87\xb4\x0a\x84\x0e\x3d\x35\xb6\xe2\x4c\xc8\x15\x83\xcb\x3a\xa7\x3d\x0e\xc3\x8d\x5c\x9b\x9f\x39\x25\xaa\xb1\x3b\x09\x89\x80\x21\x68\x47\x01\x3c\xa2\xb9\xdd\x99\xf7\x85\xcd\x17\xc1\xa5\x16\x09\xe4\xcd\x30\x3c\x2b\xcc\x56\x94\x64\x60\x1c\x86\xc5\x52\xa4\x48\xf4\x6a\xfb\xb8\x68\xe1\xf4\xef\x4f\x4f\xef\x7b\x64\x3c\x60\x5f\x70\x0c\x43\x5c\xff\x84\xa2\x15\xb9\x50\x03\x59\x61\xf8\x12\xc7\x4b\x07\x70\x81\x71\xc5\x8d\x6c\xe1\x26\x86\x54\x37\xb0\x9d\x03\x77\xaa\x32\x0e\x83\x5b\x2b\xf7\xd6\x2d\x74\x07\xcb\x6a\x0a\x3e\x30\x9e\x81\xe6\xd0\xda\x32\xdb\xcf\x19\x94\x51\x53\x03\x2e\x84\x8e\x20\x79\xb6\x0e\x5a\x4d\xfd\x4b\x83\x20\x75\x87\x1c\xbb\xd7\xca\x8d\xb3\x6f\xff\xc7\x09\x95\x0d\x9a\xd4\xa7\x6b\xe7\x51\x09\x11\x65\xf6\x10\xe1\x28\x43\xdb\xf5\xaf\xa5\xfb\xde\xc7\x13\x45\x4d\x62\x23\xd7\x49\x96\x8d\x67\x7e\xfc\xab\xbc\x21\x1d\xe1\x66\x73\xc0\xf1\xa3\xd1\xc0\xa7\x4f\xc3\x9c\xbd\x7d\x82\xfe\x37\x00\x00\xff\xff\xe1\x5c\xe6\xbf\x28\x12\x00\x00")

func nodeStartupShBytes() ([]byte, error) {
	return bindataRead(
		_nodeStartupSh,
		"node-startup.sh",
	)
}

func nodeStartupSh() (*asset, error) {
	bytes, err := nodeStartupShBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "node-startup.sh", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	canonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[canonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
func MustAsset(name string) []byte {
	a, err := Asset(name)
	if err != nil {
		panic("asset: Asset(" + name + "): " + err.Error())
	}

	return a
}

// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func AssetInfo(name string) (os.FileInfo, error) {
	canonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[canonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, fmt.Errorf("AssetInfo %s not found", name)
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() (*asset, error){
	"master-startup.sh": masterStartupSh,
	"node-startup.sh":   nodeStartupSh,
}

// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("nonexistent") would return an error
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		canonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(canonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for childName := range node.Children {
		rv = append(rv, childName)
	}
	return rv, nil
}

type bintree struct {
	Func     func() (*asset, error)
	Children map[string]*bintree
}

var _bintree = &bintree{nil, map[string]*bintree{
	"master-startup.sh": {masterStartupSh, map[string]*bintree{}},
	"node-startup.sh":   {nodeStartupSh, map[string]*bintree{}},
}}

// RestoreAsset restores an asset under the given directory
func RestoreAsset(dir, name string) error {
	data, err := Asset(name)
	if err != nil {
		return err
	}
	info, err := AssetInfo(name)
	if err != nil {
		return err
	}
	err = os.MkdirAll(_filePath(dir, filepath.Dir(name)), os.FileMode(0755))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(_filePath(dir, name), data, info.Mode())
	if err != nil {
		return err
	}
	err = os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
	if err != nil {
		return err
	}
	return nil
}

// RestoreAssets restores an asset under the given directory recursively
func RestoreAssets(dir, name string) error {
	children, err := AssetDir(name)
	// File
	if err != nil {
		return RestoreAsset(dir, name)
	}
	// Dir
	for _, child := range children {
		err = RestoreAssets(dir, filepath.Join(name, child))
		if err != nil {
			return err
		}
	}
	return nil
}

func _filePath(dir, name string) string {
	canonicalName := strings.Replace(name, "\\", "/", -1)
	return filepath.Join(append([]string{dir}, strings.Split(canonicalName, "/")...)...)
}
