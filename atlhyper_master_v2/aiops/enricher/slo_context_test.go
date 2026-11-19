package enricher

import (
	"strings"
	"testing"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/model_v3/cluster"
	"AtlHyper/model_v3/slo"
)

// 入口数据按服务标识（namespace/service）上报：事件里有这个服务（或它的入口实体）就带上它的 SLO，
// 错误率原样是百分数，⛔ 不再乘 100
func TestBuildOTelContext_SLOForAffectedService(t *testing.T) {
	otel := &cluster.OTelSnapshot{SLOIngress: []slo.IngressSLO{
		{ServiceKey: "shop/api", SuccessRate: 99.5, ErrorRate: 0.5, P99Ms: 120, RPS: 3},
		{ServiceKey: "other/web", SuccessRate: 90, ErrorRate: 10},
	}}
	entities := []*database.AIOpsIncidentEntity{{EntityKey: "shop/service/api"}}

	_, _, sloCtx := buildOTelContext(otel, entities)

	if !strings.Contains(sloCtx, "shop/api: SuccessRate=99.50% ErrorRate=0.5000%") {
		t.Errorf("应带上 shop/api 的 SLO 且单位不放大，实际:\n%s", sloCtx)
	}
	if strings.Contains(sloCtx, "other/web") {
		t.Errorf("不相关的入口不应出现:\n%s", sloCtx)
	}
}
