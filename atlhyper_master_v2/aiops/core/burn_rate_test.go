package core

import (
	"math"
	"testing"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/model_v3/slo"
)

// IngressSLO.ErrorRate 是百分数（0-100），可用性目标也是百分数：0.5% 失败 ÷ 0.1% 预算 = 5
func TestMaxBurnRate_UnitsMatch(t *testing.T) {
	targets := []*database.SLOTarget{{Host: "shop/web", AvailabilityTarget: 99.9}}
	ingress := []slo.IngressSLO{{ServiceKey: "shop/web", ErrorRate: 0.5}}

	burn, ok := maxBurnRate(targets, ingress)
	if !ok {
		t.Fatal("有目标也有数据时应算出燃烧率")
	}
	if math.Abs(burn-5) > 1e-9 {
		t.Errorf("燃烧率 = %v, want 5（0.5%% / 0.1%%）", burn)
	}
}

// 目标按域名登记（SLO 页面上就是这么设的），入口的服务标识是 namespace/service：按域名也要找得到
func TestMaxBurnRate_TargetByDomain(t *testing.T) {
	targets := []*database.SLOTarget{{Host: "shop.example.com", AvailabilityTarget: 99}}
	ingress := []slo.IngressSLO{
		{ServiceKey: "shop/web", DisplayName: "shop.example.com", ErrorRate: 2},
		{ServiceKey: "other/api", DisplayName: "api.example.com", ErrorRate: 50},
	}
	burn, ok := maxBurnRate(targets, ingress)
	if !ok || math.Abs(burn-2) > 1e-9 {
		t.Errorf("燃烧率 = %v (ok=%v), want 2（2%% / 1%%），只算对上域名的那个入口", burn, ok)
	}
}

func TestMaxBurnRate_NoData(t *testing.T) {
	if _, ok := maxBurnRate([]*database.SLOTarget{{Host: "x", AvailabilityTarget: 99}}, nil); ok {
		t.Error("没有对应数据时不应给出燃烧率")
	}
}
