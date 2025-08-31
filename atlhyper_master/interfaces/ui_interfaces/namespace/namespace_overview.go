// ui_interfaces/namespace/overview.go
package namespace

import (
	"context"
	"strings"

	"AtlHyper/atlhyper_master/interfaces/datasource"
	mod "AtlHyper/model/namespace"
)

// BuildNamespaceOverview —— 拉取全集群 NS，构建概览 DTO
func BuildNamespaceOverview(ctx context.Context, clusterID string) (*NamespaceOverviewDTO, error) {
	list, err := datasource.GetNamespaceListLatest(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	dto := fromModelToOverview(list)
	return &dto, nil
}

func fromModelToOverview(items []mod.Namespace) NamespaceOverviewDTO {
	rows := make([]NamespaceRowDTO, 0, len(items))

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

		rows = append(rows, NamespaceRowDTO{
			Name:            ns.Summary.Name,
			Status:          status,
			PodCount:        ns.Counts.Pods,
			LabelCount:      mapLen(ns.Summary.Labels),
			AnnotationCount: mapLen(ns.Summary.Annotations),
			CreatedAt:       ns.Summary.CreatedAt,
		})
	}

	return NamespaceOverviewDTO{
		Cards: OverviewCards{
			TotalNamespaces: total,
			ActiveCount:     active,
			Terminating:     term,
			TotalPods:       totalPods,
		},
		Rows: rows,
	}
}
