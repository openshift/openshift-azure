package main

import (
	"fmt"

	installer "github.com/openshift/installer/pkg/types"
	"github.com/spf13/cobra"
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

	test := installer.InstallConfig{}
	fmt.Println(test)

	return nil
}
