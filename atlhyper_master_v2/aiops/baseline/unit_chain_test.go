package baseline

import (
	"testing"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/model_v3/apm"
	"AtlHyper/model_v3/cluster"
)

// 从快照提取到硬线判定串起来测：提取出的值与阈值必须是同一个单位。
// 两头各自单测时各按各的单位假设都能通过（提取按比率、阈值按百分数），
// 串起来才看得出「10% 的错误率永远碰不到 5% 的硬线」。
func TestAPMErrorRate_ExtractThenAbsoluteGuard(t *testing.T) {
	otel := &cluster.OTelSnapshot{APMServices: []apm.APMService{
		{Name: "checkout", Namespace: "shop", SuccessRate: 0.90}, // 10% 请求失败
		{Name: "catalog", Namespace: "shop", SuccessRate: 0.995}, // 0.5% 失败
	}}
	breach := map[string]bool{}
	for _, p := range extractAPMMetrics(otel) {
		if p.MetricName != "apm_error_rate" {
			continue
		}
		st := &aiops.BaselineState{
			EntityKey: p.EntityKey, MetricName: p.MetricName,
			EMA: p.Value, Variance: 1e6, // 统计通道必然判正常，只看硬线
			Count: aiops.ColdStartMinCount + 1,
		}
		_, res := Detect(st, p.Value, 0)
		breach[p.EntityKey] = res.AbsoluteBreach
	}
	if !breach[aiops.EntityKey("shop", "service", "checkout")] {
		t.Error("10% 错误率应触碰 5% 的硬线")
	}
	if breach[aiops.EntityKey("shop", "service", "catalog")] {
		t.Error("0.5% 错误率不应触碰硬线")
	}
}
