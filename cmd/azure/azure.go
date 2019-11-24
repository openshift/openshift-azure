package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/openshift/openshift-azure/pkg/entrypoint/admissioncontroller"
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
		os.Exit(1)
	}
}

func run() error {
	rootCmd := &cobra.Command{
		Use:  "./azure [component]",
		Long: "Azure Red Hat OpenShift dispatcher",
	}
	rootCmd.SilenceUsage = true
	rootCmd.PersistentFlags().StringP("loglevel", "l", "Debug", "Valid values are [Debug, Info, Warning, Error]")
	rootCmd.Printf("gitCommit %s\n", gitCommit)

	rootCmd.AddCommand(azurecontrollers.NewCommand())
	rootCmd.AddCommand(canary.NewCommand())
	rootCmd.AddCommand(etcdbackup.NewCommand())
	rootCmd.AddCommand(metricsbridge.NewCommand())
	rootCmd.AddCommand(startup.NewCommand())
	rootCmd.AddCommand(sync.NewCommand())
	rootCmd.AddCommand(tlsproxy.NewCommand())
	rootCmd.AddCommand(admissioncontroller.NewCommand())

	return rootCmd.Execute()
}
