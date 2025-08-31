package cluster

import (
	readonlyservice "AtlHyper/atlhyper_agent/internal/readonly/namespace"
	model "AtlHyper/model/namespace"
	"context"
)

func NamespaceList(ctx context.Context) ([]model.Namespace, error) {
	m, err := readonlyservice.ListNamespaces(ctx)
	if err != nil {
		return nil, err
	}
	return m, nil
}