package aiops

import (
	"testing"

	"AtlHyper/atlhyper_master_v2/aiops"
)

// 峰值风险的等级在出接口时按 0–1 的原值分档（与实体风险同一分档），分数再换成百分制
func TestScaleIncidents_PeakRiskLevel(t *testing.T) {
	got := ScaleIncidents([]*aiops.Incident{{ID: "a", PeakRisk: 0.85}, {ID: "b", PeakRisk: 0.3}})
	if got[0].PeakRisk != 85 || got[0].PeakRiskLevel != aiops.RiskLevel(0.85) || got[0].PeakRiskLevel != "critical" {
		t.Errorf("a = %v / %s, want 85 / critical", got[0].PeakRisk, got[0].PeakRiskLevel)
	}
	if got[1].PeakRiskLevel != "low" {
		t.Errorf("b level = %s, want low", got[1].PeakRiskLevel)
	}

	detail := ScaleIncidentDetail(&aiops.IncidentDetail{Incident: aiops.Incident{PeakRisk: 0.65}})
	if detail.PeakRisk != 65 || detail.PeakRiskLevel != "high" {
		t.Errorf("detail = %v / %s, want 65 / high", detail.PeakRisk, detail.PeakRiskLevel)
	}
}
