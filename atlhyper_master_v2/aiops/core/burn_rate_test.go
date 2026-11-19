package core

import (
	"math"
	"testing"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/model_v3/slo"
)

// IngressSLO.ErrorRate 是百分数（0-100），错误预算是比率：换算后再除
func TestMaxBurnRate_UnitsMatch(t *testing.T) {
	targets := []*database.SLOTarget{{Host: "web@k8s", AvailabilityTarget: 99.9}}
	ingress := map[string]*slo.IngressSLO{"web@k8s": {ServiceKey: "web@k8s", ErrorRate: 0.5}} // 0.5% 请求失败

	burn, ok := maxBurnRate(targets, ingress)
	if !ok {
		t.Fatal("有目标也有数据时应算出燃烧率")
	}
	if math.Abs(burn-5) > 1e-9 {
		t.Errorf("燃烧率 = %v, want 5（0.5%% / 0.1%%）", burn)
	}
}

func TestMaxBurnRate_NoData(t *testing.T) {
	if _, ok := maxBurnRate([]*database.SLOTarget{{Host: "x", AvailabilityTarget: 99}}, map[string]*slo.IngressSLO{}); ok {
		t.Error("没有对应数据时不应给出燃烧率")
	}
}
