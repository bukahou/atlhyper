// atlhyper_master_v2/aiops/risk/config.go
// 风险评分权重配置
package risk

// RiskConfig 风险评分配置
type RiskConfig struct {
	// Stage 1: 局部风险权重 (按实体类型分组)
	Weights map[string]map[string]float64

	// Stage 2: 时序参数
	TemporalHalfLife float64 // τ (秒)，默认 300 (5 分钟)

	// Stage 3: 传播参数
	SelfWeight float64 // α，默认 0.6 (自身 60%，传播 40%)

	// ClusterRisk 聚合权重
	ClusterWeightMax    float64 // w1，默认 0.5
	ClusterWeightSLO    float64 // w2，默认 0.3
	ClusterWeightGrowth float64 // w3，默认 0.2
}

// DefaultRiskConfig 返回默认配置
func DefaultRiskConfig() *RiskConfig {
	return &RiskConfig{
		Weights: map[string]map[string]float64{
			"service": {
				"error_rate":   0.40,
				"avg_latency":  0.30,
				"request_rate": 0.20,
			},
			"pod": {
				"restart_count": 0.50,
				"is_running":    0.50,
			},
			"node": {
				"memory_usage": 0.25,
				"cpu_usage":    0.25,
				"disk_usage":   0.20,
				"psi_cpu":      0.10,
				"psi_memory":   0.10,
				"psi_io":       0.10,
			},
			"ingress": {
				"error_rate":  0.50,
				"avg_latency": 0.50,
			},
		},
		TemporalHalfLife:    300,
		SelfWeight:          0.6,
		ClusterWeightMax:    0.5,
		ClusterWeightSLO:    0.3,
		ClusterWeightGrowth: 0.2,
	}
}

// GetWeights 获取指定实体类型的指标权重
func (c *RiskConfig) GetWeights(entityType string) map[string]float64 {
	if w, ok := c.Weights[entityType]; ok {
		return w
	}
	return map[string]float64{}
}
