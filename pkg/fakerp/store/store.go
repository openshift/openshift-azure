package store

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/sirupsen/logrus"
)

type Store interface {
	Put(key string, b []byte) error
	Get(key string) ([]byte, error)
	Delete(key string) error
}

type Storage struct {
	mutex   sync.Mutex
	mutexes map[string]*sync.Mutex
	dir     string
	log     *logrus.Entry
}

var _ Store = &Storage{}

func New(log *logrus.Entry, dir string) (*Storage, error) {
	dir = filepath.Clean(dir)

	s := &Storage{
		dir:     dir,
		log:     log,
		mutexes: make(map[string]*sync.Mutex),
	}

	// if the database already exists, just use it
	if _, err := os.Stat(dir); err == nil {
		s.log.Debugf("Using '%s' (database already exists)", dir)
		return s, nil
	}

	// if the database doesn't exist create it
	s.log.Debugf("Creating database at '%s'", dir)
	return s, os.MkdirAll(dir, 0755)
}

func (s *Storage) Put(key string, b []byte) error {
	if key == "" {
		return fmt.Errorf("missing key - unable to save")
	}

	mutex := s.getMutex(key)
	mutex.Lock()
	defer mutex.Unlock()

	fnlPath := filepath.Join(s.dir, key)
	tmpPath := fnlPath + ".tmp"

	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return err
	}

	if err := ioutil.WriteFile(tmpPath, b, 0644); err != nil {
		return err
	}

	return os.Rename(tmpPath, fnlPath)
}

// Get a record from the database
func (s *Storage) Get(key string) ([]byte, error) {
	if key == "" {
		return nil, fmt.Errorf("missing key - unable to read")
	}
	record := filepath.Join(s.dir, key)

	if _, err := stat(record); err != nil {
		return nil, err
	}

	return ioutil.ReadFile(record)
}

func (s *Storage) Delete(key string) error {
	mutex := s.getMutex(key)
	mutex.Lock()
	defer mutex.Unlock()

	record := filepath.Join(s.dir, key)

	switch fi, err := stat(record); {
	case fi == nil, err != nil:
		return fmt.Errorf("unable to find file or directory named %v", record)
	case fi.Mode().IsDir():
		return os.RemoveAll(record)
	case fi.Mode().IsRegular():
		return os.RemoveAll(record)
	}

	return nil
}

func stat(path string) (fi os.FileInfo, err error) {
	// check for dir, if path isn't a directory check to see if it's a file
	if fi, err = os.Stat(path); os.IsNotExist(err) {
		fi, err = os.Stat(path)
	}
	return
}

func (s *Storage) getMutex(collection string) *sync.Mutex {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	m, ok := s.mutexes[collection]
	// if the mutex doesn't exist make it
	if !ok {
		m = &sync.Mutex{}
		s.mutexes[collection] = m
	}
	return m
}
