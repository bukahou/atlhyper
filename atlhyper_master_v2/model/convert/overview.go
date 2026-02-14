// atlhyper_master_v2/model/convert/overview.go
// model_v2 → model 集群概览转换函数
package convert

import (
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v2"
)

// Overview 转换集群概览
func Overview(src *model_v2.ClusterOverview) model.ClusterOverview {
	if src == nil {
		return model.ClusterOverview{
			Alerts: model.OverviewAlerts{
				Trend:  []model.OverviewAlertTrend{},
				Recent: []model.OverviewRecentAlert{},
			},
			Nodes: model.OverviewNodes{
				Usage: []model.OverviewNodeUsage{},
			},
		}
	}

	return model.ClusterOverview{
		ClusterID: src.ClusterID,
		Cards:     convertOverviewCards(src.Cards),
		Workloads: convertOverviewWorkloads(src.Workloads),
		Alerts:    convertOverviewAlerts(src.Alerts),
		Nodes:     convertOverviewNodes(src.Nodes),
	}
}

func convertOverviewCards(src model_v2.OverviewCards) model.OverviewCards {
	return model.OverviewCards{
		ClusterHealth: model.OverviewClusterHealth{
			Status:           src.ClusterHealth.Status,
			Reason:           src.ClusterHealth.Reason,
			NodeReadyPercent: src.ClusterHealth.NodeReadyPercent,
			PodReadyPercent:  src.ClusterHealth.PodReadyPercent,
		},
		NodeReady: model.OverviewResourceReady{
			Total:   src.NodeReady.Total,
			Ready:   src.NodeReady.Ready,
			Percent: src.NodeReady.Percent,
		},
		CPUUsage:  model.OverviewPercent{Percent: src.CPUUsage.Percent},
		MemUsage:  model.OverviewPercent{Percent: src.MemUsage.Percent},
		Events24h: src.Events24h,
	}
}

func convertOverviewWorkloads(src model_v2.OverviewWorkloads) model.OverviewWorkloads {
	result := model.OverviewWorkloads{
		Summary: model.OverviewWorkloadSummary{
			Deployments:  model.OverviewWorkloadStatus{Total: src.Summary.Deployments.Total, Ready: src.Summary.Deployments.Ready},
			DaemonSets:   model.OverviewWorkloadStatus{Total: src.Summary.DaemonSets.Total, Ready: src.Summary.DaemonSets.Ready},
			StatefulSets: model.OverviewWorkloadStatus{Total: src.Summary.StatefulSets.Total, Ready: src.Summary.StatefulSets.Ready},
			Jobs: model.OverviewJobStatus{
				Total: src.Summary.Jobs.Total, Running: src.Summary.Jobs.Running,
				Succeeded: src.Summary.Jobs.Succeeded, Failed: src.Summary.Jobs.Failed,
			},
		},
		PodStatus: model.OverviewPodStatus{
			Total:            src.PodStatus.Total,
			Running:          src.PodStatus.Running,
			Pending:          src.PodStatus.Pending,
			Failed:           src.PodStatus.Failed,
			Succeeded:        src.PodStatus.Succeeded,
			Unknown:          src.PodStatus.Unknown,
			RunningPercent:   src.PodStatus.RunningPercent,
			PendingPercent:   src.PodStatus.PendingPercent,
			FailedPercent:    src.PodStatus.FailedPercent,
			SucceededPercent: src.PodStatus.SucceededPercent,
		},
	}

	if src.PeakStats != nil {
		result.PeakStats = &model.OverviewPeakStats{
			PeakCPU:     src.PeakStats.PeakCPU,
			PeakCPUNode: src.PeakStats.PeakCPUNode,
			PeakMem:     src.PeakStats.PeakMem,
			PeakMemNode: src.PeakStats.PeakMemNode,
			HasData:     src.PeakStats.HasData,
		}
	}

	return result
}

func convertOverviewAlerts(src model_v2.OverviewAlerts) model.OverviewAlerts {
	trend := make([]model.OverviewAlertTrend, len(src.Trend))
	for i, t := range src.Trend {
		kinds := t.Kinds
		if kinds == nil {
			kinds = map[string]int{}
		}
		trend[i] = model.OverviewAlertTrend{
			At:    t.At,
			Kinds: kinds,
		}
	}

	recent := make([]model.OverviewRecentAlert, len(src.Recent))
	for i, r := range src.Recent {
		recent[i] = model.OverviewRecentAlert{
			Timestamp: r.Timestamp,
			Severity:  r.Severity,
			Kind:      r.Kind,
			Namespace: r.Namespace,
			Name:      r.Name,
			Message:   r.Message,
			Reason:    r.Reason,
		}
	}

	return model.OverviewAlerts{
		Trend: trend,
		Totals: model.OverviewAlertTotals{
			Critical: src.Totals.Critical,
			Warning:  src.Totals.Warning,
			Info:     src.Totals.Info,
		},
		Recent: recent,
	}
}

func convertOverviewNodes(src model_v2.OverviewNodes) model.OverviewNodes {
	usage := make([]model.OverviewNodeUsage, len(src.Usage))
	for i, u := range src.Usage {
		usage[i] = model.OverviewNodeUsage{
			Node:     u.Node,
			CPUUsage: u.CPUUsage,
			MemUsage: u.MemUsage,
		}
	}
	return model.OverviewNodes{Usage: usage}
}
