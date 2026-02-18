// atlhyper_master_v2/aiops/risk/config.go
// 风险评分权重配置
package risk

// MetricChannel 指标通道类型
type MetricChannel int

const (
	ChannelStatistical   MetricChannel = iota // 统计通道 (EMA 连续指标)
	ChannelDeterministic                      // 确定性通道 (确定性直注指标)
	ChannelBoth                               // 双通道 (同时参与统计和确定性)
)

// MetricConfig 指标配置 (权重 + 通道)
type MetricConfig struct {
	Weight  float64
	Channel MetricChannel
}

// RiskConfig 风险评分配置
type RiskConfig struct {
	// Stage 1: 局部风险 (按实体类型 → 指标名 → 配置)
	MetricConfigs map[string]map[string]MetricConfig

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
		MetricConfigs: map[string]map[string]MetricConfig{
			"service": {
				"error_rate":   {Weight: 0.40, Channel: ChannelStatistical},
				"avg_latency":  {Weight: 0.30, Channel: ChannelStatistical},
				"request_rate": {Weight: 0.20, Channel: ChannelStatistical},
			},
			"pod": {
				"restart_count":          {Weight: 0.20, Channel: ChannelBoth},
				"is_running":             {Weight: 0.10, Channel: ChannelBoth},
				"not_ready_containers":   {Weight: 0.20, Channel: ChannelBoth},
				"max_container_restarts": {Weight: 0.10, Channel: ChannelBoth},
				"container_anomaly":      {Weight: 0.25, Channel: ChannelDeterministic},
				"critical_event":         {Weight: 0.15, Channel: ChannelDeterministic},
				"deployment_impact":      {Weight: 0.25, Channel: ChannelDeterministic},
			},
			"node": {
				"memory_usage": {Weight: 0.25, Channel: ChannelStatistical},
				"cpu_usage":    {Weight: 0.25, Channel: ChannelStatistical},
				"disk_usage":   {Weight: 0.20, Channel: ChannelStatistical},
				"psi_cpu":      {Weight: 0.10, Channel: ChannelStatistical},
				"psi_memory":   {Weight: 0.10, Channel: ChannelStatistical},
				"psi_io":       {Weight: 0.10, Channel: ChannelStatistical},
			},
			"ingress": {
				"error_rate":  {Weight: 0.50, Channel: ChannelStatistical},
				"avg_latency": {Weight: 0.50, Channel: ChannelStatistical},
			},
		},
		TemporalHalfLife:    300,
		SelfWeight:          0.6,
		ClusterWeightMax:    0.5,
		ClusterWeightSLO:    0.3,
		ClusterWeightGrowth: 0.2,
	}
}

// GetWeights 获取指定实体类型的指标权重 (兼容方法，从 MetricConfig 提取 Weight)
func (c *RiskConfig) GetWeights(entityType string) map[string]float64 {
	configs := c.GetMetricConfigs(entityType)
	weights := make(map[string]float64, len(configs))
	for name, cfg := range configs {
		weights[name] = cfg.Weight
	}
	return weights
}

// GetMetricConfigs 获取指定实体类型的指标配置
func (c *RiskConfig) GetMetricConfigs(entityType string) map[string]MetricConfig {
	if m, ok := c.MetricConfigs[entityType]; ok {
		return m
	}
	return map[string]MetricConfig{}
}
