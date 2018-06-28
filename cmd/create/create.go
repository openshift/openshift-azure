package main

import (
	"github.com/jim-minter/azure-helm/pkg/osa"
)

func main() {
	rp := &osa.RP{}
	if err := rp.Run(); err != nil {
		panic(err)
	}
}
