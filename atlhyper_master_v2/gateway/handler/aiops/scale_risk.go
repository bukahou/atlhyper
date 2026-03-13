// atlhyper_master_v2/gateway/handler/aiops/scale_risk.go
// 风险分数输出转换：内部 [0,1] → API [0,100]
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
