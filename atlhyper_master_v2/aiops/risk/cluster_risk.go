// atlhyper_master_v2/aiops/risk/cluster_risk.go
// ClusterRisk 聚合
package risk

import (
	"math"
	"sort"

	"AtlHyper/atlhyper_master_v2/aiops"
)

// SLOContext SLO 上下文（从 SLO 仓库获取）
type SLOContext struct {
	MaxBurnRate     float64
	ErrorGrowthRate float64
}

// Aggregate 聚合 ClusterRisk
func Aggregate(
	clusterID string,
	entityRisks map[string]*aiops.EntityRisk,
	finalRisks map[string]float64,
	sloCtx *SLOContext,
	config *RiskConfig,
	now int64,
) *aiops.ClusterRisk {
	// 1. 找最大 R_final
	var maxRFinal float64
	for _, r := range finalRisks {
		if r > maxRFinal {
			maxRFinal = r
		}
	}

	// 2. SLO burn rate factor
	sloBurnFactor := 0.0
	if sloCtx != nil {
		switch {
		case sloCtx.MaxBurnRate >= 2.0:
			sloBurnFactor = 1.0
		case sloCtx.MaxBurnRate >= 1.0:
			sloBurnFactor = 0.5
		}
	}

	// 3. 错误增长率 factor
	errorGrowthFactor := 0.0
	if sloCtx != nil && sloCtx.ErrorGrowthRate > 0 {
		errorGrowthFactor = 1.0 / (1.0 + math.Exp(-2.0*(sloCtx.ErrorGrowthRate-0.5)))
	}

	// 4. 聚合
	risk := config.ClusterWeightMax*maxRFinal*100 +
		config.ClusterWeightSLO*sloBurnFactor*100 +
		config.ClusterWeightGrowth*errorGrowthFactor*100

	if risk > 100 {
		risk = 100
	}

	// 5. Top 5 实体
	topEntities := topN(entityRisks, 5)

	// 6. 异常计数（有局部风险的实体即为异常）
	anomalyCount := 0
	for key := range finalRisks {
		if entityRisks[key] != nil && entityRisks[key].RLocal > 0 {
			anomalyCount++
		}
	}

	return &aiops.ClusterRisk{
		ClusterID:     clusterID,
		Risk:          math.Round(risk*10) / 10,
		Level:         aiops.ClusterRiskLevel(risk),
		TopEntities:   topEntities,
		TotalEntities: len(entityRisks),
		AnomalyCount:  anomalyCount,
		UpdatedAt:     now,
	}
}

// topN 返回 R_final 最高的 N 个实体
func topN(risks map[string]*aiops.EntityRisk, n int) []*aiops.EntityRisk {
	sorted := make([]*aiops.EntityRisk, 0, len(risks))
	for _, r := range risks {
		sorted = append(sorted, r)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].RFinal > sorted[j].RFinal
	})
	if len(sorted) > n {
		sorted = sorted[:n]
	}
	return sorted
}
