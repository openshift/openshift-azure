package graphrbac

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
)

type GroupsClientAddons interface {
	GetGroupMembers(ctx context.Context, objectID string) ([]graphrbac.BasicDirectoryObject, error)
}

func (c *groupsClient) GetGroupMembers(ctx context.Context, objectID string) ([]graphrbac.BasicDirectoryObject, error) {
	pages, err := c.GroupsClient.GetGroupMembers(ctx, objectID)
	if err != nil {
		return nil, err
	}

	var objects []graphrbac.BasicDirectoryObject
	for pages.NotDone() {
		objects = append(objects, pages.Values()...)

		err = pages.Next()
		if err != nil {
			return nil, err
		}
	}

	return objects, nil

}
