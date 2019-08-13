package api

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/compute/mgmt/compute"
)

type ARMInterface interface {
	Generate(ctx context.Context, backupBlob string, isUpdate bool, suffix string) (map[string]interface{}, error)
	Vmss(app *AgentPoolProfile, backupBlob, suffix string) (*compute.VirtualMachineScaleSet, error)
	Hash(app *AgentPoolProfile) ([]byte, error)
}
