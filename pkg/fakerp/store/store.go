package store

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
)

type Store interface {
	Put(cs *api.OpenShiftManagedCluster) error
	Get() (*api.OpenShiftManagedCluster, error)
	Delete() error
}

type storage struct {
	dir string
	log *logrus.Entry
}

var _ Store = &storage{}

func New(log *logrus.Entry, dir string) Store {
	return &storage{
		dir: dir,
		log: log,
	}
}

func (s *storage) Put(cs *api.OpenShiftManagedCluster) error {
	b, err := yaml.Marshal(cs)
	if err != nil {
		return err
	}

	err = os.MkdirAll(s.dir, 0750)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filepath.Join(s.dir, "containerservice.yaml"), b, 0666)
}

// TODO: in future, can pass the resourcegroup/resource name here
func (s *storage) Get() (cs *api.OpenShiftManagedCluster, err error) {
	b, err := ioutil.ReadFile(filepath.Join(s.dir, "containerservice.yaml"))
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(b, &cs)
	return
}

// TODO: in future, can pass the resourcegroup/resource name here
func (s *storage) Delete() error {
	return os.RemoveAll(s.dir)
}
