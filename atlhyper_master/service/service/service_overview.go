package service

import (
	"context"

	"AtlHyper/atlhyper_master/model/ui"
	"AtlHyper/atlhyper_master/repository"
	mod "AtlHyper/model/k8s"
)

// BuildServiceOverview —— 拉取全集群 Service 并构建概览 DTO
func BuildServiceOverview(ctx context.Context, clusterID string) (*ui.ServiceOverviewDTO, error) {
	list, err := repository.GetServiceListLatest(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	dto := fromModelToOverview(list)
	return &dto, nil
}

func fromModelToOverview(svcs []mod.Service) ui.ServiceOverviewDTO {
	rows := make([]ui.ServiceRowSimple, 0, len(svcs))

	var total, ext, internal, headless int
	total = len(svcs)

	for _, s := range svcs {
		if isHeadless(s) {
			headless++
		}
		if isExternal(s) {
			ext++
		} else if s.Summary.Type == "ClusterIP" && !isHeadless(s) {
			internal++
		}

		rows = append(rows, ui.ServiceRowSimple{
			Name:      s.Summary.Name,
			Namespace: s.Summary.Namespace,
			Type:      pickType(s),
			ClusterIP: firstClusterIPForTable(s),
			Ports:     formatPortsForTable(s),
			Protocol:  joinProtocols(s),
			Selector:  formatSelectorKV(s.Selector),
			CreatedAt: s.Summary.CreatedAt,
		})
	}

	return ui.ServiceOverviewDTO{
		Cards: ui.ServiceCards{
			TotalServices:    total,
			ExternalServices: ext,
			InternalServices: internal,
			HeadlessServices: headless,
		},
		Rows: rows,
	}
}
