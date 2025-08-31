package cluster

import (
	readonlynode "AtlHyper/atlhyper_agent/internal/readonly/node"
	modelnode "AtlHyper/model/node"
	"context"
)

func NodeList(ctx context.Context) ([]modelnode.Node, error) {
	m, err := readonlynode.ListNodes(ctx)
	if err != nil {
		return nil, err
	}
	return m, nil
}