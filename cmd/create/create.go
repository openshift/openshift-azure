package main

import (
	"github.com/jim-minter/azure-helm/pkg/osa"
)

func main() {
	if err := osa.Create(); err != nil {
		panic(err)
	}
}
