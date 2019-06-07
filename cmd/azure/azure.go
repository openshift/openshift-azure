package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/openshift/installer/pkg/ipnet"
	"github.com/openshift/installer/pkg/types"
	"github.com/openshift/installer/pkg/types/azure"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift/openshift-azure/pkg/util/random"
)

var (
	gitCommit  = "unknown"
	baseDomain = "osadev.cloud"
)

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
	rootCmd.PersistentFlags().StringP("name", "n", "demo-cluster", "Cluster name value")
	rootCmd.Printf("gitCommit %s\n", gitCommit)
	rootCmd.Execute()

	// TODO: move to util/secrets
	fqdn, err := random.FQDN(baseDomain, 5)
	if err != nil {
		return err
	}
	name, err := rootCmd.Flags().GetString("name")
	if err != nil {
		return err
	}

	file, err := os.Open("secrets/pull-secret.txt")
	if err != nil {
		return err
	}
	defer file.Close()
	pullSecret, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	cfg := types.InstallConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: types.InstallConfigVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		BaseDomain: fqdn,
		Compute: []types.MachinePool{
			{
				Name:           "worker",
				Replicas:       to.Int64Ptr(1),
				Hyperthreading: types.HyperthreadingEnabled,
			},
		},
		Networking: &types.Networking{
			MachineCIDR:    ipnet.MustParseCIDR("10.0.0.0/16"),
			NetworkType:    "OpenShiftSDN",
			ServiceNetwork: []ipnet.IPNet{*ipnet.MustParseCIDR("172.30.0.0/16")},
			ClusterNetwork: []types.ClusterNetworkEntry{
				{
					CIDR:       *ipnet.MustParseCIDR("10.128.0.0/14"),
					HostPrefix: 23,
				},
			},
		},
		ControlPlane: &types.MachinePool{
			Name:           "master",
			Replicas:       to.Int64Ptr(3),
			Hyperthreading: types.HyperthreadingEnabled,
		},
		Platform: types.Platform{
			Azure: &azure.Platform{},
		},
		PullSecret: string(pullSecret),
	}

	fmt.Println(cfg)

	// TODO: Start consuming assets. Assets need to be file system agnostic
	// Like: https://github.com/openshift/installer/blob/master/pkg/asset/installconfig/ssh.go#L49

	return nil
}
