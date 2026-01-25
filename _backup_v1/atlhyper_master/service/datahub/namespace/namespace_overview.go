// ui_interfaces/namespace/overview.go
package namespace

import (
	"context"
	"strings"

	"AtlHyper/atlhyper_master/model/dto"
	"AtlHyper/atlhyper_master/repository"
	mod "AtlHyper/model/k8s"
)

// BuildNamespaceOverview —— 拉取全集群 NS，构建概览 DTO
func BuildNamespaceOverview(ctx context.Context, clusterID string) (*dto.NamespaceOverviewDTO, error) {
	list, err := repository.Mem.GetNamespaceListLatest(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	dto := fromModelToOverview(list)
	return &dto, nil
}

func fromModelToOverview(items []mod.Namespace) dto.NamespaceOverviewDTO {
	rows := make([]dto.NamespaceRowDTO, 0, len(items))

	var total, active, term, totalPods int
	total = len(items)

	for _, ns := range items {
		status := ns.Summary.Phase
		if strings.EqualFold(status, "Active") {
			active++
		} else {
			term++
		}
		totalPods += ns.Counts.Pods

		rows = append(rows, dto.NamespaceRowDTO{
			Name:            ns.Summary.Name,
			Status:          status,
			PodCount:        ns.Counts.Pods,
			LabelCount:      mapLen(ns.Summary.Labels),
			AnnotationCount: mapLen(ns.Summary.Annotations),
			CreatedAt:       ns.Summary.CreatedAt,
		})
	}

	return dto.NamespaceOverviewDTO{
		Cards: dto.NamespaceOverviewCards{
			TotalNamespaces: total,
			ActiveCount:     active,
			Terminating:     term,
			TotalPods:       totalPods,
		},
		Rows: rows,
	}
}
