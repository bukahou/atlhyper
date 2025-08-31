package service

import (
	"context"

	"AtlHyper/atlhyper_master/interfaces/datasource"
	mod "AtlHyper/model/service"
)

// BuildServiceOverview —— 拉取全集群 Service 并构建概览 DTO
func BuildServiceOverview(ctx context.Context, clusterID string) (*ServiceOverviewDTO, error) {
	list, err := datasource.GetServiceListLatest(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	dto := fromModelToOverview(list)
	return &dto, nil
}

func fromModelToOverview(svcs []mod.Service) ServiceOverviewDTO {
	rows := make([]ServiceRowSimple, 0, len(svcs))

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

		rows = append(rows, ServiceRowSimple{
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

	return ServiceOverviewDTO{
		Cards: ServiceCards{
			TotalServices:    total,
			ExternalServices: ext,
			InternalServices: internal,
			HeadlessServices: headless,
		},
		Rows: rows,
	}
}
