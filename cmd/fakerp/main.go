package main

import (
	"os"

	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/fakerp"
	"github.com/openshift/openshift-azure/pkg/fakerp/shared"
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	logrus.SetReportCaller(true)
	log := logrus.NewEntry(logrus.StandardLogger())

	// TODO: Validate all required env variables are in place

	log.Info("starting the fake resource provider")
	s := fakerp.NewServer(log, os.Getenv("RESOURCEGROUP"), shared.LocalHttpAddr)
	s.Run()
}
