package azureclient

import (
	"context"
	"fmt"
)

type ServicePrincipalsClientAddons interface {
	ObjectIDforApplicationID(ctx context.Context, appID string) (string, error)
}

func (spc *servicePrincipalsClient) ObjectIDforApplicationID(ctx context.Context, appID string) (string, error) {
	sp, err := spc.List(ctx, fmt.Sprintf("appID eq '%s'", appID))
	if err != nil {
		return "", err
	}

	if len(sp.Values()) != 1 {
		return "", fmt.Errorf("graph query returned %d values", len(sp.Values()))
	}

	return *sp.Values()[0].ObjectID, nil
}
