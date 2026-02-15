// atlhyper_master_v2/aiops/risk/local.go
// Stage 1: 局部风险计算
// R_local(entity) = Σ(w_i × score_i)
package risk

import "AtlHyper/atlhyper_master_v2/aiops"

// ComputeLocalRisks 计算每个实体的局部风险分数
func ComputeLocalRisks(anomalies []*aiops.AnomalyResult, config *RiskConfig) map[string]float64 {
	// 按 entityKey 分组
	byEntity := map[string][]*aiops.AnomalyResult{}
	for _, a := range anomalies {
		byEntity[a.EntityKey] = append(byEntity[a.EntityKey], a)
	}

	localRisks := make(map[string]float64, len(byEntity))
	for entityKey, results := range byEntity {
		entityType := aiops.ExtractEntityType(entityKey)
		weights := config.GetWeights(entityType)

		var rLocal float64
		for _, r := range results {
			if !r.IsAnomaly {
				continue
			}
			w, ok := weights[r.MetricName]
			if !ok {
				w = 0.1 // 未配置的指标默认权重
			}
			rLocal += w * r.Score
		}

		// 截断到 [0, 1]
		if rLocal > 1.0 {
			rLocal = 1.0
		}
		if rLocal > 0 {
			localRisks[entityKey] = rLocal
		}
	}

	return localRisks
}
