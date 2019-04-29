package store

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
)

type Store interface {
	Put(key string, cs *api.OpenShiftManagedCluster) error
	Get(key string) (*api.OpenShiftManagedCluster, error)
	Delete(key string) error
}

type Storage struct {
	dir string
	log *logrus.Entry
}

var _ Store = &Storage{}

func New(log *logrus.Entry, dir string) *Storage {
	dir = filepath.Clean(dir)
	return &Storage{
		dir: dir,
		log: log,
	}
}

func (s *Storage) Put(key string, cs *api.OpenShiftManagedCluster) error {
	if key == "" {
		return fmt.Errorf("missing key - unable to save")
	}

	fnlPath := filepath.Join(s.dir, key)
	tmpPath := fnlPath + ".tmp"

	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return err
	}

	b, err := yaml.Marshal(cs)
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(tmpPath, b, 0644); err != nil {
		return err
	}

	return os.Rename(tmpPath, fnlPath)
}

// Get a record from the database
func (s *Storage) Get(key string) (*api.OpenShiftManagedCluster, error) {
	if key == "" {
		return nil, fmt.Errorf("missing key - unable to read")
	}
	record := filepath.Join(s.dir, key)

	b, err := ioutil.ReadFile(record)
	if err != nil {
		return nil, err
	}

	var cs *api.OpenShiftManagedCluster
	err = yaml.Unmarshal(b, &cs)
	return cs, err
}

func (s *Storage) Delete(key string) error {
	record := filepath.Join(s.dir, key)
	return os.RemoveAll(record)
}
