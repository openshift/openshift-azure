package main

import (
	"github.com/jim-minter/azure-helm/pkg/tunnel"
)

func main() {
	if err := tunnel.Run(); err != nil {
		panic(err)
	}
}
