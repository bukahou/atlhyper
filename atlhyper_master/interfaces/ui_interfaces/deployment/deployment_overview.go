// ui_interfaces/deployment/overview.go
package deployment

import (
	"context"
	"fmt"

	"AtlHyper/atlhyper_master/interfaces/datasource"
	mod "AtlHyper/model/deployment"
)

// BuildDeploymentOverview —— 聚合 Deployment 概览
func BuildDeploymentOverview(ctx context.Context, clusterID string) (*DeploymentOverviewDTO, error) {
	list, err := datasource.GetDeploymentListLatest(ctx, clusterID)
	if err != nil {
		return nil, fmt.Errorf("get deployment list failed: %w", err)
	}
	dto := fromModelToOverview(list)
	return &dto, nil
}

func fromModelToOverview(items []mod.Deployment) DeploymentOverviewDTO {
	rows := make([]DeploymentRowSimple, 0, len(items))

	nsSet := map[string]struct{}{}
	totalDesired := 0
	totalReady := 0

	for _, d := range items {
		nsSet[d.Summary.Namespace] = struct{}{}
		totalDesired += int(d.Summary.Replicas)
		totalReady += int(d.Summary.Ready)

		rows = append(rows, DeploymentRowSimple{
			Namespace:  d.Summary.Namespace,
			Name:       d.Summary.Name,
			Image:      joinImagesShort(d.Template.Containers),
			Replicas:   fmt.Sprintf("%d/%d", d.Summary.Ready, d.Summary.Replicas),
			LabelCount: mapLen(d.Labels),
			AnnoCount:  mapLen(d.Annotations),
			CreatedAt:  d.Summary.CreatedAt,
		})
	}

	return DeploymentOverviewDTO{
		Cards: OverviewCards{
			TotalDeployments: len(items),
			Namespaces:       len(nsSet),
			TotalReplicas:    totalDesired,
			ReadyReplicas:    totalReady,
		},
		Rows: rows,
	}
}
