// atlhyper_master_v2/gateway/handler/aiops/scale_risk.go
//
// 风险分数单位约定（项目全局）:
//   - AIOps 核心层 (aiops/risk/): [0, 1] 浮点概率（算法友好）
//   - Gateway / API 层:           [0, 100] 整数（前端友好，已 Scale）
//   - 前端 (atlhyper_web/src/lib/risk.ts): 全部按 [0, 100] 处理
//
// 本文件的 ScaleEntityRisk / ScaleEntityRiskDetail 负责边界转换。
// 所有暴露给前端的 EntityRisk / EntityRiskDetail 响应都必须经过 Scale。
//
// 注意：曾出现过前端 topology 组件在收到 [0,100] 分数后再次 × 100，
// 导致节点 badge 显示 4500（实际 45%）的单位混用 bug（2026-04 修复）。
// 修改此文件或前端任何风险分数显示代码时，务必保持单位一致性，
// 前端必须通过 @/lib/risk 的共享函数消费，禁止散落的 × 100 / 阈值写法。
package aiops

import (
	"math"

	"AtlHyper/atlhyper_master_v2/aiops"
)

// toPercent 将 [0,1] 转换为 [0,100] 整数
func toPercent(v float64) float64 {
	return math.Round(v * 100)
}

// ScaleEntityRisk 转换单个 EntityRisk（返回副本）
func ScaleEntityRisk(src *aiops.EntityRisk) *aiops.EntityRisk {
	if src == nil {
		return nil
	}
	cp := *src
	cp.RLocal = toPercent(cp.RLocal)
	cp.WTime = math.Round(cp.WTime*100) / 100 // wTime 保留两位小数（权重系数）
	cp.RWeighted = toPercent(cp.RWeighted)
	cp.RFinal = toPercent(cp.RFinal)
	return &cp
}

// ScaleEntityRisks 批量转换 EntityRisk 列表
func ScaleEntityRisks(src []*aiops.EntityRisk) []*aiops.EntityRisk {
	if src == nil {
		return nil
	}
	result := make([]*aiops.EntityRisk, len(src))
	for i, r := range src {
		result[i] = ScaleEntityRisk(r)
	}
	return result
}

// ScaleClusterRisk 转换 ClusterRisk（Risk 已是 0-100，只转 TopEntities）
func ScaleClusterRisk(src *aiops.ClusterRisk) *aiops.ClusterRisk {
	if src == nil {
		return nil
	}
	cp := *src
	cp.TopEntities = ScaleEntityRisks(cp.TopEntities)
	return &cp
}

// ScaleEntityRiskDetail 转换 EntityRiskDetail（含 CausalTree 递归）
func ScaleEntityRiskDetail(src *aiops.EntityRiskDetail) *aiops.EntityRiskDetail {
	if src == nil {
		return nil
	}
	cp := *src
	cp.EntityRisk = *ScaleEntityRisk(&src.EntityRisk)
	cp.CausalTree = scaleCausalTree(src.CausalTree)
	return &cp
}

// scaleCausalTree 递归转换因果树节点
func scaleCausalTree(nodes []*aiops.CausalTreeNode) []*aiops.CausalTreeNode {
	if nodes == nil {
		return nil
	}
	result := make([]*aiops.CausalTreeNode, len(nodes))
	for i, n := range nodes {
		cp := *n
		cp.RFinal = toPercent(cp.RFinal)
		cp.Children = scaleCausalTree(n.Children)
		result[i] = &cp
	}
	return result
}

// ScaleIncidentDetail 转换事件详情中的实体分数
func ScaleIncidentDetail(src *aiops.IncidentDetail) *aiops.IncidentDetail {
	if src == nil {
		return nil
	}
	cp := *src
	cp.PeakRisk = toPercent(cp.PeakRisk)
	if src.Entities != nil {
		cp.Entities = make([]*aiops.IncidentEntity, len(src.Entities))
		for i, e := range src.Entities {
			eCp := *e
			eCp.RLocal = toPercent(eCp.RLocal)
			eCp.RFinal = toPercent(eCp.RFinal)
			cp.Entities[i] = &eCp
		}
	}
	return &cp
}
