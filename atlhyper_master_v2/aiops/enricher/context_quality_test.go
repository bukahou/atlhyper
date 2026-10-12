package enricher

import (
	"strings"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

// A1：核心层风险是 [0,1]，界面显示 [0,100]（gateway/handler/aiops/scale_risk.go）。
// 给模型的提示词必须与界面同一刻度：0.48 曾被 %.0f 写成「峰值风险: 0」，模型据此报告「峰值风险为 0」。
func TestBuildIncidentContext_RiskUsesDisplayScale(t *testing.T) {
	inc := &database.AIOpsIncident{ID: "inc-1", State: "stable", PeakRisk: 0.48, StartedAt: time.Now(), RootCause: "ns/pod/a"}
	entities := []*database.AIOpsIncidentEntity{
		{IncidentID: "inc-1", EntityKey: "ns/pod/a", Role: "root_cause", RLocal: 0.31, RFinal: 0.48},
		{IncidentID: "inc-1", EntityKey: "ns/svc/b", Role: "affected", RFinal: 0.2},
	}
	ctx := BuildIncidentContext(inc, entities, nil, nil)

	if !strings.Contains(ctx.IncidentSummary, "峰值风险: 48") {
		t.Errorf("summary 应写 48（0-100 刻度）：%s", ctx.IncidentSummary)
	}
	if !strings.Contains(ctx.RootCauseEntity, "R_local: 31") || !strings.Contains(ctx.RootCauseEntity, "R_final: 48") {
		t.Errorf("根因实体应为 0-100 刻度：%s", ctx.RootCauseEntity)
	}
	if !strings.Contains(ctx.AffectedEntities, "R=20") {
		t.Errorf("受影响实体应为 0-100 刻度：%s", ctx.AffectedEntities)
	}
	for _, s := range []string{ctx.IncidentSummary, ctx.RootCauseEntity, ctx.AffectedEntities} {
		if strings.Contains(s, "0.48") || strings.Contains(s, "峰值风险: 0 ") {
			t.Errorf("不得出现 [0,1] 刻度的值：%s", s)
		}
	}
}

// A2：相似历史事件按根因实体查询，必须排除当前事件本身（曾把自己当成「相同 ID 的历史事件」）
func TestExcludeIncident(t *testing.T) {
	list := []*database.AIOpsIncident{{ID: "inc-cur"}, {ID: "inc-old-001"}, {ID: "inc-old-002"}}
	got := excludeIncident(list, "inc-cur")
	if len(got) != 2 || got[0].ID != "inc-old-001" || got[1].ID != "inc-old-002" {
		t.Errorf("got %+v", got)
	}
	if got := excludeIncident(nil, "x"); len(got) != 0 {
		t.Errorf("nil → 空，得到 %+v", got)
	}
}
