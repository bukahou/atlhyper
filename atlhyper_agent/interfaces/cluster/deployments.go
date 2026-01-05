package cluster

import (
	readonlydeployment "AtlHyper/atlhyper_agent/internal/readonly/deployment"
	modeldeployment "AtlHyper/model/k8s"
	"context"
)

func DeploymentList(ctx context.Context) ([]modeldeployment.Deployment, error) {
	m, err := readonlydeployment.ListDeployments(ctx)
	if err != nil {
		return nil, err
	}
	return m, nil
}