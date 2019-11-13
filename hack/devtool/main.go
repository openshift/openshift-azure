package main

import (
	"github.com/spf13/cobra"

	"github.com/openshift/openshift-azure/hack/devtool/newdev"
	"github.com/openshift/openshift-azure/hack/devtool/version"
)

var gitCommit = "unknown"

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	rootCmd := &cobra.Command{
		Use:  "./devtool [component]",
		Long: "Azure Red Hat OpenShift dispatcher",
	}
	rootCmd.AddCommand(version.NewCommand())
	rootCmd.AddCommand(newdev.NewCommand())

	return rootCmd.Execute()
}
