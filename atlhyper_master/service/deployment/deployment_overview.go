// ui_interfaces/deployment/overview.go
package deployment

import (
	"context"
	"fmt"

	"AtlHyper/atlhyper_master/model/ui"
	"AtlHyper/atlhyper_master/repository"
	mod "AtlHyper/model/k8s"
)

// BuildDeploymentOverview —— 聚合 Deployment 概览
func BuildDeploymentOverview(ctx context.Context, clusterID string) (*ui.DeploymentOverviewDTO, error) {
	list, err := repository.GetDeploymentListLatest(ctx, clusterID)
	if err != nil {
		return nil, fmt.Errorf("get deployment list failed: %w", err)
	}
	dto := fromModelToOverview(list)
	return &dto, nil
}

func fromModelToOverview(items []mod.Deployment) ui.DeploymentOverviewDTO {
	rows := make([]ui.DeploymentRowSimple, 0, len(items))

	nsSet := map[string]struct{}{}
	totalDesired := 0
	totalReady := 0

	for _, d := range items {
		nsSet[d.Summary.Namespace] = struct{}{}
		totalDesired += int(d.Summary.Replicas)
		totalReady += int(d.Summary.Ready)

		rows = append(rows, ui.DeploymentRowSimple{
			Namespace:  d.Summary.Namespace,
			Name:       d.Summary.Name,
			Image:      joinImagesShort(d.Template.Containers),
			Replicas:   fmt.Sprintf("%d/%d", d.Summary.Ready, d.Summary.Replicas),
			LabelCount: mapLen(d.Labels),
			AnnoCount:  mapLen(d.Annotations),
			CreatedAt:  d.Summary.CreatedAt,
		})
	}

	return ui.DeploymentOverviewDTO{
		Cards: ui.DeploymentOverviewCards{
			TotalDeployments: len(items),
			Namespaces:       len(nsSet),
			TotalReplicas:    totalDesired,
			ReadyReplicas:    totalReady,
		},
		Rows: rows,
	}
}
