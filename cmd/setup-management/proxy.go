package main

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/test/management"
)

func main() {
	logger := logrus.New()
	logger.Formatter = &logrus.TextFormatter{FullTimestamp: true}
	logger.SetLevel(logrus.DebugLevel)
	log := logrus.NewEntry(logger)

	if err := management.Run(context.Background(), log); err != nil {
		panic(err)
	}
}
