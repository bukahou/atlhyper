package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"github.com/google/jsonschema-go/jsonschema"

	"AtlHyper/atlhyper_master_v2/ai/toolkit"
	"AtlHyper/atlhyper_master_v2/aiops"
)

const maxAIOpsListSize = 50

type analyzeIncidentParams struct {
	IncidentID string `json:"incident_id" jsonschema:"事件 ID，如 inc-1737364200（从 get_recent_incidents 取）"`
}

type clusterRiskParams struct {
	TopN int `json:"top_n,omitempty" jsonschema:"返回风险最高的前 N 个实体（不填为 10）"`
}

type recentIncidentsParams struct {
	State string `json:"state,omitempty" jsonschema:"按状态过滤，不填返回全部"`
	Limit int    `json:"limit,omitempty" jsonschema:"返回数量（不填为 10）"`
}

type entityDetailParams struct {
	EntityType string `json:"entity_type" jsonschema:"实体类型"`
	EntityName string `json:"entity_name" jsonschema:"实体名称"`
	Namespace  string `json:"namespace,omitempty" jsonschema:"命名空间（Pod / Service / Ingress 需要；Node 不填）"`
}

func aiopsTools(d Deps) []*toolkit.Tool {
	listLimit := func(name string) func(*jsonschema.Schema) {
		return func(s *jsonschema.Schema) {
			s.Properties[name].Minimum = ptr(1.0)
			s.Properties[name].Maximum = ptr(float64(maxAIOpsListSize))
		}
	}
	return []*toolkit.Tool{
		toolkit.MustDefine(toolkit.Spec[analyzeIncidentParams]{
			Name:        "analyze_incident",
			Description: "对指定事件做 AI 分析：根因、影响面、处置建议。用户要深入了解某个事件时用。",
			Permission:  toolkit.PermAIAnalyze,
			Handler: func(ctx context.Context, _ string, p analyzeIncidentParams) (string, error) {
				result, err := d.SummarizeIncident(ctx, p.IncidentID)
				if err != nil {
					return "", fmt.Errorf("分析事件失败: %w", err)
				}
				return marshal(result)
			},
		}),
		toolkit.MustDefine(toolkit.Spec[clusterRiskParams]{
			Name:        "get_cluster_risk",
			Description: "集群当前的风险评分（0–100）与风险最高的实体。想知道「现在整体健康吗、哪里最可疑」时先用它。",
			Permission:  toolkit.PermAIOpsRead,
			Refine:      listLimit("top_n"),
			Handler: func(_ context.Context, clusterID string, p clusterRiskParams) (string, error) {
				topN := orDefault(p.TopN, 10)
				risk := d.AIOps.GetClusterRisk(clusterID)
				entities := d.AIOps.GetEntityRisks(clusterID, "r_final", topN)
				return marshal(map[string]any{
					"clusterRisk": scaleClusterRisk(risk),
					"topEntities": scaleEntityRisks(entities),
					"entityCount": len(entities),
				})
			},
		}),
		toolkit.MustDefine(toolkit.Spec[recentIncidentsParams]{
			Name:        "get_recent_incidents",
			Description: "最近的事件列表（AIOps 检测到的异常），可按状态过滤。了解近期发生过什么时用。",
			Permission:  toolkit.PermAIOpsRead,
			Refine: func(s *jsonschema.Schema) {
				s.Properties["state"].Enum = []any{"warning", "incident", "recovery", "stable"}
				listLimit("limit")(s)
			},
			Handler: func(ctx context.Context, clusterID string, p recentIncidentsParams) (string, error) {
				incidents, total, err := d.AIOps.GetIncidents(ctx, aiops.IncidentQueryOpts{
					ClusterID: clusterID, State: p.State, Limit: orDefault(p.Limit, 10),
				})
				if err != nil {
					return "", fmt.Errorf("获取事件列表失败: %w", err)
				}
				return marshal(map[string]any{"incidents": incidents, "total": total})
			},
		}),
		toolkit.MustDefine(toolkit.Spec[entityDetailParams]{
			Name: "get_entity_detail",
			Description: "某个实体（Pod / Service / Node / Ingress）的风险详情：风险分数、异常指标、因果树（上下游的异常实体）、传播路径。" +
				"深挖某个实体为什么异常时用。",
			Permission: toolkit.PermAIOpsRead,
			Refine: func(s *jsonschema.Schema) {
				s.Properties["entity_type"].Enum = []any{"pod", "service", "node", "ingress"}
			},
			Handler: func(_ context.Context, clusterID string, p entityDetailParams) (string, error) {
				entityKey := entityKeyFor(p.EntityType, p.Namespace, p.EntityName)
				detail := d.AIOps.GetEntityRisk(clusterID, entityKey)
				if detail == nil {
					return fmt.Sprintf("未找到实体 %s（不在依赖图中，或没有异常数据）", entityKey), nil
				}
				return marshal(simplifyEntityDetail(detail))
			},
		}),
	}
}

// entityKeyFor 模型给的类型 / 命名空间 / 名称 → 实体键。命名空间级对象没给命名空间时按 default 查
func entityKeyFor(entityType, namespace, name string) string {
	if namespace == "" && entityType != "node" {
		namespace = "default"
	}
	return aiops.EntityKey(namespace, entityType, name)
}

// simplifyEntityDetail 只留异常指标，减少模型上下文消耗；分数换成百分制
func simplifyEntityDetail(detail *aiops.EntityRiskDetail) map[string]any {
	var anomalyMetrics []map[string]any
	for _, m := range detail.Metrics {
		if m.IsAnomaly {
			anomalyMetrics = append(anomalyMetrics, map[string]any{
				"metricName":   m.MetricName,
				"currentValue": m.CurrentValue,
				"baseline":     m.Baseline,
				"deviation":    m.Deviation,
			})
		}
	}
	return map[string]any{
		"entityKey":   detail.EntityKey,
		"entityType":  detail.EntityType,
		"namespace":   detail.Namespace,
		"name":        detail.Name,
		"rFinal":      math.Round(detail.RFinal * 100),
		"riskLevel":   detail.RiskLevel,
		"metrics":     anomalyMetrics,
		"causalTree":  detail.CausalTree,
		"propagation": detail.Propagation,
	}
}

// scaleEntityRisks 实体风险分数从 [0,1] 换成 [0,100]
func scaleEntityRisks(src []*aiops.EntityRisk) []map[string]any {
	result := make([]map[string]any, len(src))
	for i, r := range src {
		result[i] = map[string]any{
			"entityKey":    r.EntityKey,
			"entityType":   r.EntityType,
			"namespace":    r.Namespace,
			"name":         r.Name,
			"rFinal":       math.Round(r.RFinal * 100),
			"riskLevel":    r.RiskLevel,
			"firstAnomaly": r.FirstAnomaly,
		}
	}
	return result
}

// scaleClusterRisk 集群风险（Risk 已是 0–100）
func scaleClusterRisk(src *aiops.ClusterRisk) map[string]any {
	if src == nil {
		return nil
	}
	return map[string]any{
		"clusterId":     src.ClusterID,
		"risk":          src.Risk,
		"level":         src.Level,
		"totalEntities": src.TotalEntities,
		"anomalyCount":  src.AnomalyCount,
	}
}

func marshal(v any) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("结果序列化失败: %w", err)
	}
	return string(data), nil
}

func orDefault(v, def int) int {
	if v == 0 {
		return def
	}
	return v
}
