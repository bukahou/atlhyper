package node

import (
	"context"
	"strings"

	"AtlHyper/atlhyper_master/interfaces/datasource"
	mod "AtlHyper/model/node"
)

// BuildNodeOverview —— 读取全集群节点并聚合成概览 DTO
func BuildNodeOverview(ctx context.Context, clusterID string) (*NodeOverviewDTO, error) {
	nodes, err := datasource.GetNodeListLatest(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	dto := fromModelToOverview(nodes)
	return &dto, nil
}

// fromModelToOverview —— Store 模型 → 概览 DTO
func fromModelToOverview(nodes []mod.Node) NodeOverviewDTO {
	rows := make([]NodeRowSimple, 0, len(nodes))
	var totalCPU int
	var totalMemGiB float64
	var ready int

	for _, n := range nodes {
		r := strings.EqualFold(n.Summary.Ready, "true")
		if r {
			ready++
		}

		cores := parseCPUToInt(n.Capacity.CPU)
		memGiB := parseMemToGiB(n.Capacity.Memory)

		totalCPU += cores
		totalMemGiB += memGiB

		rows = append(rows, NodeRowSimple{
			Name:         n.Summary.Name,
			Ready:        r,
			InternalIP:   n.Addresses.InternalIP,
			OSImage:      n.Info.OSImage,
			Architecture: n.Info.Architecture,
			CPUCores:     cores,
			MemoryGiB:    round1(memGiB),
			Schedulable:  n.Summary.Schedulable,
		})
	}

	return NodeOverviewDTO{
		Cards: NodeCards{
			TotalNodes:     len(nodes),
			ReadyNodes:     ready,
			TotalCPU:       totalCPU,
			TotalMemoryGiB: round1(totalMemGiB),
		},
		Rows: rows,
	}
}
