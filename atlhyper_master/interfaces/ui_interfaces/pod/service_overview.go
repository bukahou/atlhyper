package pod

import (
	"context"
	"strings"

	"AtlHyper/atlhyper_master/interfaces/datasource"
)

func BuildPodOverview(ctx context.Context, clusterID string) (*PodOverviewDTO, error) {
    pods, err := datasource.GetPodListLatest(ctx, clusterID)
    if err != nil {
        return nil, err
    }

    var running, pending, failed, unknown int
    items := make([]PodOverviewItem, 0, len(pods))

    for _, p := range pods {
        switch strings.ToLower(p.Summary.Phase) {
        case "running":
            running++
        case "pending":
            pending++
        case "failed":
            failed++
        default:
            unknown++
        }

        // Deployment 名（ControlledBy）
        deployment := ""
        if p.Summary.ControlledBy != nil {
            deployment = p.Summary.ControlledBy.Name
        }

        // Pod Metrics
        var (
            cpu    string
            cpuPct float64
            mem    string
            memPct float64
        )
        if p.Metrics != nil {
            cpu = p.Metrics.CPU.Usage
            cpuPct = p.Metrics.CPU.UtilPct
            mem = p.Metrics.Memory.Usage
            memPct = p.Metrics.Memory.UtilPct
        }

        items = append(items, PodOverviewItem{
            Namespace:  p.Summary.Namespace,
            Deployment: deployment,
            Name:       p.Summary.Name,
            Ready:      p.Summary.Ready,
            Phase:      p.Summary.Phase,
            Restarts:   p.Summary.Restarts,
            CPU:        cpu,
            CPUPercent: cpuPct,
            Memory:     mem,
            MemPercent: memPct,
            StartTime:  p.Summary.StartTime,
            Node:       p.Summary.Node,
        })
    }

    return &PodOverviewDTO{
        Cards: PodCards{
            Running: running,
            Pending: pending,
            Failed:  failed,
            Unknown: unknown,
        },
        Pods: items,
    }, nil
}
