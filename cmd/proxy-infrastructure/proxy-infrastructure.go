package main

import (
	"context"

	"github.com/sirupsen/logrus"

	proxyinfrastructure "github.com/openshift/openshift-azure/hack/proxy-infrastructure"
)

// Deploy management proxy infrastructure

func main() {
	logger := logrus.New()
	logger.Formatter = &logrus.TextFormatter{FullTimestamp: true}
	logger.SetLevel(logrus.DebugLevel)
	log := logrus.NewEntry(logger)

	if err := proxyinfrastructure.Run(context.Background(), log); err != nil {
		panic(err)
	}
}
