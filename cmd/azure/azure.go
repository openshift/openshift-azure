package main

import (
	"github.com/spf13/cobra"

	azurecontrollers "github.com/openshift/openshift-azure/pkg/entrypoint/azure-controllers"
	"github.com/openshift/openshift-azure/pkg/entrypoint/canary"
	"github.com/openshift/openshift-azure/pkg/entrypoint/etcdbackup"
	"github.com/openshift/openshift-azure/pkg/entrypoint/metricsbridge"
	"github.com/openshift/openshift-azure/pkg/entrypoint/startup"
	"github.com/openshift/openshift-azure/pkg/entrypoint/sync"
	"github.com/openshift/openshift-azure/pkg/entrypoint/tlsproxy"
)

var gitCommit = "unknown"

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	rootCmd := &cobra.Command{
		Use:  "./azure [component]",
		Long: "Azure Red Hat OpenShift dispatcher",
	}
	rootCmd.PersistentFlags().StringP("loglevel", "l", "Debug", "Valid values are [Debug, Info, Warning, Error]")
	rootCmd.Printf("gitCommit %s\n", gitCommit)

	rootCmd.AddCommand(azurecontrollers.NewCommand())
	rootCmd.AddCommand(canary.NewCommand())
	rootCmd.AddCommand(etcdbackup.NewCommand())
	rootCmd.AddCommand(metricsbridge.NewCommand())
	rootCmd.AddCommand(startup.NewCommand())
	rootCmd.AddCommand(sync.NewCommand())
	rootCmd.AddCommand(tlsproxy.NewCommand())

	return rootCmd.Execute()
}
