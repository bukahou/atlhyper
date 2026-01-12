// NeuroController/internal/interfaces/cluster/service.go
package cluster

import (
	readonlyservice "AtlHyper/atlhyper_agent/internal/readonly/service"
	modelservice "AtlHyper/model/k8s"
	"context"
)

func ServiceList(ctx context.Context) ([]modelservice.Service, error) {
	m, err := readonlyservice.ListServices(ctx)
	if err != nil {
		return nil, err
	}
	return m, nil
}