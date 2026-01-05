package cluster

import (
	readonlyingress "AtlHyper/atlhyper_agent/internal/readonly/ingress"
	modelingress "AtlHyper/model/k8s"
	"context"
)

func IngressList(ctx context.Context) ([]modelingress.Ingress, error) {
	m, err := readonlyingress.ListIngresses(ctx)
	if err != nil {
		return nil, err
	}
	return m, nil
}